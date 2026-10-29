package publicapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---------- helpers ----------

type user struct {
	id, handle, access, refresh, session string
	f                                    *fixture
	t                                    *testing.T
}

func (u *user) auth() map[string]string {
	return map[string]string{"Authorization": "Bearer " + u.access}
}

var consents = map[string]any{"terms": true, "privacy": true, "age_confirmation": true, "text_version": "test-1"}

var codeRe = regexp.MustCompile(`[0-9]{6}`)

func (f *fixture) do(t *testing.T, method, path string, body any, hdr map[string]string) (int, map[string]any, http.Header) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, f.srv.URL+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	return resp.StatusCode, m, resp.Header
}

func (u *user) do(method, path string, body any, extra ...map[string]string) (int, map[string]any, http.Header) {
	u.t.Helper()
	h := u.auth()
	for _, e := range extra {
		for k, v := range e {
			h[k] = v
		}
	}
	return u.f.do(u.t, method, path, body, h)
}

func (f *fixture) requestCode(t *testing.T, phone string) (string, string) {
	t.Helper()
	st, b, _ := f.do(t, "POST", "/v1/auth/otp", map[string]any{"phone": phone}, map[string]string{"X-Device-Id": "device-" + phone})
	if st != 202 {
		t.Fatalf("otp request: %d %v", st, b)
	}
	msg, ok := f.sms.Last(normalize(phone))
	if !ok {
		t.Fatalf("no SMS sent to %s", phone)
	}
	return b["challenge_id"].(string), codeRe.FindString(msg.Text)
}

func normalize(p string) string {
	if strings.HasPrefix(p, "+98") {
		return "0" + p[3:]
	}
	return p
}

func device(phone string) map[string]any {
	return map[string]any{"id": "device-" + phone, "platform": "android", "model": "Pixel", "app_version": "0.2.0"}
}

func (f *fixture) signUp(t *testing.T, phone string) *user {
	t.Helper()
	ch, code := f.requestCode(t, phone)
	st, b, _ := f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch, "code": code, "device": device(phone), "consents": consents}, nil)
	if st != 200 {
		t.Fatalf("verify: %d %v", st, b)
	}
	return &user{id: b["user_id"].(string), handle: b["handle"].(string), access: b["access_token"].(string),
		refresh: b["refresh_token"].(string), session: b["session_id"].(string), f: f, t: t}
}

func (u *user) version() string {
	u.t.Helper()
	_, _, h := u.do("GET", "/v1/me/profile", nil)
	return h.Get("ETag")
}

func (u *user) post(caption, visibility string, media ...string) string {
	u.t.Helper()
	if media == nil {
		media = []string{}
	}
	st, b, _ := u.do("POST", "/v1/posts", map[string]any{"caption": caption, "visibility": visibility, "media_ids": media})
	if st != 201 {
		u.t.Fatalf("create post: %d %v", st, b)
	}
	return b["id"].(string)
}

func (u *user) upload(purpose string) map[string]any {
	u.t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	for x := 0; x < 300; x++ {
		for y := 0; y < 200; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf, body bytes.Buffer
	_ = png.Encode(&buf, img)
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "photo.png")
	_, _ = fw.Write(buf.Bytes())
	_ = mw.WriteField("purpose", purpose)
	_ = mw.Close()
	req, _ := http.NewRequest("POST", u.f.srv.URL+"/v1/media/images", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+u.access)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		u.t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	if resp.StatusCode != 201 {
		u.t.Fatalf("upload: %d %v", resp.StatusCode, m)
	}
	return m
}

func ids(page map[string]any) []string {
	var out []string
	for _, it := range page["data"].([]any) {
		out = append(out, it.(map[string]any)["id"].(string))
	}
	return out
}

// ---------- identity: OTP, sessions, refresh ----------

