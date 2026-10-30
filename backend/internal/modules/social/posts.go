package social

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	mediac "github.com/NADER-CY/vira/backend/internal/modules/media/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/audit"
	"github.com/NADER-CY/vira/backend/internal/platform/db"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
	"github.com/NADER-CY/vira/backend/internal/platform/outbox"
)

// Post is a post as seen by a viewer.
type Post struct {
	ID              string        `json:"id"`
	Author          Author        `json:"author"`
	Caption         string        `json:"caption"`
	Visibility      string        `json:"visibility"`
	Media           []mediac.Info `json:"media"`
	LikeCount       *int          `json:"like_count"` // null when the author hides like counts (except for the author)
	CommentCount    int           `json:"comment_count"`
	ViewerLiked     bool          `json:"viewer_liked"`
	IsOwner         bool          `json:"is_owner"`
	CommentsAllowed bool          `json:"comments_allowed"`
	PublishedAt     time.Time     `json:"published_at"`
	EditedAt        *time.Time    `json:"edited_at"`
	Version         int           `json:"version"`
	ViewerFollows   bool          `json:"viewer_follows_author"`
}

// PostInput creates a post.
type PostInput struct {
	Caption    string   `json:"caption"`
	Visibility string   `json:"visibility"`
	MediaIDs   []string `json:"media_ids"`
}

func validatePost(caption, visibility string, mediaIDs []string, creating bool) []apperr.FieldError {
	var fe []apperr.FieldError
	if utf8.RuneCountInString(caption) > 2200 {
		fe = append(fe, apperr.FieldError{Field: "caption", Code: "too_long"})
	}
	switch visibility {
	case "public", "followers":
	default:
		fe = append(fe, apperr.FieldError{Field: "visibility", Code: "invalid"})
	}
	if creating {
		if len(mediaIDs) > 10 {
			fe = append(fe, apperr.FieldError{Field: "media_ids", Code: "too_many"})
		}
		if len(mediaIDs) == 0 && strings.TrimSpace(caption) == "" {
			fe = append(fe, apperr.FieldError{Field: "caption", Code: "required", Detail: "a post needs text or at least one image"})
		}
		seen := map[string]bool{}
		for _, id := range mediaIDs {
			if !ids.Valid(id) || seen[id] {
				fe = append(fe, apperr.FieldError{Field: "media_ids", Code: "invalid"})
				break
			}
			seen[id] = true
		}
	}
	return fe
}

// CreatePost publishes a post with optional images.
func (s *Service) CreatePost(ctx context.Context, userID string, in PostInput, m Meta) (Post, error) {
	if err := restricted(ctx); err != nil {
		return Post{}, err
	}
	in.Caption = strings.TrimSpace(in.Caption)
	if in.Visibility == "" {
		in.Visibility = "public"
	}
	if fe := validatePost(in.Caption, in.Visibility, in.MediaIDs, true); fe != nil {
		return Post{}, apperr.Validation(fe...)
	}
	if err := s.limit(ctx, RulePosts, userID); err != nil {
		return Post{}, err
	}
	id := ids.New()
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		if err := mediac.ValidateOwned(ctx, tx, userID, mediac.PurposePost, in.MediaIDs); err != nil {
			if errors.Is(err, mediac.ErrInvalid) {
				return apperr.Validation(apperr.FieldError{Field: "media_ids", Code: "invalid"})
			}
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO social.posts (id, author_id, caption, visibility) VALUES ($1, $2, $3, $4)`,
			id, userID, in.Caption, in.Visibility); err != nil {
			return err
		}
		for i, mid := range in.MediaIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO social.post_media (post_id, position, media_id) VALUES ($1, $2, $3)`, id, i, mid); err != nil {
				if db.IsUniqueViolation(err) {
					return apperr.Validation(apperr.FieldError{Field: "media_ids", Code: "already_used"})
				}
				return err
			}
		}
		_, err := outbox.Write(ctx, tx, outbox.Event{Module: "content", AggregateType: "post", AggregateID: id, AggregateVersion: 1,
			Type: "vira.content.post_published.v1", Data: map[string]any{"post_id": id, "author_id": userID}})
		return err
	})
	if err != nil {
		return Post{}, err
	}
	return s.GetPost(ctx, userID, id)
}

