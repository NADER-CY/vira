package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	idc "github.com/NADER-CY/vira/backend/internal/modules/identity/contract"
	mediac "github.com/NADER-CY/vira/backend/internal/modules/media/contract"
	"github.com/NADER-CY/vira/backend/internal/modules/social/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/outbox"
	"github.com/NADER-CY/vira/backend/internal/platform/ratelimit"
)

// Limits on writes (abuse prevention; doc 08).
var (
	RulePosts    = ratelimit.Rule{Name: "posts", Limit: 30, Window: time.Hour}
	RuleComments = ratelimit.Rule{Name: "comments", Limit: 60, Window: 10 * time.Minute}
	RuleFollows  = ratelimit.Rule{Name: "follows", Limit: 200, Window: time.Hour}
	RuleLikes    = ratelimit.Rule{Name: "likes", Limit: 600, Window: time.Hour}
)

// PageSize bounds.
const (
	DefaultPage = 20
	MaxPage     = 50
)

// Service implements profiles, the social graph, posts, engagement and feeds.
type Service struct {
	Pool    *db.Pool
	Limiter *ratelimit.Limiter
	Log     *slog.Logger
}

// Meta carries request attributes for audit records.
type Meta struct {
	IP, UserAgent, TraceID string
}

var errNotFound = apperr.New(apperr.NotFound, "")

func forbiddenPrivate() error {
	e := apperr.New(apperr.Forbidden, "private account")
	e.Extra = map[string]any{"reason": "private_account"}
	return e
}

func restricted(ctx context.Context) error {
	if idc.CanWrite(ctx) {
		return nil
	}
	return apperr.New(apperr.AccountRestricted, "")
}

func (s *Service) limit(ctx context.Context, r ratelimit.Rule, userID string) error {
	if s.Limiter == nil {
		return nil
	}
	return s.Limiter.Check(ctx, false, ratelimit.Keyed{Rule: r, Key: userID})
}

// ---------- cursors ----------

type cursor struct {
	T  time.Time
	ID string
}

func (c cursor) String() string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(c.T.UnixMicro(), 10) + "|" + c.ID))
}

func parseCursor(s string) (*cursor, error) {
	if s == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, apperr.Validation(apperr.FieldError{Field: "cursor", Code: "invalid"})
	}
	ts, id, ok := strings.Cut(string(b), "|")
	us, err := strconv.ParseInt(ts, 10, 64)
	if !ok || err != nil || !ids.Valid(id) {
		return nil, apperr.Validation(apperr.FieldError{Field: "cursor", Code: "invalid"})
	}
	return &cursor{T: time.UnixMicro(us).UTC(), ID: id}, nil
}

func pageSize(n int) int {
	if n <= 0 {
		return DefaultPage
	}
	if n > MaxPage {
		return MaxPage
	}
	return n
}

// ---------- visibility ----------

// visiblePost is the shared visibility predicate (see contract.VisiblePostSQL).
const visiblePost = contract.VisiblePostSQL

func blockedEitherWay(ctx context.Context, q db.Querier, a, b string) (bool, error) {
	var x bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM social.blocks WHERE (blocker_id = $1 AND blocked_id = $2)
		OR (blocker_id = $2 AND blocked_id = $1))`, a, b).Scan(&x)
	return x, err
}

// relationship of viewer to target.
type relationship struct {
	IsSelf     bool `json:"is_self"`
	Following  bool `json:"following"`
	Requested  bool `json:"requested"`
	FollowsYou bool `json:"follows_you"`
	Blocking   bool `json:"blocking"`
}

func relationOf(ctx context.Context, q db.Querier, viewer, target string) (relationship, error) {
	r := relationship{IsSelf: viewer == target}
	err := q.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM social.follows WHERE follower_id = $1 AND followee_id = $2),
		EXISTS (SELECT 1 FROM social.follow_requests WHERE requester_id = $1 AND target_id = $2),
		EXISTS (SELECT 1 FROM social.follows WHERE follower_id = $2 AND followee_id = $1),
		EXISTS (SELECT 1 FROM social.blocks WHERE blocker_id = $1 AND blocked_id = $2)`, viewer, target).
		Scan(&r.Following, &r.Requested, &r.FollowsYou, &r.Blocking)
	return r, err
}

