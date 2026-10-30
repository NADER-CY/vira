// Package contract is the social module's public surface for other modules (doc 04: modules talk through
// contracts). Functions take the caller's transaction so cross-module changes stay atomic (doc 06 §5).
package contract

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
)

// Summary is the public face of an account (handle, name, avatar).
type Summary struct {
	UserID        string
	Handle        string
	DisplayName   string
	AvatarMediaID string
	IsPrivate     bool
}

// CreateProfile creates the social profile of a new user with a generated handle ("user" + 8 digits), which
// the user can change later. Returns the handle.
func CreateProfile(ctx context.Context, tx db.Tx, userID string) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		n, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
		if err != nil {
			return "", err
		}
		handle := fmt.Sprintf("user%08d", n.Int64())
		tag, err := tx.Exec(ctx, `INSERT INTO social.profiles (user_id, handle, handle_norm) VALUES ($1, $2, $3)
			ON CONFLICT (handle_norm) DO NOTHING`, userID, handle, NormalizeHandle(handle))
		if err != nil {
			return "", fmt.Errorf("social: create profile: %w", err)
		}
		if tag.RowsAffected() == 1 {
			return handle, nil
		}
	}
	return "", errors.New("social: could not allocate a handle")
}

// NormalizeHandle returns the case-insensitive uniqueness key of a handle.
func NormalizeHandle(h string) string { return strings.ToLower(strings.TrimSpace(h)) }

// SetAccountStatus mirrors the identity account status (used to hide suspended or deleted accounts).
func SetAccountStatus(ctx context.Context, tx db.Tx, userID, status string) error {
	_, err := tx.Exec(ctx, `UPDATE social.profiles SET account_status = $2, updated_at = now() WHERE user_id = $1`, userID, status)
	return err
}

// Summaries loads summaries for user ids (missing ids are omitted).
func Summaries(ctx context.Context, q db.Querier, ids []string) (map[string]Summary, error) {
	out := map[string]Summary{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT user_id, handle, display_name, COALESCE(avatar_media_id::text, ''), is_private
		FROM social.profiles WHERE user_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.UserID, &s.Handle, &s.DisplayName, &s.AvatarMediaID, &s.IsPrivate); err != nil {
			return nil, err
		}
		out[s.UserID] = s
	}
	return out, rows.Err()
}

// SearchUserIDs returns user ids whose handle starts with prefix (case-insensitive).
func SearchUserIDs(ctx context.Context, q db.Querier, prefix string, limit int) ([]string, error) {
	p := NormalizeHandle(strings.TrimPrefix(prefix, "@"))
	p = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(p)
	rows, err := q.Query(ctx, `SELECT user_id FROM social.profiles WHERE handle_norm LIKE $1 || '%' ORDER BY handle_norm LIMIT $2`, p, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Stats are profile statistics.
type Stats struct {
	Posts     int `json:"posts"`
	Followers int `json:"followers"`
	Following int `json:"following"`
}

// StatsOf counts published posts, followers and following.
func StatsOf(ctx context.Context, q db.Querier, userID string) (Stats, error) {
	var s Stats
	err := q.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM social.posts WHERE author_id = $1 AND state = 'published'),
		(SELECT count(*) FROM social.follows WHERE followee_id = $1),
		(SELECT count(*) FROM social.follows WHERE follower_id = $1)`, userID).Scan(&s.Posts, &s.Followers, &s.Following)
	return s, err
}

// ErrNotFound is returned when a target does not exist.
var ErrNotFound = errors.New("social: not found")

// ContentOwner returns the author of a post or comment (moderation reports).
func ContentOwner(ctx context.Context, q db.Querier, targetType, id string) (string, error) {
	var sql string
	switch targetType {
	case "post":
		sql = `SELECT author_id FROM social.posts WHERE id = $1 AND state <> 'deleted'`
	case "comment":
		sql = `SELECT author_id FROM social.comments WHERE id = $1 AND state <> 'deleted'`
	case "user":
		sql = `SELECT user_id FROM social.profiles WHERE user_id = $1`
	default:
		return "", ErrNotFound
	}
	var owner string
	if err := q.QueryRow(ctx, sql, id).Scan(&owner); err != nil {
		return "", ErrNotFound
	}
	return owner, nil
}

// RemoveContent hides a post or comment after a moderation decision (not a user deletion).
func RemoveContent(ctx context.Context, tx db.Tx, targetType, id, reason string) error {
	var err error
	switch targetType {
	case "post":
		_, err = tx.Exec(ctx, `UPDATE social.posts SET state = 'removed', removed_reason = $2, updated_at = now()
			WHERE id = $1 AND state = 'published'`, id, reason)
	case "comment":
		_, err = tx.Exec(ctx, `UPDATE social.comments SET state = 'removed' WHERE id = $1 AND state = 'visible'`, id)
	default:
		return fmt.Errorf("social: cannot remove %s", targetType)
	}
	return err
}

// MediaVisible reports whether viewer may fetch a media item: their own upload, an avatar of an account that does
// not block them, or an image of a post they can see.
func MediaVisible(ctx context.Context, q db.Querier, viewer, mediaID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM social.profiles pr WHERE pr.avatar_media_id = $2
		        AND NOT EXISTS (SELECT 1 FROM social.blocks b WHERE b.blocker_id = pr.user_id AND b.blocked_id = $1))
		OR EXISTS (SELECT 1 FROM social.post_media pm JOIN social.posts p ON p.id = pm.post_id
		           JOIN social.profiles a ON a.user_id = p.author_id
		           WHERE pm.media_id = $2 AND `+VisiblePostSQL+`)`, viewer, mediaID).Scan(&ok)
	return ok, err
}

// VisiblePostSQL is the predicate "viewer $1 may see post p (author profile a)": published, author account
// visible, no block in either direction, and public or followed (doc 08: privacy and blocking rules).
const VisiblePostSQL = `p.state = 'published'
	AND a.account_status IN ('active', 'restricted')
	AND NOT EXISTS (SELECT 1 FROM social.blocks b WHERE (b.blocker_id = p.author_id AND b.blocked_id = $1)
	                                                  OR (b.blocker_id = $1 AND b.blocked_id = p.author_id))
	AND (p.author_id = $1
	     OR (NOT a.is_private AND p.visibility = 'public')
	     OR EXISTS (SELECT 1 FROM social.follows f WHERE f.follower_id = $1 AND f.followee_id = p.author_id))`