func TestOTPSignUpSignInAndSessions(t *testing.T) {
	f := setup(t)
	// Persian digits and +98 are accepted.
	ch, code := f.requestCode(t, "+989121112233")
	if len(code) != 6 {
		t.Fatalf("code length %d", len(code))
	}
	persian := strings.Map(func(r rune) rune { return '۰' + (r - '0') }, code)
	st, b, _ := f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch, "code": persian, "device": device("+989121112233"),
		"consents": consents, "intends_to_sell": true}, nil)
	if st != 200 || b["is_new"] != true || !strings.HasPrefix(b["handle"].(string), "user") {
		t.Fatalf("first sign-in: %d %v", st, b)
	}
	u := &user{id: b["user_id"].(string), access: b["access_token"].(string), refresh: b["refresh_token"].(string), f: f, t: t}

	// The account view shows the masked number, never the full one.
	st, acc, _ := u.do("GET", "/v1/me/account", nil)
	if st != 200 || acc["phone_masked"] != "09*****2233" || acc["intends_to_sell"] != true || acc["status"] != "active" {
		t.Fatalf("account: %d %v", st, acc)
	}
	var stored []byte
	if err := f.d.Pool("superuser").QueryRow(context.Background(), `SELECT phone_enc FROM identity.phone_identities WHERE user_id = $1`, u.id).Scan(&stored); err != nil ||
		bytes.Contains(stored, []byte("09121112233")) {
		t.Fatalf("phone must be stored encrypted: %v", err)
	}

	// Second sign-in on another device: same account, not new; consents recorded once per version.
	ch2, code2 := f.requestCode(t, "09121112233")
	f.mr.FastForward(0)
	dev := device("09121112233")
	dev["id"] = "second-device-1"
	st, b2, _ := f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch2, "code": code2, "device": dev, "consents": consents}, nil)
	if st != 200 || b2["is_new"] != false || b2["user_id"] != u.id {
		t.Fatalf("second sign-in: %d %v", st, b2)
	}
	var nConsents int
	_ = f.d.Pool("superuser").QueryRow(context.Background(), `SELECT count(*) FROM identity.consents WHERE user_id = $1`, u.id).Scan(&nConsents)
	if nConsents != 3 {
		t.Fatalf("consents recorded %d times, want 3 (terms, privacy, age)", nConsents)
	}

	st, sess, _ := u.do("GET", "/v1/me/sessions", nil)
	if st != 200 || len(sess["data"].([]any)) != 2 {
		t.Fatalf("sessions: %d %v", st, sess)
	}
	// Sign out the other device from the first one; its token stops working at once.
	other := &user{access: b2["access_token"].(string), f: f, t: t}
	if st, _, _ := u.do("POST", "/v1/me/sessions/revoke-others", nil); st != 204 {
		t.Fatalf("revoke others: %d", st)
	}
	if st, _, _ := other.do("GET", "/v1/me/account", nil); st != 401 {
		t.Fatalf("revoked session still works: %d", st)
	}

	// Logout revokes the current session.
	if st, _, _ := u.do("POST", "/v1/auth/logout", nil); st != 204 {
		t.Fatalf("logout: %d", st)
	}
	if st, _, _ := u.do("GET", "/v1/me/account", nil); st != 401 {
		t.Fatalf("token after logout: %d", st)
	}
}

func TestRefreshRotationAndReplayRevokesSession(t *testing.T) {
	f := setup(t)
	u := f.signUp(t, "09120000010")
	st, r1, _ := f.do(t, "POST", "/v1/auth/token/refresh", map[string]any{"refresh_token": u.refresh}, nil)
	if st != 200 || r1["refresh_token"] == u.refresh || r1["session_id"] != u.session {
		t.Fatalf("refresh: %d %v", st, r1)
	}
	fresh := &user{access: r1["access_token"].(string), f: f, t: t}
	if st, _, _ := fresh.do("GET", "/v1/me/account", nil); st != 200 {
		t.Fatalf("rotated access token: %d", st)
	}
	// Presenting the old (rotated) refresh token again is a replay: refused, and the session is revoked.
	if st, _, _ := f.do(t, "POST", "/v1/auth/token/refresh", map[string]any{"refresh_token": u.refresh}, nil); st != 401 {
		t.Fatalf("replay accepted: %d", st)
	}
	if st, _, _ := fresh.do("GET", "/v1/me/account", nil); st != 401 {
		t.Fatalf("session must be revoked after replay: %d", st)
	}
	if st, _, _ := f.do(t, "POST", "/v1/auth/token/refresh", map[string]any{"refresh_token": r1["refresh_token"]}, nil); st != 401 {
		t.Fatalf("new token of a revoked session: %d", st)
	}
	var n int
	_ = f.d.Pool("superuser").QueryRow(context.Background(), `SELECT count(*) FROM platform.audit_events WHERE action = 'auth.refresh_replay_detected' AND actor_id = $1`, u.id).Scan(&n)
	if n != 1 {
		t.Fatalf("replay audit events: %d", n)
	}
}