// canViewContent reports whether viewer may see target's posts and lists.
func canViewContent(isPrivate bool, rel relationship) bool {
	return rel.IsSelf || !isPrivate || rel.Following
}

// ---------- profiles ----------

// Author is the compact account shown on posts, comments and lists.
type Author struct {
	ID          string `json:"id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	IsPrivate   bool   `json:"is_private"`
}

func mediaURL(id string) string {
	if id == "" {
		return ""
	}
	return "/v1/media/" + id
}

// Profile is a user profile as seen by a viewer.
type Profile struct {
	Author
	Bio          string          `json:"bio"`
	Website      string          `json:"website"`
	Stats        contract.Stats  `json:"stats"`
	Relationship relationship    `json:"relationship"`
	CanViewPosts bool            `json:"can_view_posts"`
	Version      int             `json:"version,omitempty"`
	Privacy      *PrivacySetting `json:"privacy,omitempty"`
}

type profileRow struct {
	id, handle, name, bio, website, avatar, status, commentsFrom string
	private, hideLikes                                           bool
	version                                                      int
	settings                                                     map[string]any
}

func loadProfile(ctx context.Context, q db.Querier, where string, arg any) (profileRow, error) {
	var p profileRow
	err := q.QueryRow(ctx, `SELECT user_id, handle, display_name, bio, website, COALESCE(avatar_media_id::text, ''), account_status,
		comments_from, is_private, hide_like_counts, version, settings FROM social.profiles WHERE `+where, arg).
		Scan(&p.id, &p.handle, &p.name, &p.bio, &p.website, &p.avatar, &p.status, &p.commentsFrom, &p.private, &p.hideLikes, &p.version, &p.settings)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, errNotFound
	}
	return p, err
}

func (p profileRow) author() Author {
	return Author{ID: p.id, Handle: p.handle, DisplayName: p.name, AvatarURL: mediaURL(p.avatar), IsPrivate: p.private}
}

func visibleStatus(s string) bool { return s == idc.StatusActive || s == idc.StatusRestricted }

// ProfileByHandle returns a profile for viewer. Accounts that blocked the viewer, or that are suspended or
// deleted, are reported as not found (no information leak).
func (s *Service) ProfileByHandle(ctx context.Context, viewer, handle string) (Profile, error) {
	p, err := loadProfile(ctx, s.Pool, "handle_norm = $1", contract.NormalizeHandle(strings.TrimPrefix(handle, "@")))
	if err != nil {
		return Profile{}, err
	}
	return s.profileView(ctx, viewer, p)
}

// ProfileByID is ProfileByHandle by user id.
func (s *Service) ProfileByID(ctx context.Context, viewer, id string) (Profile, error) {
	if !ids.Valid(id) {
		return Profile{}, errNotFound
	}
	p, err := loadProfile(ctx, s.Pool, "user_id = $1", id)
	if err != nil {
		return Profile{}, err
	}
	return s.profileView(ctx, viewer, p)
}

func (s *Service) profileView(ctx context.Context, viewer string, p profileRow) (Profile, error) {
	if viewer != p.id && !visibleStatus(p.status) {
		return Profile{}, errNotFound
	}
	var blockedMe bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM social.blocks WHERE blocker_id = $1 AND blocked_id = $2)`, p.id, viewer).Scan(&blockedMe); err != nil {
		return Profile{}, err
	}
	if blockedMe {
		return Profile{}, errNotFound
	}
	rel, err := relationOf(ctx, s.Pool, viewer, p.id)
	if err != nil {
		return Profile{}, err
	}
	st, err := contract.StatsOf(ctx, s.Pool, p.id)
	if err != nil {
		return Profile{}, err
	}
	out := Profile{Author: p.author(), Bio: p.bio, Website: p.website, Stats: st, Relationship: rel,
		CanViewPosts: canViewContent(p.private, rel) && !rel.Blocking}
	if rel.IsSelf {
		out.Version = p.version
		ps := privacyOf(p)
		out.Privacy = &ps
	}
	return out, nil
}

