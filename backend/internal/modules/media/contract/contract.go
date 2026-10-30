// Package contract is the media module's public surface: validation of uploaded images that other modules attach
// (post media, avatars) and their dimensions.
package contract

import (
	"context"
	"errors"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
)

// Purposes of uploaded media.
const (
	PurposePost   = "post"
	PurposeAvatar = "avatar"
)

// ErrInvalid is returned when a media id is unknown, not owned by the caller, of another purpose or not ready.
var ErrInvalid = errors.New("media: invalid media reference")

// ValidateOwned checks that every id is a ready asset of owner with the given purpose.
func ValidateOwned(ctx context.Context, q db.Querier, owner, purpose string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	var n int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM media.assets WHERE id = ANY($1::uuid[]) AND owner_id = $2 AND purpose = $3 AND state = 'ready'`,
		ids, owner, purpose).Scan(&n); err != nil {
		return err
	}
	if n != len(ids) {
		return ErrInvalid
	}
	return nil
}

// Info describes an image.
type Info struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// URL is the API path that serves a media item.
func URL(id string) string { return "/v1/media/" + id }

// Infos loads dimensions for ids.
func Infos(ctx context.Context, q db.Querier, ids []string) (map[string]Info, error) {
	out := map[string]Info{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT id, width, height FROM media.assets WHERE id = ANY($1::uuid[]) AND state = 'ready'`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var i Info
		if err := rows.Scan(&i.ID, &i.Width, &i.Height); err != nil {
			return nil, err
		}
		i.URL = URL(i.ID)
		out[i.ID] = i
	}
	return out, rows.Err()
}
