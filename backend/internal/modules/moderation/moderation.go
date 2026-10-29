// Package moderation is the moderation module (P2): user reports and moderation decisions (doc 06 §4.2,
// screens _213/_210 "report violation"). Users file reports through the public API; staff review and resolve
// them in the Super Admin (admin realm). Decisions act through the identity and social contracts.
package moderation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	idc "github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	socialc "github.com/NADER-CY/vira/backend/internal/modules/social/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/ratelimit"
)

// RuleReports limits reports per user.
var RuleReports = ratelimit.Rule{Name: "reports", Limit: 20, Window: time.Hour}

// Reasons accepted for a report (the categories of the report screens).
var Reasons = []string{"spam", "harassment", "hate", "violence", "nudity", "scam", "counterfeit", "misinformation", "self_harm", "other"}

// Resolutions staff can apply.
var Resolutions = []string{"dismiss", "remove_content", "restrict_user", "suspend_user"}

// Service files and resolves reports.
type Service struct {
	Pool    *db.Pool
	Limiter *ratelimit.Limiter
	Log     *slog.Logger
}

// Report is a moderation report.
type Report struct {
	ID             string     `json:"id"`
	ReporterID     string     `json:"reporter_id,omitempty"`
	TargetType     string     `json:"target_type"`
	TargetID       string     `json:"target_id"`
	TargetOwnerID  string     `json:"target_owner_id,omitempty"`
	Reason         string     `json:"reason"`
	Details        string     `json:"details,omitempty"`
	State          string     `json:"state"`
	Resolution     string     `json:"resolution,omitempty"`
	ResolutionNote string     `json:"resolution_note,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	// Admin views only.
	ReporterHandle string `json:"reporter_handle,omitempty"`
	OwnerHandle    string `json:"target_owner_handle,omitempty"`
	TargetPreview  string `json:"target_preview,omitempty"`
	OpenReports    int    `json:"open_reports_on_target,omitempty"`
}

func valid(v string, set []string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// File creates a report. A second open report by the same user on the same target returns the existing one.
func (s *Service) File(ctx context.Context, reporter, targetType, targetID, reason, details string, m audit.Event) (Report, error) {
	details = strings.TrimSpace(details)
	var fe []apperr.FieldError
	if !valid(targetType, []string{"post", "comment", "user"}) {
		fe = append(fe, apperr.FieldError{Field: "target_type", Code: "invalid"})
	}
	if !ids.Valid(targetID) {
		fe = append(fe, apperr.FieldError{Field: "target_id", Code: "invalid"})
	}
	if !valid(reason, Reasons) {
		fe = append(fe, apperr.FieldError{Field: "reason", Code: "invalid"})
	}
	if utf8.RuneCountInString(details) > 1000 {
		fe = append(fe, apperr.FieldError{Field: "details", Code: "too_long"})
	}
	if fe != nil {
		return Report{}, apperr.Validation(fe...)
	}
	if s.Limiter != nil {
		if err := s.Limiter.Check(ctx, false, ratelimit.Keyed{Rule: RuleReports, Key: reporter}); err != nil {
			return Report{}, err
		}
	}
	var out Report
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: reporter}, db.TxOptions{}, func(tx db.Tx) error {
		owner, err := socialc.ContentOwner(ctx, tx, targetType, targetID)
		if err != nil {
			return apperr.New(apperr.NotFound, "")
		}
		if owner == reporter {
			return apperr.Validation(apperr.FieldError{Field: "target_id", Code: "self"})
		}
		err = tx.QueryRow(ctx, `SELECT id, created_at, state FROM moderation.reports WHERE reporter_id = $1 AND target_type = $2
			AND target_id = $3 AND state = 'open'`, reporter, targetType, targetID).Scan(&out.ID, &out.CreatedAt, &out.State)
		if err == nil {
			out.TargetType, out.TargetID, out.Reason = targetType, targetID, reason
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		out = Report{ID: ids.New(), TargetType: targetType, TargetID: targetID, Reason: reason, Details: details, State: "open"}
		if err := tx.QueryRow(ctx, `INSERT INTO moderation.reports (id, reporter_id, target_type, target_id, target_owner_id, reason, details)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`, out.ID, reporter, targetType, targetID, owner, reason, details).
			Scan(&out.CreatedAt); err != nil {
			if db.IsUniqueViolation(err) {
				return apperr.New(apperr.Conflict, "report already filed")
			}
			return err
		}
		m.ActorType, m.ActorID, m.Action, m.TargetType, m.TargetID, m.Outcome = "user", reporter, "report.filed", "report", out.ID, audit.Success
		m.Metadata = map[string]any{"target_type": targetType, "reason": reason}
		_, err = audit.Record(ctx, tx, m)
		return err
	})
	return out, err
}

// Routes registers the user-facing report endpoint.
func (s *Service) Routes(r *httpx.Router, session httpx.Middleware) {
	r.HandleFunc(http.MethodPost, "/v1/reports", s.hFile, session)
}

func (s *Service) hFile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
		Reason     string `json:"reason"`
		Details    string `json:"details"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, _ := auth.FromContext(r.Context())
	rep, err := s.File(r.Context(), p.Subject, in.TargetType, in.TargetID, in.Reason, in.Details,
		audit.Event{UserAgent: r.UserAgent(), TraceID: httpx.TraceID(r.Context())})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": rep.ID, "state": rep.State, "created_at": rep.CreatedAt})
}