// postQuery selects posts joined with author profiles; callers append WHERE/ORDER/LIMIT using $1 = viewer.
const postQuery = `SELECT p.id, p.author_id, a.handle, a.display_name, COALESCE(a.avatar_media_id::text, ''), a.is_private,
	a.hide_like_counts, a.comments_from, p.caption, p.visibility, p.published_at, p.edited_at, p.version
	FROM social.posts p JOIN social.profiles a ON a.user_id = p.author_id`

type postRow struct {
	post         Post
	hideLikes    bool
	commentsFrom string
}

func scanPosts(rows pgx.Rows) ([]postRow, error) {
	defer rows.Close()
	var out []postRow
	for rows.Next() {
		var r postRow
		var avatar string
		p := &r.post
		if err := rows.Scan(&p.ID, &p.Author.ID, &p.Author.Handle, &p.Author.DisplayName, &avatar, &p.Author.IsPrivate,
			&r.hideLikes, &r.commentsFrom, &p.Caption, &p.Visibility, &p.PublishedAt, &p.EditedAt, &p.Version); err != nil {
			return nil, err
		}
		p.Author.AvatarURL = mediaURL(avatar)
		out = append(out, r)
	}
	return out, rows.Err()
}

// hydrate adds media, counts and viewer state to posts (batched queries).
func (s *Service) hydrate(ctx context.Context, q db.Querier, viewer string, rows []postRow) ([]Post, error) {
	posts := make([]Post, len(rows))
	if len(rows) == 0 {
		return posts, nil
	}
	pids := make([]string, len(rows))
	for i, r := range rows {
		pids[i] = r.post.ID
	}
	type agg struct {
		likes, comments int
		liked           bool
		media           []string
	}
	a := map[string]*agg{}
	for _, id := range pids {
		a[id] = &agg{}
	}
	rs, err := q.Query(ctx, `SELECT p.id,
		(SELECT count(*) FROM social.likes l WHERE l.post_id = p.id),
		(SELECT count(*) FROM social.comments c JOIN social.profiles ca ON ca.user_id = c.author_id
		  WHERE c.post_id = p.id AND c.state = 'visible' AND ca.account_status IN ('active', 'restricted')),
		EXISTS (SELECT 1 FROM social.likes l WHERE l.post_id = p.id AND l.user_id = $1),
		COALESCE((SELECT array_agg(pm.media_id::text ORDER BY pm.position) FROM social.post_media pm WHERE pm.post_id = p.id), '{}')
		FROM social.posts p WHERE p.id = ANY($2::uuid[])`, viewer, pids)
	if err != nil {
		return nil, err
	}
	var allMedia []string
	for rs.Next() {
		var id string
		x := &agg{}
		if err := rs.Scan(&id, &x.likes, &x.comments, &x.liked, &x.media); err != nil {
			rs.Close()
			return nil, err
		}
		a[id] = x
		allMedia = append(allMedia, x.media...)
	}
	rs.Close()
	if err := rs.Err(); err != nil {
		return nil, err
	}
	infos, err := mediac.Infos(ctx, q, allMedia)
	if err != nil {
		return nil, err
	}
	authors := map[string]bool{}
	for _, r := range rows {
		authors[r.post.Author.ID] = true
	}
	following := map[string]bool{}
	var aids []string
	for id := range authors {
		aids = append(aids, id)
	}
	fr, err := q.Query(ctx, `SELECT followee_id FROM social.follows WHERE follower_id = $1 AND followee_id = ANY($2::uuid[])`, viewer, aids)
	if err != nil {
		return nil, err
	}
	for fr.Next() {
		var id string
		if err := fr.Scan(&id); err != nil {
			fr.Close()
			return nil, err
		}
		following[id] = true
	}
	fr.Close()
	for i, r := range rows {
		p := r.post
		x := a[p.ID]
		p.IsOwner = p.Author.ID == viewer
		p.ViewerFollows = following[p.Author.ID]
		p.CommentCount = x.comments
		p.ViewerLiked = x.liked
		if !r.hideLikes || p.IsOwner {
			n := x.likes
			p.LikeCount = &n
		}
		p.Media = []mediac.Info{}
		for _, mid := range x.media {
			if inf, ok := infos[mid]; ok {
				p.Media = append(p.Media, inf)
			}
		}
		p.CommentsAllowed = commentsAllowed(r.commentsFrom, p.IsOwner, p.ViewerFollows)
		posts[i] = p
	}
	return posts, nil
}

func commentsAllowed(from string, isOwner, follows bool) bool {
	switch {
	case isOwner:
		return true
	case from == "everyone":
		return true
	case from == "followers":
		return follows
	default:
		return false
	}
}