var handleRe = regexp.MustCompile(`^[A-Za-z0-9_.]{3,30}$`)

// reservedHandles cannot be taken by users (impersonation of the platform or support).
var reservedHandles = map[string]bool{"vira": true, "admin": true, "support": true, "official": true, "help": true, "security": true,
	"ویرا": true, "root": true, "system": true, "moderator": true}

// ProfileUpdate changes the caller's own profile; nil fields are unchanged.
type ProfileUpdate struct {
	Handle        *string `json:"handle"`
	DisplayName   *string `json:"display_name"`
	Bio           *string `json:"bio"`
	Website       *string `json:"website"`
	AvatarMediaID *string `json:"avatar_media_id"`
}

// UpdateProfile applies u with optimistic concurrency (expected version from If-Match).
func (s *Service) UpdateProfile(ctx context.Context, userID string, version int, u ProfileUpdate, m Meta) (Profile, error) {
	var fe []apperr.FieldError
	if u.Handle != nil {
		h := strings.TrimSpace(strings.TrimPrefix(*u.Handle, "@"))
		u.Handle = &h
		if !handleRe.MatchString(h) || strings.HasPrefix(h, ".") || strings.HasSuffix(h, ".") || strings.Contains(h, "..") {
			fe = append(fe, apperr.FieldError{Field: "handle", Code: "invalid", Detail: "3-30 letters, digits, _ or ."})
		} else if reservedHandles[contract.NormalizeHandle(h)] {
			fe = append(fe, apperr.FieldError{Field: "handle", Code: "reserved"})
		}
	}
	if u.DisplayName != nil {
		d := strings.TrimSpace(*u.DisplayName)
		u.DisplayName = &d
		if utf8.RuneCountInString(d) > 50 {
			fe = append(fe, apperr.FieldError{Field: "display_name", Code: "too_long"})
		}
	}
	if u.Bio != nil {
		b := strings.TrimSpace(*u.Bio)
		u.Bio = &b
		if utf8.RuneCountInString(b) > 300 {
			fe = append(fe, apperr.FieldError{Field: "bio", Code: "too_long"})
		}
	}
	if u.Website != nil {
		w := strings.TrimSpace(*u.Website)
		u.Website = &w
		if w != "" {
			pu, err := url.Parse(w)
			if err != nil || (pu.Scheme != "https" && pu.Scheme != "http") || pu.Host == "" || len(w) > 200 {
				fe = append(fe, apperr.FieldError{Field: "website", Code: "invalid"})
			}
		}
	}
	if u.AvatarMediaID != nil && *u.AvatarMediaID != "" && !ids.Valid(*u.AvatarMediaID) {
		fe = append(fe, apperr.FieldError{Field: "avatar_media_id", Code: "invalid"})
	}
	if fe != nil {
		return Profile{}, apperr.Validation(fe...)
	}
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		var cur int
		if err := tx.QueryRow(ctx, `SELECT version FROM social.profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&cur); err != nil {
			return err
		}
		if cur != version {
			return apperr.New(apperr.PreconditionFailed, "")
		}
		if u.AvatarMediaID != nil && *u.AvatarMediaID != "" {
			if err := mediac.ValidateOwned(ctx, tx, userID, mediac.PurposeAvatar, []string{*u.AvatarMediaID}); err != nil {
				return apperr.Validation(apperr.FieldError{Field: "avatar_media_id", Code: "invalid"})
			}
		}
		var avatar any = nil
		setAvatar := u.AvatarMediaID != nil
		if setAvatar && *u.AvatarMediaID != "" {
			avatar = *u.AvatarMediaID
		}
		var handle, handleNorm any
		if u.Handle != nil {
			handle, handleNorm = *u.Handle, contract.NormalizeHandle(*u.Handle)
		}
		_, err := tx.Exec(ctx, `UPDATE social.profiles SET
			handle = COALESCE($2, handle), handle_norm = COALESCE($3, handle_norm),
			display_name = COALESCE($4, display_name), bio = COALESCE($5, bio), website = COALESCE($6, website),
			avatar_media_id = CASE WHEN $7 THEN $8::uuid ELSE avatar_media_id END,
			version = version + 1, updated_at = now() WHERE user_id = $1`,
			userID, handle, handleNorm, u.DisplayName, u.Bio, u.Website, setAvatar, avatar)
		if db.IsUniqueViolation(err) {
			return apperr.Validation(apperr.FieldError{Field: "handle", Code: "taken"})
		}
		if err != nil {
			return err
		}
		_, err = audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "profile.updated", TargetType: "user",
			TargetID: userID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID,
			Metadata: map[string]any{"handle_changed": u.Handle != nil}})
		return err
	})
	if err != nil {
		return Profile{}, err
	}
	return s.ProfileByID(ctx, userID, userID)
}

// PrivacySetting is the caller's privacy configuration (references: privacy screens _125/_46/_1).
type PrivacySetting struct {
	IsPrivate      bool           `json:"is_private"`
	CommentsFrom   string         `json:"comments_from"`
	HideLikeCounts bool           `json:"hide_like_counts"`
	Settings       map[string]any `json:"settings"`
}

func privacyOf(p profileRow) PrivacySetting {
	st := p.settings
	if st == nil {
		st = map[string]any{}
	}
	return PrivacySetting{IsPrivate: p.private, CommentsFrom: p.commentsFrom, HideLikeCounts: p.hideLikes, Settings: st}
}

// laterPhaseSettings lists preferences stored now and enforced when their feature ships, with allowed values.
var laterPhaseSettings = map[string][]string{
	"story_replies_from": {"everyone", "followers", "nobody"}, // P3 stories
	"mentions_from":      {"everyone", "followers", "nobody"}, // P3/P5
	"messages_from":      {"everyone", "followers", "nobody"}, // P5 messaging
	"live_comments_from": {"everyone", "followers", "nobody"}, // P8 live
	"allow_product_tags": {"true", "false"},                   // P4 commerce tags
	"show_activity":      {"true", "false"},                   // P5 presence
	"allow_reshare":      {"true", "false"},                   // P3
}

// Privacy returns the caller's privacy settings.
func (s *Service) Privacy(ctx context.Context, userID string) (PrivacySetting, error) {
	p, err := loadProfile(ctx, s.Pool, "user_id = $1", userID)
	if err != nil {
		return PrivacySetting{}, err
	}
	return privacyOf(p), nil
}

// SetPrivacy replaces the caller's privacy settings. Making an account public approves pending follow requests.
func (s *Service) SetPrivacy(ctx context.Context, userID string, in PrivacySetting, m Meta) (PrivacySetting, error) {
	var fe []apperr.FieldError
	switch in.CommentsFrom {
	case "everyone", "followers", "nobody":
	default:
		fe = append(fe, apperr.FieldError{Field: "comments_from", Code: "invalid"})
	}
	clean := map[string]any{}
	for k, v := range in.Settings {
		allowed, ok := laterPhaseSettings[k]
		if !ok {
			fe = append(fe, apperr.FieldError{Field: "settings." + k, Code: "unknown"})
			continue
		}
		sv := fmt.Sprint(v)
		valid := false
		for _, a := range allowed {
			if sv == a {
				valid = true
			}
		}
		if !valid {
			fe = append(fe, apperr.FieldError{Field: "settings." + k, Code: "invalid"})
			continue
		}
		clean[k] = v
	}
	if fe != nil {
		return PrivacySetting{}, apperr.Validation(fe...)
	}
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		var wasPrivate bool
		if err := tx.QueryRow(ctx, `SELECT is_private FROM social.profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&wasPrivate); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE social.profiles SET is_private = $2, comments_from = $3, hide_like_counts = $4,
			settings = $5::jsonb, version = version + 1, updated_at = now() WHERE user_id = $1`,
			userID, in.IsPrivate, in.CommentsFrom, in.HideLikeCounts, jsonString(clean)); err != nil {
			return err
		}
		if wasPrivate && !in.IsPrivate {
			if _, err := tx.Exec(ctx, `INSERT INTO social.follows (follower_id, followee_id)
				SELECT requester_id, target_id FROM social.follow_requests WHERE target_id = $1 ON CONFLICT DO NOTHING`, userID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM social.follow_requests WHERE target_id = $1`, userID); err != nil {
				return err
			}
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "privacy.updated", TargetType: "user",
			TargetID: userID, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID,
			Metadata: map[string]any{"is_private": in.IsPrivate, "comments_from": in.CommentsFrom}})
		return err
	})
	if err != nil {
		return PrivacySetting{}, err
	}
	return s.Privacy(ctx, userID)
}