func TestOTPAbusePrevention(t *testing.T) {
	f := setup(t)
	// Wrong codes: attempts remaining, then lock after 5.
	ch, code := f.requestCode(t, "09120000020")
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 1; i <= 5; i++ {
		st, b, _ := f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch, "code": wrong, "device": device("09120000020"), "consents": consents}, nil)
		if i < 5 && (st != 401 || b["code"] != "otp_invalid" || b["attempts_remaining"] != float64(5-i)) {
			t.Fatalf("attempt %d: %d %v", i, st, b)
		}
		if i == 5 && (st != 429 || b["code"] != "otp_locked") {
			t.Fatalf("lock: %d %v", st, b)
		}
	}
	// Even the right code no longer works on a locked challenge.
	if st, b, _ := f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch, "code": code, "device": device("09120000020"), "consents": consents}, nil); st != 429 {
		t.Fatalf("locked challenge accepted right code: %d %v", st, b)
	}

	// Resend cooldown: a second request within 60 s is refused with retry_after.
	f.requestCode(t, "09120000021")
	st, b, _ := f.do(t, "POST", "/v1/auth/otp", map[string]any{"phone": "09120000021"}, map[string]string{"X-Device-Id": "device-x"})
	if st != 429 || b["retry_after"] == nil {
		t.Fatalf("cooldown: %d %v", st, b)
	}

	// Per-number limit (3 per 10 min) is silent: the 4th request answers 202 but no SMS is sent.
	phone := "09120000022"
	for i := 0; i < 3; i++ {
		f.requestCode(t, phone)
		_, _ = f.d.Pool("superuser").Exec(context.Background(), `UPDATE identity.otp_challenges SET created_at = created_at - interval '2 minutes'`)
	}
	sent := f.sms.Count(phone)
	st, _, _ = f.do(t, "POST", "/v1/auth/otp", map[string]any{"phone": phone}, map[string]string{"X-Device-Id": "device-y"})
	if st != 202 || f.sms.Count(phone) != sent {
		t.Fatalf("per-number limit must be silent: %d, sms %d→%d", st, sent, f.sms.Count(phone))
	}

	// Invalid numbers and missing consents are validation errors.
	if st, _, _ := f.do(t, "POST", "/v1/auth/otp", map[string]any{"phone": "12345"}, nil); st != 422 {
		t.Fatalf("invalid phone: %d", st)
	}
	ch3, code3 := f.requestCode(t, "09120000023")
	st, b, _ = f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch3, "code": code3, "device": device("x09120000023"),
		"consents": map[string]any{"terms": true, "privacy": false, "age_confirmation": true, "text_version": "test-1"}}, nil)
	if st != 422 || !strings.Contains(fmt.Sprint(b["errors"]), "consents.privacy") {
		t.Fatalf("missing consent: %d %v", st, b)
	}

	// Per-IP limit: explicit 429 after 20 requests in 10 minutes.
	var last int
	for i := 0; i < 25; i++ {
		last, _, _ = f.do(t, "POST", "/v1/auth/otp", map[string]any{"phone": fmt.Sprintf("0912555%04d", i)}, map[string]string{"X-Device-Id": fmt.Sprintf("dev-ip-%02d", i)})
	}
	if last != 429 {
		t.Fatalf("per-IP limit: %d", last)
	}
}

func TestExpiredCode(t *testing.T) {
	f := setup(t)
	ch, code := f.requestCode(t, "09120000030")
	_, _ = f.d.Pool("superuser").Exec(context.Background(), `UPDATE identity.otp_challenges SET expires_at = now() - interval '1 second' WHERE id = $1`, ch)
	st, b, _ := f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch, "code": code, "device": device("09120000030"), "consents": consents}, nil)
	if st != 401 || b["code"] != "otp_expired" {
		t.Fatalf("expired: %d %v", st, b)
	}
}

func TestDeletionRequestAndCancelBySignIn(t *testing.T) {
	f := setup(t)
	u := f.signUp(t, "09120000040")
	st, b, _ := u.do("POST", "/v1/me/deletion", map[string]any{"confirm": true})
	if st != 202 || b["status"] != "deletion_pending" {
		t.Fatalf("deletion: %d %v", st, b)
	}
	if st, _, _ := u.do("GET", "/v1/me/account", nil); st != 401 {
		t.Fatalf("sessions must end on deletion request: %d", st)
	}
	f.mr.FastForward(2 * time.Minute)
	u2 := f.signUp(t, "09120000040")
	if _, acc, _ := u2.do("GET", "/v1/me/account", nil); acc["status"] != "active" {
		t.Fatalf("sign-in during grace period must cancel deletion: %v", acc)
	}
}

// ---------- profiles and privacy ----------