// ---------- staff (admin realm; called by the admin API with the app_admin pool) ----------

// Query filters the report queue.
type Query struct {
	State      string
	TargetType string
	Before     *time.Time
	BeforeID   string
	Limit      int
}

// List returns reports, newest first, with handles and a short preview of the reported content.
func (s *Service) List(ctx context.Context, q Query) ([]Report, error) {
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}
	var before any
	if q.Before != nil {
		before = *q.Before
	}
	var beforeID any
	if q.BeforeID != "" {
		beforeID = q.BeforeID
	}
	rows, err := s.Pool.Query(ctx, `SELECT r.id, r.reporter_id, r.target_type, r.target_id, COALESCE(r.target_owner_id::text, ''),
		r.reason, r.details, r.state, COALESCE(r.resolution, ''), COALESCE(r.resolution_note, ''), r.created_at, r.resolved_at,
		COALESCE(rp.handle, ''), COALESCE(op.handle, ''),
		COALESCE(CASE r.target_type
		  WHEN 'post' THEN (SELECT left(caption, 140) FROM social.posts WHERE id = r.target_id)
		  WHEN 'comment' THEN (SELECT left(body, 140) FROM social.comments WHERE id = r.target_id)
		  ELSE (SELECT '@' || handle FROM social.profiles WHERE user_id = r.target_id) END, ''),
		(SELECT count(*) FROM moderation.reports x WHERE x.target_type = r.target_type AND x.target_id = r.target_id AND x.state = 'open')
		FROM moderation.reports r
		LEFT JOIN social.profiles rp ON rp.user_id = r.reporter_id
		LEFT JOIN social.profiles op ON op.user_id = r.target_owner_id
		WHERE ($1 = '' OR r.state = $1) AND ($2 = '' OR r.target_type = $2)
		  AND ($3::timestamptz IS NULL OR (r.created_at, r.id) < ($3, $4::uuid))
		ORDER BY r.created_at DESC, r.id DESC LIMIT $5`, q.State, q.TargetType, before, beforeID, q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Report{}
	for rows.Next() {
		var x Report
		if err := rows.Scan(&x.ID, &x.ReporterID, &x.TargetType, &x.TargetID, &x.TargetOwnerID, &x.Reason, &x.Details, &x.State,
			&x.Resolution, &x.ResolutionNote, &x.CreatedAt, &x.ResolvedAt, &x.ReporterHandle, &x.OwnerHandle, &x.TargetPreview, &x.OpenReports); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ErrResolved is returned when a report was already resolved.
var ErrResolved = errors.New("moderation: report already resolved")

// Resolve applies a staff decision to a report. Content removal and user restrictions also close every other open
// report on the same target. The caller records the admin audit event in the same transaction.
func (s *Service) Resolve(ctx context.Context, tx db.Tx, reportID, resolution, note, adminID string) (Report, error) {
	if !ids.Valid(reportID) {
		return Report{}, apperr.New(apperr.NotFound, "")
	}
	note = strings.TrimSpace(note)
	var fe []apperr.FieldError
	if !valid(resolution, Resolutions) {
		fe = append(fe, apperr.FieldError{Field: "resolution", Code: "invalid"})
	}
	if l := utf8.RuneCountInString(note); l == 0 || l > 500 {
		fe = append(fe, apperr.FieldError{Field: "note", Code: "required", Detail: "1-500 characters"})
	}
	if fe != nil {
		return Report{}, apperr.Validation(fe...)
	}
	var r Report
	err := tx.QueryRow(ctx, `SELECT id, target_type, target_id, COALESCE(target_owner_id::text, ''), reason, state FROM moderation.reports
		WHERE id = $1 FOR UPDATE`, reportID).Scan(&r.ID, &r.TargetType, &r.TargetID, &r.TargetOwnerID, &r.Reason, &r.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, apperr.New(apperr.NotFound, "")
	}
	if err != nil {
		return Report{}, err
	}
	if r.State != "open" {
		return Report{}, apperr.New(apperr.Conflict, "report already resolved")
	}
	state := "actioned"
	switch resolution {
	case "dismiss":
		state = "dismissed"
	case "remove_content":
		if r.TargetType == "user" {
			return Report{}, apperr.Validation(apperr.FieldError{Field: "resolution", Code: "invalid", Detail: "a user report cannot remove content"})
		}
		if err := socialc.RemoveContent(ctx, tx, r.TargetType, r.TargetID, "moderation: "+r.Reason); err != nil {
			return Report{}, err
		}
	case "restrict_user", "suspend_user":
		status := idc.StatusRestricted
		if resolution == "suspend_user" {
			status = idc.StatusSuspended
		}
		if r.TargetOwnerID == "" {
			return Report{}, apperr.New(apperr.Conflict, "the reported account no longer exists")
		}
		if _, err := idc.SetStatus(ctx, tx, r.TargetOwnerID, status, "moderation report "+r.ID+": "+note, "admin", adminID); err != nil &&
			!errors.Is(err, idc.ErrNoChange) {
			return Report{}, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE moderation.reports SET state = $2, resolution = $3, resolution_note = $4, resolved_by = $7,
		resolved_at = now() WHERE id = $1 OR ($8 AND target_type = $5 AND target_id = $6::uuid AND state = 'open')`,
		r.ID, state, resolution, note, r.TargetType, r.TargetID, adminID, state == "actioned"); err != nil {
		return Report{}, err
	}
	r.State, r.Resolution, r.ResolutionNote = state, resolution, note
	return r, nil
}