// GetPost returns a post if the viewer may see it (otherwise not found — no information leak).
func (s *Service) GetPost(ctx context.Context, viewer, id string) (Post, error) {
	if !ids.Valid(id) {
		return Post{}, errNotFound
	}
	rows, err := s.Pool.Query(ctx, postQuery+` WHERE p.id = $2 AND `+visiblePost, viewer, id)
	if err != nil {
		return Post{}, err
	}
	pr, err := scanPosts(rows)
	if err != nil {
		return Post{}, err
	}
	if len(pr) == 0 {
		return Post{}, errNotFound
	}
	posts, err := s.hydrate(ctx, s.Pool, viewer, pr)
	if err != nil {
		return Post{}, err
	}
	return posts[0], nil
}

// PostUpdate edits an own post (caption and visibility; images cannot be changed after publishing).
type PostUpdate struct {
	Caption    *string `json:"caption"`
	Visibility *string `json:"visibility"`
}

// UpdatePost edits an own post with optimistic concurrency.
func (s *Service) UpdatePost(ctx context.Context, userID, id string, version int, u PostUpdate) (Post, error) {
	if err := restricted(ctx); err != nil {
		return Post{}, err
	}
	if !ids.Valid(id) {
		return Post{}, errNotFound
	}
	err := db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		var author, caption, visibility string
		var cur, nMedia int
		err := tx.QueryRow(ctx, `SELECT author_id, caption, visibility, version,
			(SELECT count(*) FROM social.post_media WHERE post_id = $1) FROM social.posts WHERE id = $1 AND state = 'published' FOR UPDATE`, id).
			Scan(&author, &caption, &visibility, &cur, &nMedia)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNotFound
		}
		if err != nil {
			return err
		}
		if author != userID {
			return apperr.New(apperr.Forbidden, "")
		}
		if cur != version {
			return apperr.New(apperr.PreconditionFailed, "")
		}
		if u.Caption != nil {
			caption = strings.TrimSpace(*u.Caption)
		}
		if u.Visibility != nil {
			visibility = *u.Visibility
		}
		if fe := validatePost(caption, visibility, nil, false); fe != nil {
			return apperr.Validation(fe...)
		}
		if nMedia == 0 && caption == "" {
			return apperr.Validation(apperr.FieldError{Field: "caption", Code: "required"})
		}
		_, err = tx.Exec(ctx, `UPDATE social.posts SET caption = $2, visibility = $3, version = version + 1, edited_at = now(), updated_at = now()
			WHERE id = $1`, id, caption, visibility)
		return err
	})
	if err != nil {
		return Post{}, err
	}
	return s.GetPost(ctx, userID, id)
}

// DeletePost deletes an own post.
func (s *Service) DeletePost(ctx context.Context, userID, id string, m Meta) error {
	if !ids.Valid(id) {
		return errNotFound
	}
	return db.InTx(ctx, s.Pool, db.Scope{Audience: db.AudiencePublic, ActorID: userID}, db.TxOptions{}, func(tx db.Tx) error {
		var author string
		err := tx.QueryRow(ctx, `SELECT author_id FROM social.posts WHERE id = $1 AND state = 'published' FOR UPDATE`, id).Scan(&author)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNotFound
		}
		if err != nil {
			return err
		}
		if author != userID {
			return apperr.New(apperr.Forbidden, "")
		}
		if _, err := tx.Exec(ctx, `UPDATE social.posts SET state = 'deleted', updated_at = now() WHERE id = $1`, id); err != nil {
			return err
		}
		_, err = audit.Record(ctx, tx, audit.Event{ActorType: "user", ActorID: userID, Action: "post.deleted", TargetType: "post",
			TargetID: id, Outcome: audit.Success, IP: m.IP, UserAgent: m.UserAgent, TraceID: m.TraceID})
		return err
	})
}