func TestProfileEditingWithConcurrencyAndValidation(t *testing.T) {
	f := setup(t)
	a := f.signUp(t, "09120000050")
	b := f.signUp(t, "09120000051")
	v := a.version()
	st, p, h := a.do("PATCH", "/v1/me/profile", map[string]any{"handle": "@Sara.Style", "display_name": "سارا", "bio": "طراح مد", "website": "https://example.ir"},
		map[string]string{"If-Match": v})
	if st != 200 || p["handle"] != "Sara.Style" || h.Get("ETag") == v {
		t.Fatalf("update: %d %v", st, p)
	}
	// Stale version → 412; missing If-Match → 428.
	if st, _, _ := a.do("PATCH", "/v1/me/profile", map[string]any{"bio": "x"}, map[string]string{"If-Match": v}); st != 412 {
		t.Fatalf("stale version: %d", st)
	}
	if st, _, _ := a.do("PATCH", "/v1/me/profile", map[string]any{"bio": "x"}); st != 428 {
		t.Fatalf("missing If-Match: %d", st)
	}
	// Handles are unique case-insensitively; reserved and malformed handles are refused.
	for handle, code := range map[string]string{"sara.style": "taken", "vira": "reserved", "ab": "invalid", "bad..dots": "invalid", "سلام": "invalid"} {
		st, e, _ := b.do("PATCH", "/v1/me/profile", map[string]any{"handle": handle}, map[string]string{"If-Match": b.version()})
		if st != 422 || !strings.Contains(fmt.Sprint(e["errors"]), code) {
			t.Fatalf("handle %q: %d %v", handle, st, e)
		}
	}
	st, pub, _ := b.do("GET", "/v1/handles/sara.style", nil)
	if st != 200 || pub["display_name"] != "سارا" || pub["privacy"] != nil || pub["version"] != nil {
		t.Fatalf("public view must not expose private settings: %d %v", st, pub)
	}
	// Avatar: own uploaded avatar only.
	av := a.upload("avatar")
	if st, _, _ := a.do("PATCH", "/v1/me/profile", map[string]any{"avatar_media_id": av["id"]}, map[string]string{"If-Match": a.version()}); st != 200 {
		t.Fatalf("set avatar: %d", st)
	}
	if st, _, _ := b.do("PATCH", "/v1/me/profile", map[string]any{"avatar_media_id": av["id"]}, map[string]string{"If-Match": b.version()}); st != 422 {
		t.Fatalf("someone else's image as avatar: %d", st)
	}
	if st, _, _ := b.do("GET", "/v1/media/"+av["id"].(string), nil); st != 200 {
		t.Fatalf("avatar should be visible: %d", st)
	}
}