// ---------- graph ----------

// FollowResult reports the relationship after a follow request.
type FollowResult struct {
	State string `json:"state"` // following | requested
}

// Follow follows target, or files a follow request when target is private.
func (s *Service) Follow(ctx context.Context, viewer, target string, m Meta) (FollowResult, error) {
	if err := restricted(ctx); err != nil {
		return FollowResult{}, err
	}
	if viewer == target {
		return FollowResult{}, apperr.Validation(apperr.FieldError{Field: "user_id", Code: "self"})
	}
	if err := s.limit(ctx, RuleFollows, viewer); err != nil {
		return FollowResult{}, err
	}
	var res FollowResult
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: viewer}, db.TxOptions{}, func(tx db.Tx) error {
		p, err := loadProfile(ctx, tx, "user_id = $1", target)
		if err != nil || !visibleStatus(p.status) {
			return errNotFound
		}
		if b, err := blockedEitherWay(ctx, tx, viewer, target); err != nil {
			return err
		} else if b {
			return errNotFound
		}
		var already bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM social.follows WHERE follower_id = $1 AND followee_id = $2)`, viewer, target).Scan(&already); err != nil {
			return err
		}
		if already {
			res.State = "following"
			return nil
		}
		if p.private {
			res.State = "requested"
			_, err = tx.Exec(ctx, `INSERT INTO social.follow_requests (requester_id, target_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, viewer, target)
			return err
		}
		res.State = "following"
		if _, err = tx.Exec(ctx, `INSERT INTO social.follows (follower_id, followee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, viewer, target); err != nil {
			return err
		}
		_, err = outbox.Write(ctx, tx, outbox.Event{Module: "social", AggregateType: "user", AggregateID: target, Type: "vira.social.followed.v1",
			Data: map[string]any{"follower_id": viewer, "followee_id": target}})
		return err
	})
	return res, err
}

// Unfollow removes a follow or cancels a pending request (idempotent).
func (s *Service) Unfollow(ctx context.Context, viewer, target string) error {
	if !ids.Valid(target) {
		return errNotFound
	}
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: viewer}, db.TxOptions{}, func(tx db.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM social.follows WHERE follower_id = $1 AND followee_id = $2`, viewer, target); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM social.follow_requests WHERE requester_id = $1 AND target_id = $2`, viewer, target)
		return err
	})
}

// RemoveFollower removes follower from the caller's followers.
func (s *Service) RemoveFollower(ctx context.Context, userID, follower string) error {
	if !ids.Valid(follower) {
		return errNotFound
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM social.follows WHERE follower_id = $1 AND followee_id = $2`, follower, userID)
	return err
}