// listPosts runs a feed-style query: extra WHERE clause after visibility, keyset pagination on (published_at, id).
// $1 = viewer; extra args start at $2; the cursor and limit placeholders follow.
func (s *Service) listPosts(ctx context.Context, viewer, where string, args []any, cur string, n int) (Page[Post], error) {
	c, err := parseCursor(cur)
	if err != nil {
		return Page[Post]{}, err
	}
	n = pageSize(n)
	all := append([]any{viewer}, args...)
	k := len(all)
	var ct, cid any
	if c != nil {
		ct, cid = c.T, c.ID
	}
	all = append(all, ct, cid, n+1)
	sql := postQuery + ` WHERE ` + visiblePost + where +
		` AND ($` + itoa(k+1) + `::timestamptz IS NULL OR (p.published_at, p.id) < ($` + itoa(k+1) + `, $` + itoa(k+2) + `::uuid))` +
		` ORDER BY p.published_at DESC, p.id DESC LIMIT $` + itoa(k+3)
	rows, err := s.Pool.Query(ctx, sql, all...)
	if err != nil {
		return Page[Post]{}, err
	}
	pr, err := scanPosts(rows)
	if err != nil {
		return Page[Post]{}, err
	}
	var next *string
	if len(pr) > n {
		last := pr[n-1].post
		x := cursor{T: last.PublishedAt, ID: last.ID}.String()
		next = &x
		pr = pr[:n]
	}
	posts, err := s.hydrate(ctx, s.Pool, viewer, pr)
	if err != nil {
		return Page[Post]{}, err
	}
	return Page[Post]{Data: posts, NextCursor: next}, nil
}

func itoa(i int) string {
	const d = "0123456789"
	if i < 10 {
		return d[i : i+1]
	}
	return itoa(i/10) + d[i%10:i%10+1]
}

// HomeFeed ("برای شما") lists recent posts the viewer may see, newest first. Ranking arrives in P9; P2 is
// chronological over public and followed accounts.
func (s *Service) HomeFeed(ctx context.Context, viewer, cur string, n int) (Page[Post], error) {
	return s.listPosts(ctx, viewer, "", nil, cur, n)
}

// FollowingFeed ("دنبال‌شده‌ها") lists posts of followed accounts and the viewer's own.
func (s *Service) FollowingFeed(ctx context.Context, viewer, cur string, n int) (Page[Post], error) {
	return s.listPosts(ctx, viewer, ` AND (p.author_id = $1 OR EXISTS (SELECT 1 FROM social.follows ff WHERE ff.follower_id = $1 AND ff.followee_id = p.author_id))`,
		nil, cur, n)
}

// UserPosts lists a user's posts (private accounts: followers and the owner only).
func (s *Service) UserPosts(ctx context.Context, viewer, subject, cur string, n int) (Page[Post], error) {
	if !ids.Valid(subject) {
		return Page[Post]{}, errNotFound
	}
	if err := s.graphGate(ctx, viewer, subject); err != nil {
		return Page[Post]{}, err
	}
	return s.listPosts(ctx, viewer, ` AND p.author_id = $2`, []any{subject}, cur, n)
}

// ---------- engagement ----------

// LikeState is returned by like/unlike.
type LikeState struct {
	Liked     bool `json:"liked"`
	LikeCount *int `json:"like_count"`
}

// SetLike likes or unlikes a visible post (idempotent).
func (s *Service) SetLike(ctx context.Context, viewer, postID string, like bool) (LikeState, error) {
	if like {
		if err := restricted(ctx); err != nil {
			return LikeState{}, err
		}
		if err := s.limit(ctx, RuleLikes, viewer); err != nil {
			return LikeState{}, err
		}
	}
	p, err := s.GetPost(ctx, viewer, postID)
	if err != nil {
		return LikeState{}, err
	}
	if like {
		_, err = s.Pool.Exec(ctx, `INSERT INTO social.likes (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, postID, viewer)
	} else {
		_, err = s.Pool.Exec(ctx, `DELETE FROM social.likes WHERE post_id = $1 AND user_id = $2`, postID, viewer)
	}
	if err != nil {
		return LikeState{}, err
	}
	st := LikeState{Liked: like}
	if p.LikeCount != nil {
		var n int
		if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM social.likes WHERE post_id = $1`, postID).Scan(&n); err != nil {
			return LikeState{}, err
		}
		st.LikeCount = &n
	}
	return st, nil
}

// Comment is a comment as seen by a viewer.
type Comment struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	Author    Author    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	CanDelete bool      `json:"can_delete"`
	IsAuthor  bool      `json:"is_post_author"` // the comment was written by the post's author ("نویسنده" badge)
}