func TestPrivateAccountFollowRequestsAndAccess(t *testing.T) {
	f := setup(t)
	owner := f.signUp(t, "09120000060")
	fan := f.signUp(t, "09120000061")
	stranger := f.signUp(t, "09120000062")
	if st, _, _ := owner.do("PUT", "/v1/me/privacy", map[string]any{"is_private": true, "comments_from": "followers", "hide_like_counts": true,
		"settings": map[string]any{"messages_from": "followers"}}); st != 200 {
		t.Fatalf("privacy: %d", st)
	}
	if st, e, _ := owner.do("PUT", "/v1/me/privacy", map[string]any{"is_private": true, "comments_from": "x", "hide_like_counts": false,
		"settings": map[string]any{"unknown": true}}); st != 422 || len(e["errors"].([]any)) != 2 {
		t.Fatalf("invalid privacy: %d %v", st, e)
	}
	pid := owner.post("فقط برای دنبال‌کنندگان", "public")

	// Before approval: profile visible (bio, counts), posts and lists are not.
	_, prof, _ := stranger.do("GET", "/v1/users/"+owner.id, nil)
	if prof["can_view_posts"] != false || prof["is_private"] != true {
		t.Fatalf("private profile: %v", prof)
	}
	if st, e, _ := stranger.do("GET", "/v1/users/"+owner.id+"/posts", nil); st != 403 || e["reason"] != "private_account" {
		t.Fatalf("private posts: %d %v", st, e)
	}
	if st, _, _ := stranger.do("GET", "/v1/posts/"+pid, nil); st != 404 {
		t.Fatalf("private post detail: %d", st)
	}
	if st, _, _ := stranger.do("GET", "/v1/users/"+owner.id+"/followers", nil); st != 403 {
		t.Fatalf("private followers: %d", st)
	}

	// Follow → request → approve.
	if _, r, _ := fan.do("POST", "/v1/users/"+owner.id+"/follow", nil); r["state"] != "requested" {
		t.Fatalf("follow private: %v", r)
	}
	_, reqs, _ := owner.do("GET", "/v1/me/follow-requests", nil)
	if got := ids(reqs); len(got) != 1 || got[0] != fan.id {
		t.Fatalf("requests: %v", reqs)
	}
	if st, _, _ := owner.do("POST", "/v1/me/follow-requests/"+fan.id+"/approve", nil); st != 204 {
		t.Fatalf("approve: %d", st)
	}
	if st, page, _ := fan.do("GET", "/v1/users/"+owner.id+"/posts", nil); st != 200 || len(page["data"].([]any)) != 1 {
		t.Fatalf("approved follower posts: %d %v", st, page)
	}
	// Like counts hidden from others, visible to the owner.
	_, p1, _ := fan.do("GET", "/v1/posts/"+pid, nil)
	_, p2, _ := owner.do("GET", "/v1/posts/"+pid, nil)
	if p1["like_count"] != nil || p2["like_count"] == nil {
		t.Fatalf("hide_like_counts: fan %v owner %v", p1["like_count"], p2["like_count"])
	}
	// Comments limited to followers: the stranger cannot comment, the fan can.
	if st, _, _ := stranger.do("POST", "/v1/posts/"+pid+"/comments", map[string]any{"body": "سلام"}); st != 404 {
		t.Fatalf("stranger comment on invisible post: %d", st)
	}
	if st, _, _ := fan.do("POST", "/v1/posts/"+pid+"/comments", map[string]any{"body": "عالی"}); st != 201 {
		t.Fatalf("fan comment: %d", st)
	}
	// Owner removes the follower → access ends.
	if st, _, _ := owner.do("DELETE", "/v1/me/followers/"+fan.id, nil); st != 204 {
		t.Fatalf("remove follower: %d", st)
	}
	if st, _, _ := fan.do("GET", "/v1/posts/"+pid, nil); st != 404 {
		t.Fatalf("removed follower still sees post: %d", st)
	}
	// Going public approves pending requests.
	_, _, _ = stranger.do("POST", "/v1/users/"+owner.id+"/follow", nil)
	_, _, _ = owner.do("PUT", "/v1/me/privacy", map[string]any{"is_private": false, "comments_from": "everyone", "hide_like_counts": false})
	if _, prof, _ := stranger.do("GET", "/v1/users/"+owner.id, nil); prof["relationship"].(map[string]any)["following"] != true {
		t.Fatalf("pending request should be approved when going public: %v", prof)
	}
}

func TestBlockingRules(t *testing.T) {
	f := setup(t)
	a := f.signUp(t, "09120000070")
	b := f.signUp(t, "09120000071")
	_, _, _ = b.do("POST", "/v1/users/"+a.id+"/follow", nil)
	_, _, _ = a.do("POST", "/v1/users/"+b.id+"/follow", nil)
	pa := a.post("پست الف", "public")
	pb := b.post("پست ب", "public")
	if st, _, _ := a.do("POST", "/v1/users/"+b.id+"/block", nil); st != 204 {
		t.Fatalf("block: %d", st)
	}
	// Follows removed both ways; neither sees the other's profile or posts; b cannot re-follow.
	if _, prof, _ := a.do("GET", "/v1/me/profile", nil); prof["stats"].(map[string]any)["followers"] != float64(0) {
		t.Fatalf("follows not removed: %v", prof["stats"])
	}
	if st, _, _ := b.do("GET", "/v1/users/"+a.id, nil); st != 404 {
		t.Fatalf("blocked user sees blocker profile: %d", st)
	}
	if st, _, _ := b.do("GET", "/v1/posts/"+pa, nil); st != 404 {
		t.Fatalf("blocked user sees blocker post: %d", st)
	}
	if st, _, _ := a.do("GET", "/v1/posts/"+pb, nil); st != 404 {
		t.Fatalf("blocker sees blocked post: %d", st)
	}
	if st, _, _ := b.do("POST", "/v1/users/"+a.id+"/follow", nil); st != 404 {
		t.Fatalf("blocked user can follow: %d", st)
	}
	if st, _, _ := b.do("POST", "/v1/posts/"+pa+"/comments", map[string]any{"body": "x"}); st != 404 {
		t.Fatalf("blocked user can comment: %d", st)
	}
	_, feed, _ := b.do("GET", "/v1/feed/home", nil)
	for _, id := range ids(feed) {
		if id == pa {
			t.Fatal("blocker post in blocked user's feed")
		}
	}
	_, search, _ := b.do("GET", "/v1/users/search?q="+a.handle[:5], nil)
	for _, it := range search["data"].([]any) {
		if it.(map[string]any)["id"] == a.id {
			t.Fatal("blocker appears in search")
		}
	}
	_, blocks, _ := a.do("GET", "/v1/me/blocks", nil)
	if len(blocks["data"].([]any)) != 1 {
		t.Fatalf("block list: %v", blocks)
	}
	if st, _, _ := a.do("DELETE", "/v1/users/"+b.id+"/block", nil); st != 204 {
		t.Fatalf("unblock: %d", st)
	}
	if st, _, _ := b.do("GET", "/v1/posts/"+pa, nil); st != 200 {
		t.Fatalf("after unblock: %d", st)
	}
}

