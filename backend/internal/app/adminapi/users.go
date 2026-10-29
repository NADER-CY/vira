package adminapi

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NADER-CY/vira/backend/internal/modules/identity"
	"github.com/NADER-CY/vira/backend/internal/modules/moderation"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Phase 2 staff surfaces: user management and the report queue (doc 09 §4.2 "Users & Trust", "Content").

func (a *api) routesP2(r *httpx.Router) {
	authn, perm := a.requireSession, a.requirePermission
	r.HandleFunc(http.MethodGet, "/v1/admin/users", a.listUsers, authn, perm("users.read"))
	r.HandleFunc(http.MethodGet, "/v1/admin/users/{id}", a.getUser, authn, perm("users.read"))
	r.HandleFunc(http.MethodPost, "/v1/admin/users/{id}/status", a.setUserStatus, authn, perm("users.write"))
	r.HandleFunc(http.MethodPost, "/v1/admin/users/{id}/sessions/revoke", a.revokeUserSessions, authn, perm("users.write"))
	r.HandleFunc(http.MethodGet, "/v1/admin/reports", a.listReports, authn, perm("reports.read"))
	r.HandleFunc(http.MethodPost, "/v1/admin/reports/{id}/resolve", a.resolveReport, authn, perm("reports.write"))
}

func cursorOf(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func parseCursor(c string) (*time.Time, string, error) {
	if c == "" {
		return nil, "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(c)
	ts, id, ok := strings.Cut(string(raw), "|")
	if err != nil || !ok || !ids.Valid(id) {
		return nil, "", apperr.Validation(apperr.FieldError{Field: "cursor", Code: "invalid"})
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return nil, "", apperr.Validation(apperr.FieldError{Field: "cursor", Code: "invalid"})
	}
	return &t, id, nil
}

func limitOf(r *http.Request) (int, error) {
	l := r.URL.Query().Get("limit")
	if l == "" {
		return 50, nil
	}
	n, err := strconv.Atoi(l)
	if err != nil || n < 1 || n > 100 {
		return 0, apperr.Validation(apperr.FieldError{Field: "limit", Code: "out_of_range"})
	}
	return n, nil
}

// auditView records that staff viewed personal data (one event per request, doc 06 §8).
func (a *api) auditView(r *http.Request, tx db.Tx, action, targetType, targetID string, md map[string]any) error {
	ad, m := adminFrom(r.Context()), meta(r)
	_, err := audit.Record(r.Context(), tx, audit.Event{ActorType: "admin", ActorID: ad.ID, Action: action, TargetType: targetType,
		TargetID: targetID, IP: m.IP, UserAgent: m.UA, TraceID: m.TraceID, Metadata: md})
	return err
}

func (a *api) listUsers(w http.ResponseWriter, r *http.Request) {
	n, err := limitOf(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	before, beforeID, err := parseCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	q := identity.AdminQuery{Q: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), Before: before, BeforeID: beforeID, Limit: n}
	list, err := a.d.Users.ListUsers(r.Context(), q)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	err = db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: adminFrom(r.Context()).ID}, db.TxOptions{}, func(tx db.Tx) error {
		// The search term may be a phone number: record only whether one was given.
		return a.auditView(r, tx, "users.list", "", "", map[string]any{"has_query": q.Q != "", "status": q.Status, "rows": len(list)})
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var next any
	if len(list) == n {
		last := list[len(list)-1]
		next = cursorOf(last.CreatedAt, last.ID)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": list, "next_cursor": next})
}

func (a *api) getUser(w http.ResponseWriter, r *http.Request) {
	u, err := a.d.Users.User(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	err = db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: adminFrom(r.Context()).ID}, db.TxOptions{}, func(tx db.Tx) error {
		return a.auditView(r, tx, "users.view", "user", u.ID, nil)
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

func (a *api) setUserStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ad := adminFrom(r.Context())
	id := r.PathValue("id")
	var prev string
	err := db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		if prev, err = a.d.Users.SetStatus(r.Context(), tx, id, in.Status, in.Reason, ad.ID); err != nil {
			return err
		}
		return a.auditView(r, tx, "users.status_changed", "user", id, map[string]any{"from": prev, "to": in.Status, "reason": in.Reason})
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"id": id, "previous_status": prev, "status": in.Status})
}

func (a *api) revokeUserSessions(w http.ResponseWriter, r *http.Request) {
	ad := adminFrom(r.Context())
	id := r.PathValue("id")
	if !ids.Valid(id) {
		httpx.WriteError(w, r, apperr.New(apperr.NotFound, ""))
		return
	}
	var n int64
	err := db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		if n, err = a.d.Users.RevokeSessions(r.Context(), tx, id); err != nil {
			return err
		}
		return a.auditView(r, tx, "users.sessions_revoked", "user", id, map[string]any{"sessions": n})
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"revoked": n})
}

func (a *api) listReports(w http.ResponseWriter, r *http.Request) {
	n, err := limitOf(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	before, beforeID, err := parseCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	state, tt := r.URL.Query().Get("state"), r.URL.Query().Get("target_type")
	if state != "" && state != "open" && state != "actioned" && state != "dismissed" {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "state", Code: "invalid"}))
		return
	}
	if tt != "" && tt != "post" && tt != "comment" && tt != "user" {
		httpx.WriteError(w, r, apperr.Validation(apperr.FieldError{Field: "target_type", Code: "invalid"}))
		return
	}
	list, err := a.d.Moderation.List(r.Context(), moderation.Query{State: state, TargetType: tt, Before: before, BeforeID: beforeID, Limit: n})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	err = db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: adminFrom(r.Context()).ID}, db.TxOptions{}, func(tx db.Tx) error {
		return a.auditView(r, tx, "reports.list", "", "", map[string]any{"state": state, "rows": len(list)})
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var next any
	if len(list) == n {
		last := list[len(list)-1]
		next = cursorOf(last.CreatedAt, last.ID)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": list, "next_cursor": next})
}

func (a *api) resolveReport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Resolution string `json:"resolution"`
		Note       string `json:"note"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ad := adminFrom(r.Context())
	var rep moderation.Report
	err := db.InTx(r.Context(), a.d.Pool, db.Scope{Audience: db.AudienceAdmin, ActorID: ad.ID}, db.TxOptions{}, func(tx db.Tx) error {
		var err error
		if rep, err = a.d.Moderation.Resolve(r.Context(), tx, r.PathValue("id"), in.Resolution, in.Note, ad.ID); err != nil {
			return err
		}
		return a.auditView(r, tx, "reports.resolved", "report", rep.ID, map[string]any{"resolution": in.Resolution,
			"target_type": rep.TargetType, "target_id": rep.TargetID})
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rep)
}