// Comments lists visible comments of a visible post, oldest first.
func (s *Service) Comments(ctx context.Context, viewer, postID, cur string, n int) (Page[Comment], error) {
	post, err := s.GetPost(ctx, viewer, postID)
	if err != nil {
		return Page[Comment]{}, err
	}
	c, err := parseCursor(cur)
	if err != nil {
		return Page[Comment]{}, err
	}
	n = pageSize(n)
	var ct, cid any
	if c != nil {
		ct, cid = c.T, c.ID
	}
	rows, err := s.Pool.Query(ctx, `SELECT c.id, c.body, c.created_at, ca.user_id, ca.handle, ca.display_name, COALESCE(ca.avatar_media_id::text, ''), ca.is_private
		FROM social.comments c JOIN social.profiles ca ON ca.user_id = c.author_id
		WHERE c.post_id = $2 AND c.state = 'visible' AND ca.account_status IN ('active', 'restricted')
		  AND NOT EXISTS (SELECT 1 FROM social.blocks b WHERE (b.blocker_id = c.author_id AND b.blocked_id = $1)
		                                                 OR (b.blocker_id = $1 AND b.blocked_id = c.author_id))
		  AND ($3::timestamptz IS NULL OR (c.created_at, c.id) > ($3, $4::uuid))
		ORDER BY c.created_at, c.id LIMIT $5`, viewer, postID, ct, cid, n+1)
	if err != nil {
		return Page[Comment]{}, err
	}
	defer rows.Close()
	out := Page[Comment]{Data: []Comment{}}
	for rows.Next() {
		var x Comment
		var avatar string
		if err := rows.Scan(&x.ID, &x.Body, &x.CreatedAt, &x.Author.ID, &x.Author.Handle, &x.Author.DisplayName, &avatar, &x.Author.IsPrivate); err != nil {
			return out, err
		}
		x.PostID = postID
		x.Author.AvatarURL = mediaURL(avatar)
		x.CanDelete = x.Author.ID == viewer || post.IsOwner
		x.IsAuthor = x.Author.ID == post.Author.ID
		out.Data = append(out.Data, x)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Data) > n {
		last := out.Data[n-1]
		x := cursor{T: last.CreatedAt, ID: last.ID}.String()
		out.NextCursor = &x
		out.Data = out.Data[:n]
	}
	return out, nil
}

// AddComment comments on a visible post, honouring the author's comments_from setting.
func (s *Service) AddComment(ctx context.Context, viewer, postID, body string) (Comment, error) {
	if err := restricted(ctx); err != nil {
		return Comment{}, err
	}
	body = strings.TrimSpace(body)
	if l := utf8.RuneCountInString(body); l == 0 || l > 1000 {
		return Comment{}, apperr.Validation(apperr.FieldError{Field: "body", Code: "invalid", Detail: "1-1000 characters"})
	}
	if err := s.limit(ctx, RuleComments, viewer); err != nil {
		return Comment{}, err
	}
	post, err := s.GetPost(ctx, viewer, postID)
	if err != nil {
		return Comment{}, err
	}
	if !post.CommentsAllowed {
		e := apperr.New(apperr.Forbidden, "comments are limited by the author")
		e.Extra = map[string]any{"reason": "comments_limited"}
		return Comment{}, e
	}
	id := ids.New()
	var created time.Time
	if err := s.Pool.QueryRow(ctx, `INSERT INTO social.comments (id, post_id, author_id, body) VALUES ($1, $2, $3, $4) RETURNING created_at`,
		id, postID, viewer, body).Scan(&created); err != nil {
		return Comment{}, err
	}
	pr, err := loadProfile(ctx, s.Pool, "user_id = $1", viewer)
	if err != nil {
		return Comment{}, err
	}
	return Comment{ID: id, PostID: postID, Author: pr.author(), Body: body, CreatedAt: created, CanDelete: true, IsAuthor: post.IsOwner}, nil
}

// DeleteComment deletes a comment written by the viewer, or any comment on the viewer's own post.
func (s *Service) DeleteComment(ctx context.Context, viewer, commentID string) error {
	if !ids.Valid(commentID) {
		return errNotFound
	}
	var author, postAuthor string
	err := s.Pool.QueryRow(ctx, `SELECT c.author_id, p.author_id FROM social.comments c JOIN social.posts p ON p.id = c.post_id
		WHERE c.id = $1 AND c.state = 'visible'`, commentID).Scan(&author, &postAuthor)
	if err != nil {
		return errNotFound
	}
	if author != viewer && postAuthor != viewer {
		return apperr.New(apperr.Forbidden, "")
	}
	_, err = s.Pool.Exec(ctx, `UPDATE social.comments SET state = 'deleted', deleted_at = now() WHERE id = $1 AND state = 'visible'`, commentID)
	return err
}