// Block blocks target: both follow directions and pending requests are removed and neither side sees the other.
func (s *Service) Block(ctx context.Context, viewer, target string, m Meta) error {
	if viewer == target {
		return apperr.Validation(apperr.FieldError{Field: "user_id", Code: "self"})
	}
	if !ids.Valid(target) {
		return errNotFound
	}
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: viewer}, db.TxOptions{}, func(tx db.Tx) error {
		if _, err := loadProfile(ctx, tx, "user_id = $1", target); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO social.blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, viewer, target); err != nil {
			return err
		}
		for _, q := range []string{
			`DELETE FROM social.follows WHERE (follower_id = $1 AND followee_id = $2) OR (follower_id = $2 AND followee_id = $1)`,
			`DELETE FROM social.follow_requests WHERE (requester_id = $1 AND target_id = $2) OR (requester_id = $2 AND target_id = $1)`,
		} {
			if _, err := tx.Exec(ctx, q, viewer, target); err != nil {
				return err
			}
		}
		_, err := audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: viewer, Action: "user.blocked", TargetType: "user",
			TargetID: target, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
		return err
	})
}

// Unblock removes a block (idempotent).
func (s *Service) Unblock(ctx context.Context, viewer, target string) error {
	if !ids.Valid(target) {
		return errNotFound
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM social.blocks WHERE blocker_id = $1 AND blocked_id = $2`, viewer, target)
	return err
}

// Page is a cursor-paginated list.
type Page[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor"`
}

// ListEntry is an account in a followers/following/blocks/requests list.
type ListEntry struct {
	Author
	Since     time.Time `json:"since"`
	Following bool      `json:"following"` // whether the viewer follows this account
}

func (s *Service) listAccounts(ctx context.Context, viewer, sql string, args []any, cur *cursor, n int) (Page[ListEntry], error) {
	full := append([]any{viewer}, args...)
	if cur != nil {
		full = append(full, cur.T, cur.ID)
	} else {
		full = append(full, nil, nil)
	}
	full = append(full, n+1)
	rows, err := s.Pool.Query(ctx, sql, full...)
	if err != nil {
		return Page[ListEntry]{}, err
	}
	defer rows.Close()
	out := Page[ListEntry]{Data: []ListEntry{}}
	for rows.Next() {
		var e ListEntry
		var avatar string
		if err := rows.Scan(&e.ID, &e.Handle, &e.DisplayName, &avatar, &e.IsPrivate, &e.Since, &e.Following); err != nil {
			return out, err
		}
		e.AvatarURL = mediaURL(avatar)
		out.Data = append(out.Data, e)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Data) > n {
		last := out.Data[n-1]
		c := cursor{T: last.Since, ID: last.ID}.String()
		out.NextCursor = &c
		out.Data = out.Data[:n]
	}
	return out, nil
}

// listSQL builds a list query over relation table rel joining profiles on column join, filtered by where.
// Placeholders: $1 viewer, $2 subject, then cursor time/id and limit.
func listSQL(rel, join, where string) string {
	return `SELECT pr.user_id, pr.handle, pr.display_name, COALESCE(pr.avatar_media_id::text, ''), pr.is_private, r.created_at,
		EXISTS (SELECT 1 FROM social.follows vf WHERE vf.follower_id = $1 AND vf.followee_id = pr.user_id)
		FROM ` + rel + ` r JOIN social.profiles pr ON pr.user_id = r.` + join + `
		WHERE ` + where + ` AND pr.account_status IN ('active', 'restricted')
		  AND NOT EXISTS (SELECT 1 FROM social.blocks b WHERE (b.blocker_id = pr.user_id AND b.blocked_id = $1)
		                                                 OR (b.blocker_id = $1 AND b.blocked_id = pr.user_id))
		  AND ($3::timestamptz IS NULL OR (r.created_at, pr.user_id) < ($3, $4::uuid))
		ORDER BY r.created_at DESC, pr.user_id DESC LIMIT $5`
}

func (s *Service) graphGate(ctx context.Context, viewer, subject string) error {
	p, err := loadProfile(ctx, s.Pool, "user_id = $1", subject)
	if err != nil || (!visibleStatus(p.status) && viewer != subject) {
		return errNotFound
	}
	rel, err := relationOf(ctx, s.Pool, viewer, subject)
	if err != nil {
		return err
	}
	if b, err := blockedEitherWay(ctx, s.Pool, viewer, subject); err != nil {
		return err
	} else if b && !rel.IsSelf {
		return errNotFound
	}
	if !canViewContent(p.private, rel) {
		return forbiddenPrivate()
	}
	return nil
}

// Followers lists who follows subject (private accounts: followers and the owner only).
func (s *Service) Followers(ctx context.Context, viewer, subject, cur string, n int) (Page[ListEntry], error) {
	if !ids.Valid(subject) {
		return Page[ListEntry]{}, errNotFound
	}
	c, err := parseCursor(cur)
	if err != nil {
		return Page[ListEntry]{}, err
	}
	if err := s.graphGate(ctx, viewer, subject); err != nil {
		return Page[ListEntry]{}, err
	}
	return s.listAccounts(ctx, viewer, listSQL("social.follows", "follower_id", "r.followee_id = $2"), []any{subject}, c, pageSize(n))
}

// Following lists whom subject follows.
func (s *Service) Following(ctx context.Context, viewer, subject, cur string, n int) (Page[ListEntry], error) {
	if !ids.Valid(subject) {
		return Page[ListEntry]{}, errNotFound
	}
	c, err := parseCursor(cur)
	if err != nil {
		return Page[ListEntry]{}, err
	}
	if err := s.graphGate(ctx, viewer, subject); err != nil {
		return Page[ListEntry]{}, err
	}
	return s.listAccounts(ctx, viewer, listSQL("social.follows", "followee_id", "r.follower_id = $2"), []any{subject}, c, pageSize(n))
}

// FollowRequests lists pending requests to the caller.
func (s *Service) FollowRequests(ctx context.Context, userID, cur string, n int) (Page[ListEntry], error) {
	c, err := parseCursor(cur)
	if err != nil {
		return Page[ListEntry]{}, err
	}
	return s.listAccounts(ctx, userID, listSQL("social.follow_requests", "requester_id", "r.target_id = $2"), []any{userID}, c, pageSize(n))
}

// Blocks lists accounts the caller blocked. (Blocked accounts are excluded by the generic list filter, so this
// uses its own query.)
func (s *Service) Blocks(ctx context.Context, userID string) ([]ListEntry, error) {
	rows, err := s.Pool.Query(ctx, `SELECT pr.user_id, pr.handle, pr.display_name, COALESCE(pr.avatar_media_id::text, ''), pr.is_private, b.created_at
		FROM social.blocks b JOIN social.profiles pr ON pr.user_id = b.blocked_id WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 500`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ListEntry{}
	for rows.Next() {
		var e ListEntry
		var avatar string
		if err := rows.Scan(&e.ID, &e.Handle, &e.DisplayName, &avatar, &e.IsPrivate, &e.Since); err != nil {
			return nil, err
		}
		e.AvatarURL = mediaURL(avatar)
		out = append(out, e)
	}
	return out, rows.Err()
}

// DecideFollowRequest approves or declines a pending request.
func (s *Service) DecideFollowRequest(ctx context.Context, userID, requester string, approve bool) error {
	if !ids.Valid(requester) {
		return errNotFound
	}
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM social.follow_requests WHERE requester_id = $1 AND target_id = $2`, requester, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errNotFound
		}
		if approve {
			_, err = tx.Exec(ctx, `INSERT INTO social.follows (follower_id, followee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, requester, userID)
		}
		return err
	})
}

func jsonString(v map[string]any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