// ---------- posts, feed, engagement ----------

func TestPostLifecycleAndOwnership(t *testing.T) {
	f := setup(t)
	a := f.signUp(t, "09120000080")
	b := f.signUp(t, "09120000081")
	img := a.upload("post")
	if img["width"] != float64(300) || img["height"] != float64(200) {
		t.Fatalf("image dims: %v", img)
	}
	// Idempotent create: retrying with the same key returns the same post.
	key := map[string]string{"Idempotency-Key": "create-post-1"}
	body := map[string]any{"caption": "کیف چرم دست‌دوز", "visibility": "public", "media_ids": []string{img["id"].(string)}}
	st1, p1, _ := a.do("POST", "/v1/posts", body, key)
	st2, p2, _ := a.do("POST", "/v1/posts", body, key)
	if st1 != 201 || st2 != 201 || p1["id"] != p2["id"] || len(p1["media"].([]any)) != 1 {
		t.Fatalf("idempotent create: %d %d %v %v", st1, st2, p1["id"], p2["id"])
	}
	id := p1["id"].(string)
	// The same image cannot be attached twice; someone else's image cannot be attached.
	if st, _, _ := a.do("POST", "/v1/posts", body); st != 422 {
		t.Fatalf("reused image: %d", st)
	}
	other := b.upload("post")
	if st, _, _ := a.do("POST", "/v1/posts", map[string]any{"caption": "x", "media_ids": []string{other["id"].(string)}}); st != 422 {
		t.Fatalf("foreign image: %d", st)
	}
	if st, _, _ := a.do("POST", "/v1/posts", map[string]any{"caption": "  "}); st != 422 {
		t.Fatalf("empty post: %d", st)
	}
	// Edit with version; other users cannot edit or delete.
	_, _, h := a.do("GET", "/v1/posts/"+id, nil)
	st, ed, _ := a.do("PATCH", "/v1/posts/"+id, map[string]any{"caption": "کیف چرم دست‌دوز — ویرایش", "visibility": "followers"}, map[string]string{"If-Match": h.Get("ETag")})
	if st != 200 || ed["edited_at"] == nil || ed["visibility"] != "followers" {
		t.Fatalf("edit: %d %v", st, ed)
	}
	if st, _, _ := b.do("GET", "/v1/posts/"+id, nil); st != 404 {
		t.Fatalf("followers-only post visible to non-follower: %d", st)
	}
	_, _, _ = b.do("POST", "/v1/users/"+a.id+"/follow", nil)
	_, _, h = b.do("GET", "/v1/posts/"+id, nil)
	if st, _, _ := b.do("PATCH", "/v1/posts/"+id, map[string]any{"caption": "hacked"}, map[string]string{"If-Match": h.Get("ETag")}); st != 403 {
		t.Fatalf("edit by other user: %d", st)
	}
	if st, _, _ := b.do("DELETE", "/v1/posts/"+id, nil); st != 403 {
		t.Fatalf("delete by other user: %d", st)
	}
	if st, _, _ := b.do("GET", "/v1/media/"+img["id"].(string), nil); st != 200 {
		t.Fatalf("follower fetches post image: %d", st)
	}
	if st, _, _ := a.do("DELETE", "/v1/posts/"+id, nil); st != 204 {
		t.Fatalf("delete: %d", st)
	}
	if st, _, _ := a.do("GET", "/v1/posts/"+id, nil); st != 404 {
		t.Fatalf("deleted post: %d", st)
	}
	if st, _, _ := b.do("GET", "/v1/media/"+img["id"].(string), nil); st != 404 {
		t.Fatalf("image of deleted post still visible: %d", st)
	}
}

