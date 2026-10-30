package adminapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/NADER-CY/vira/backend/internal/modules/identity"
	idc "github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	"github.com/NADER-CY/vira/backend/internal/modules/moderation"
	"github.com/NADER-CY/vira/backend/internal/modules/social"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/ratelimit"
	"github.com/NADER-CY/vira/backend/internal/platform/sms"
)

func fill(b byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = b
	}
	return out
}

var testPII = idc.PII{EncKey: fill(7), IndexKey: fill(8)}

// appUsers creates real accounts through the identity service (public realm) for the admin tests.
type appUsers struct {
	id   *identity.Service
	soc  *social.Service
	mod  *moderation.Service
	sms  *sms.Mock
	pass int
}

func (e *env) appUsers(t *testing.T) *appUsers {
	mr := miniredis.RunT(t)
	lim := &ratelimit.Limiter{Client: redis.NewClient(&redis.Options{Addr: mr.Addr()}), Prefix: "a:"}
	k, _ := auth.GenerateKey()
	signer := &auth.Signer{Key: k, KeyID: auth.KeyID(&k.PublicKey), Issuer: "vira", Audience: "vira-api", TTL: time.Minute}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	pool := e.d.Pool("app_public")
	m := &sms.Mock{}
	return &appUsers{id: &identity.Service{Pool: pool, Signer: signer, Limiter: lim, SMS: m, PII: testPII, Log: log},
		soc: &social.Service{Pool: pool, Log: log}, mod: &moderation.Service{Pool: pool, Log: log}, sms: m}
}

var six = regexp.MustCompile(`[0-9]{6}`)