func TestFeedPagination(t *testing.T) {
	f := setup(t)
	a := f.signUp(t, "09120000090")
	b := f.signUp(t, "09120000091")
	c := f.signUp(t, "09120000092")
	_, _, _ = b.do("POST", "/v1/users/"+a.id+"/follow", nil)
	want := map[string]bool{}
	for i := 0; i < 23; i++ {
		want[a.post("پست "+strconv.Itoa(i), "public")] = true
	}
	c.post("از کاربر دنبال‌نشده", "public")
	// Following feed of b: exactly a's 23 posts in pages of 10, newest first, no duplicates.
	seen := map[string]bool{}
	cur := ""
	pages := 0
	var lastTime string
	for {
		path := "/v1/feed/following?limit=10"
		if cur != "" {
			path += "&cursor=" + cur
		}
		st, page, _ := b.do("GET", path, nil)
		if st != 200 {
			t.Fatalf("page %d: %d %v", pages, st, page)
		}
		pages++
		for _, it := range page["data"].([]any) {
			p := it.(map[string]any)
			id := p["id"].(string)
			if seen[id] || !want[id] {
				t.Fatalf("unexpected or duplicate post %s", id)
			}
			if lastTime != "" && p["published_at"].(string) > lastTime {
				t.Fatal("not newest first")
			}
			lastTime = p["published_at"].(string)
			seen[id] = true
		}
		if page["next_cursor"] == nil {
			break
		}
		cur = page["next_cursor"].(string)
	}
	if len(seen) != 23 || pages != 3 {
		t.Fatalf("got %d posts in %d pages", len(seen), pages)
	}
	// Home feed includes public posts of non-followed accounts.
	_, home, _ := b.do("GET", "/v1/feed/home?limit=50", nil)
	if len(home["data"].([]any)) != 24 {
		t.Fatalf("home feed: %d posts", len(home["data"].([]any)))
	}
	if st, _, _ := b.do("GET", "/v1/feed/home?cursor=garbage", nil); st != 422 {
		t.Fatalf("bad cursor: %d", st)
	}
	// Empty feed for a new account.
	d := f.signUp(t, "09120000093")
	if _, empty, _ := d.do("GET", "/v1/feed/following", nil); len(empty["data"].([]any)) != 0 || empty["next_cursor"] != nil {
		t.Fatalf("empty feed: %v", empty)
	}
}

func TestLikesAndComments(t *testing.T) {
	f := setup(t)
	a := f.signUp(t, "09120000100")
	b := f.signUp(t, "09120000101")
	pid := a.post("سلام دنیا", "public")
	for i := 0; i < 2; i++ { // idempotent
		if st, l, _ := b.do("PUT", "/v1/posts/"+pid+"/like", nil); st != 200 || l["liked"] != true || l["like_count"] != float64(1) {
			t.Fatalf("like: %d %v", st, l)
		}
	}
	if _, p, _ := b.do("GET", "/v1/posts/"+pid, nil); p["viewer_liked"] != true || p["like_count"] != float64(1) {
		t.Fatalf("post after like: %v", p)
	}
	if _, l, _ := b.do("DELETE", "/v1/posts/"+pid+"/like", nil); l["liked"] != false || l["like_count"] != float64(0) {
		t.Fatalf("unlike: %v", l)
	}
	var cids []string
	for _, body := range []string{"اول", "دوم", "سوم"} {
		st, c, _ := b.do("POST", "/v1/posts/"+pid+"/comments", map[string]any{"body": body})
		if st != 201 || c["can_delete"] != true {
			t.Fatalf("comment: %d %v", st, c)
		}
		cids = append(cids, c["id"].(string))
	}
	_, ac, _ := a.do("POST", "/v1/posts/"+pid+"/comments", map[string]any{"body": "پاسخ نویسنده"})
	if ac["is_post_author"] != true {
		t.Fatalf("author badge: %v", ac)
	}
	_, list, _ := a.do("GET", "/v1/posts/"+pid+"/comments?limit=2", nil)
	if got := ids(list); len(got) != 2 || got[0] != cids[0] || list["next_cursor"] == nil {
		t.Fatalf("comments page: %v", list)
	}
	if _, p, _ := a.do("GET", "/v1/posts/"+pid, nil); p["comment_count"] != float64(4) {
		t.Fatalf("comment count: %v", p["comment_count"])
	}
	// b deletes own comment; a (post owner) deletes b's other comment; a third user cannot delete.
	c := f.signUp(t, "09120000102")
	if st, _, _ := c.do("DELETE", "/v1/comments/"+cids[2], nil); st != 403 {
		t.Fatalf("third party delete: %d", st)
	}
	if st, _, _ := b.do("DELETE", "/v1/comments/"+cids[0], nil); st != 204 {
		t.Fatalf("own delete: %d", st)
	}
	if st, _, _ := a.do("DELETE", "/v1/comments/"+cids[1], nil); st != 204 {
		t.Fatalf("owner delete: %d", st)
	}
	if _, p, _ := a.do("GET", "/v1/posts/"+pid, nil); p["comment_count"] != float64(2) {
		t.Fatalf("comment count after delete: %v", p["comment_count"])
	}
	if st, _, _ := b.do("POST", "/v1/posts/"+pid+"/comments", map[string]any{"body": strings.Repeat("ب", 1001)}); st != 422 {
		t.Fatalf("too long: %d", st)
	}
	// comments_from = nobody: only the author may comment.
	_, _, _ = a.do("PUT", "/v1/me/privacy", map[string]any{"is_private": false, "comments_from": "nobody", "hide_like_counts": false})
	if st, e, _ := b.do("POST", "/v1/posts/"+pid+"/comments", map[string]any{"body": "x"}); st != 403 || e["reason"] != "comments_limited" {
		t.Fatalf("comments limited: %d %v", st, e)
	}
}