func (a *appUsers) signUp(t *testing.T, phone string) identity.SignIn {
	t.Helper()
	ctx := context.Background()
	ch, err := a.id.RequestOTP(ctx, phone, identity.RequestMeta{IP: "10.0.0.1", DeviceID: "dev-" + phone})
	if err != nil {
		t.Fatal(err)
	}
	msg, _ := a.sms.Last(phone)
	in, err := a.id.VerifyOTP(ctx, ch.ChallengeID, six.FindString(msg.Text), identity.DeviceInfo{ID: "device-" + phone, Platform: "android"}, false,
		identity.Consents{Terms: true, Privacy: true, AgeConfirmation: true, TextVersion: "t1"}, identity.RequestMeta{IP: "10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func activeSessions(t *testing.T, e *env, userID string) int {
	var n int
	if err := e.d.Pool("superuser").QueryRow(context.Background(), `SELECT count(*) FROM identity.sessions WHERE user_id = $1 AND revoked_at IS NULL`, userID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUserManagement(t *testing.T) {
	e := newEnv(t)
	app := e.appUsers(t)
	u1 := app.signUp(t, "09121230001")
	u2 := app.signUp(t, "09121230002")

	enr := e.createAdmin(t, "mod@vira.test", "moderator")
	c := e.client(t)
	e.login(t, c, "mod@vira.test", enr, 0)

	// List: newest first, masked phones only.
	st, list, _ := c.do("GET", "/v1/admin/users", nil, nil)
	data := list["data"].([]any)
	if st != 200 || len(data) != 2 || data[0].(map[string]any)["id"] != u2.UserID {
		t.Fatalf("list: %d %v", st, list)
	}
	if data[0].(map[string]any)["phone_masked"] != "09*****0002" {
		t.Fatalf("masked phone: %v", data[0])
	}
	// Search by exact phone (blind index), by handle prefix and by id; status filter.
	for q, want := range map[string]string{"۰۹۱۲۱۲۳۰۰۰۱": u1.UserID, "+989121230002": u2.UserID, u1.Handle[:9]: "", u2.UserID: u2.UserID} {
		_, res, _ := c.do("GET", "/v1/admin/users?q="+url.QueryEscape(q), nil, nil)
		rows := res["data"].([]any)
		if want != "" && (len(rows) != 1 || rows[0].(map[string]any)["id"] != want) {
			t.Fatalf("search %q: %v", q, res)
		}
	}
	if _, res, _ := c.do("GET", "/v1/admin/users?status=suspended", nil, nil); len(res["data"].([]any)) != 0 {
		t.Fatalf("status filter: %v", res)
	}
	if st, _, _ := c.do("GET", "/v1/admin/users?status=nope", nil, nil); st != 422 {
		t.Fatalf("bad status filter: %d", st)
	}

	// Suspend: requires a reason, revokes sessions, is visible in history and the audit log.
	if st, _, _ := c.do("POST", "/v1/admin/users/"+u1.UserID+"/status", map[string]string{"status": "suspended"}, nil); st != 422 {
		t.Fatalf("missing reason: %d", st)
	}
	if st, _, _ := c.do("POST", "/v1/admin/users/"+u1.UserID+"/status", map[string]string{"status": "deleted", "reason": "x"}, nil); st != 422 {
		t.Fatalf("staff cannot delete accounts: %d", st)
	}
	if activeSessions(t, e, u1.UserID) != 1 {
		t.Fatal("expected an active session before suspension")
	}
	st, res, _ := c.do("POST", "/v1/admin/users/"+u1.UserID+"/status", map[string]string{"status": "suspended", "reason": "کلاهبرداری تأییدشده"}, nil)
	if st != 200 || res["previous_status"] != "active" {
		t.Fatalf("suspend: %d %v", st, res)
	}
	if activeSessions(t, e, u1.UserID) != 0 {
		t.Fatal("suspension must revoke sessions")
	}
	if st, _, _ := c.do("POST", "/v1/admin/users/"+u1.UserID+"/status", map[string]string{"status": "suspended", "reason": "again"}, nil); st != 409 {
		t.Fatalf("unchanged status: %d", st)
	}
	_, det, _ := c.do("GET", "/v1/admin/users/"+u1.UserID, nil, nil)
	if det["status"] != "suspended" || len(det["status_history"].([]any)) != 1 || len(det["sessions"].([]any)) != 1 {
		t.Fatalf("detail: %v", det)
	}
	// Revoke sessions of the other user.
	if st, r, _ := c.do("POST", "/v1/admin/users/"+u2.UserID+"/sessions/revoke", nil, nil); st != 200 || r["revoked"] != float64(1) {
		t.Fatalf("revoke: %d %v", st, r)
	}
	// Summary: counts only, available to readers of the user list.
	if st, sum, _ := c.do("GET", "/v1/admin/users/summary", nil, nil); st != 200 || sum["total"] != float64(2) || sum["suspended"] != float64(1) ||
		sum["active"] != float64(1) || sum["signups_today"] != float64(2) || sum["active_sessions"] != float64(0) {
		t.Fatalf("summary: %d %v", st, sum)
	}
	var target string
	var nEv int
	_ = e.d.Pool("superuser").QueryRow(context.Background(), `SELECT count(*), max(target_id) FROM platform.audit_events WHERE action = 'users.status_changed'`).Scan(&nEv, &target)
	if nEv != 1 || target != u1.UserID {
		t.Fatalf("audit: %d %s", nEv, target)
	}

	// An auditor can read but not change.
	enrA := e.createAdmin(t, "aud@vira.test", "auditor")
	ca := e.client(t)
	e.login(t, ca, "aud@vira.test", enrA, 0)
	if st, _, _ := ca.do("GET", "/v1/admin/users", nil, nil); st != 200 {
		t.Fatalf("auditor read: %d", st)
	}
	if st, _, _ := ca.do("POST", "/v1/admin/users/"+u2.UserID+"/status", map[string]string{"status": "restricted", "reason": "x"}, nil); st != 403 {
		t.Fatalf("auditor write: %d", st)
	}
	// Operators cannot see reports.
	enrO := e.createAdmin(t, "op@vira.test", "operator")
	co := e.client(t)
	e.login(t, co, "op@vira.test", enrO, 0)
	if st, _, _ := co.do("GET", "/v1/admin/reports/summary", nil, nil); st != 403 {
		t.Fatalf("operator reports summary: %d", st)
	}
	if st, _, _ := co.do("GET", "/v1/admin/reports", nil, nil); st != 403 {
		t.Fatalf("operator reports: %d", st)
	}
}

func TestReportReview(t *testing.T) {
	e := newEnv(t)
	app := e.appUsers(t)
	author := app.signUp(t, "09121230011")
	r1 := app.signUp(t, "09121230012")
	r2 := app.signUp(t, "09121230013")
	ctx := idc.WithAccountStatus(context.Background(), idc.StatusActive)
	post, err := app.soc.CreatePost(ctx, author.UserID, social.PostInput{Caption: "فروش کالای تقلبی"}, social.Meta{})
	if err != nil {
		t.Fatal(err)
	}
	rep1, err := app.mod.File(ctx, r1.UserID, "post", post.ID, "counterfeit", "", audit.Event{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.mod.File(ctx, r2.UserID, "post", post.ID, "scam", "", audit.Event{}); err != nil {
		t.Fatal(err)
	}
	other, _ := app.soc.CreatePost(ctx, author.UserID, social.PostInput{Caption: "پست عادی"}, social.Meta{})
	rep3, _ := app.mod.File(ctx, r1.UserID, "post", other.ID, "spam", "", audit.Event{})

	enr := e.createAdmin(t, "mod@vira.test", "moderator")
	c := e.client(t)
	e.login(t, c, "mod@vira.test", enr, 0)
	st, q, _ := c.do("GET", "/v1/admin/reports?state=open", nil, nil)
	if st != 200 || len(q["data"].([]any)) != 3 {
		t.Fatalf("queue: %d %v", st, q)
	}
	first := q["data"].([]any)[2].(map[string]any)
	if first["target_preview"] != "فروش کالای تقلبی" || first["open_reports_on_target"] != float64(2) || first["target_owner_handle"] != author.Handle {
		t.Fatalf("report row: %v", first)
	}
	if st, _, _ := c.do("POST", "/v1/admin/reports/"+rep1.ID+"/resolve", map[string]string{"resolution": "remove_content"}, nil); st != 422 {
		t.Fatalf("note required: %d", st)
	}
	st, done, _ := c.do("POST", "/v1/admin/reports/"+rep1.ID+"/resolve", map[string]string{"resolution": "remove_content", "note": "کالای تقلبی"}, nil)
	if st != 200 || done["state"] != "actioned" {
		t.Fatalf("resolve: %d %v", st, done)
	}
	// Both reports on the post are closed; the post is no longer visible; resolving again conflicts.
	if _, q, _ = c.do("GET", "/v1/admin/reports?state=open", nil, nil); len(q["data"].([]any)) != 1 {
		t.Fatalf("open after resolve: %v", q)
	}
	if _, err := app.soc.GetPost(ctx, r1.UserID, post.ID); err == nil {
		t.Fatal("removed post still visible")
	}
	if st, _, _ := c.do("POST", "/v1/admin/reports/"+rep1.ID+"/resolve", map[string]string{"resolution": "dismiss", "note": "x"}, nil); st != 409 {
		t.Fatalf("double resolve: %d", st)
	}
	// Dismissal leaves the content and closes only that report.
	if st, d, _ := c.do("POST", "/v1/admin/reports/"+rep3.ID+"/resolve", map[string]string{"resolution": "dismiss", "note": "بدون تخلف"}, nil); st != 200 || d["state"] != "dismissed" {
		t.Fatalf("dismiss: %d %v", st, d)
	}
	if _, err := app.soc.GetPost(ctx, r1.UserID, other.ID); err != nil {
		t.Fatalf("dismissed report removed content: %v", err)
	}
	// Restrict the author through a new report.
	rep4, _ := app.mod.File(ctx, r2.UserID, "user", author.UserID, "harassment", "", audit.Event{})
	if st, _, _ := c.do("POST", "/v1/admin/reports/"+rep4.ID+"/resolve", map[string]string{"resolution": "restrict_user", "note": "هشدار"}, nil); st != 200 {
		t.Fatalf("restrict: %d", st)
	}
	if st, sum, _ := c.do("GET", "/v1/admin/reports/summary", nil, nil); st != 200 || sum["open"] != float64(0) || sum["actioned_today"] != float64(3) ||
		sum["dismissed_today"] != float64(1) || sum["total"] != float64(4) {
		t.Fatalf("reports summary: %d %v", st, sum)
	}
	var status string
	_ = e.d.Pool("superuser").QueryRow(context.Background(), `SELECT status FROM identity.users WHERE id = $1`, author.UserID).Scan(&status)
	if status != "restricted" {
		t.Fatalf("author status %s", status)
	}
}