// ---------- reports, account status ----------

func TestReportsAndRestrictedAccounts(t *testing.T) {
	f := setup(t)
	a := f.signUp(t, "09120000110")
	b := f.signUp(t, "09120000111")
	pid := a.post("محتوای مشکوک", "public")
	st, r1, _ := b.do("POST", "/v1/reports", map[string]any{"target_type": "post", "target_id": pid, "reason": "scam", "details": "کلاهبرداری"})
	_, r2, _ := b.do("POST", "/v1/reports", map[string]any{"target_type": "post", "target_id": pid, "reason": "scam"})
	if st != 201 || r1["id"] != r2["id"] {
		t.Fatalf("duplicate report must return the open one: %v %v", r1, r2)
	}
	if st, _, _ := a.do("POST", "/v1/reports", map[string]any{"target_type": "post", "target_id": pid, "reason": "spam"}); st != 422 {
		t.Fatalf("self report: %d", st)
	}
	if st, _, _ := b.do("POST", "/v1/reports", map[string]any{"target_type": "post", "target_id": pid, "reason": "nope"}); st != 422 {
		t.Fatalf("bad reason: %d", st)
	}
	// A restricted account can read but not write.
	_, _ = f.d.Pool("superuser").Exec(context.Background(), `UPDATE identity.users SET status = 'restricted' WHERE id = $1`, b.id)
	if st, _, _ := b.do("GET", "/v1/feed/home", nil); st != 200 {
		t.Fatalf("restricted read: %d", st)
	}
	if st, e, _ := b.do("POST", "/v1/posts", map[string]any{"caption": "x"}); st != 403 || e["code"] != "account_restricted" {
		t.Fatalf("restricted write: %d %v", st, e)
	}
	if st, _, _ := b.do("POST", "/v1/users/"+a.id+"/follow", nil); st != 403 {
		t.Fatalf("restricted follow: %d", st)
	}
	// Suspended: the next request fails (session check), and sign-in is refused.
	_, _ = f.d.Pool("superuser").Exec(context.Background(), `UPDATE identity.users SET status = 'suspended' WHERE id = $1`, b.id)
	if st, _, _ := b.do("GET", "/v1/feed/home", nil); st != 401 {
		t.Fatalf("suspended session: %d", st)
	}
	f.mr.FastForward(2 * time.Minute)
	ch, code := f.requestCode(t, "09120000111")
	if st, e, _ := f.do(t, "POST", "/v1/auth/otp/verify", map[string]any{"challenge_id": ch, "code": code, "device": device("09120000111"), "consents": consents}, nil); st != 403 || e["code"] != "account_restricted" {
		t.Fatalf("suspended sign-in: %d %v", st, e)
	}
}

func TestEveryWriteIsAudited(t *testing.T) {
	f := setup(t)
	a := f.signUp(t, "09120000120")
	b := f.signUp(t, "09120000121")
	_, _, _ = a.do("POST", "/v1/users/"+b.id+"/block", nil)
	pid := a.post("x", "public")
	_, _, _ = a.do("DELETE", "/v1/posts/"+pid, nil)
	rows, err := f.d.Pool("superuser").Query(context.Background(), `SELECT DISTINCT action FROM platform.audit_events WHERE actor_id = $1`, a.id)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		got[s] = true
	}
	rows.Close()
	for _, want := range []string{"account.registered", "auth.login", "user.blocked", "post.deleted"} {
		if !got[want] {
			t.Errorf("missing audit event %s (have %v)", want, got)
		}
	}
	// Audit records never contain the phone number.
	var leak bool
	_ = f.d.Pool("superuser").QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM platform.audit_events WHERE metadata::text LIKE '%09120000120%')`).Scan(&leak)
	if leak {
		t.Fatal("phone number found in audit metadata")
	}
}
