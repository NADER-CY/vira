package social

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/NADER-CY/vira/backend/internal/modules/social/contract"
	"github.com/NADER-CY/vira/backend/internal/platform/apperr"
	"github.com/NADER-CY/vira/backend/internal/platform/auth"
	"github.com/NADER-CY/vira/backend/internal/platform/httpx"
	"github.com/NADER-CY/vira/backend/internal/platform/ids"
)

// Routes registers the social endpoints (doc 07 §2: profiles, graph, posts, feed, engagement). Every route needs
// a signed-in session (P2 has no anonymous browsing).
func (s *Service) Routes(r *httpx.Router, session httpx.Middleware) {
	h := func(method, path string, fn http.HandlerFunc) { r.HandleFunc(method, path, fn, session) }
	h(http.MethodGet, "/v1/me/profile", s.hMyProfile)
	h(http.MethodPatch, "/v1/me/profile", s.hUpdateProfile)
	h(http.MethodGet, "/v1/me/privacy", s.hPrivacy)
	h(http.MethodPut, "/v1/me/privacy", s.hSetPrivacy)
	h(http.MethodGet, "/v1/me/blocks", s.hBlocks)
	h(http.MethodGet, "/v1/me/follow-requests", s.hFollowRequests)
	h(http.MethodPost, "/v1/me/follow-requests/{id}/approve", s.hDecideRequest(true))
	h(http.MethodPost, "/v1/me/follow-requests/{id}/decline", s.hDecideRequest(false))
	h(http.MethodDelete, "/v1/me/followers/{id}", s.hRemoveFollower)

	h(http.MethodGet, "/v1/users/search", s.hSearch)
	h(http.MethodGet, "/v1/handles/{handle}", s.hProfileByHandle)
	h(http.MethodGet, "/v1/users/{id}", s.hProfileByID)
	h(http.MethodGet, "/v1/users/{id}/posts", s.hUserPosts)
	h(http.MethodGet, "/v1/users/{id}/followers", s.hFollowers)
	h(http.MethodGet, "/v1/users/{id}/following", s.hFollowing)
	h(http.MethodPost, "/v1/users/{id}/follow", s.hFollow)
	h(http.MethodDelete, "/v1/users/{id}/follow", s.hUnfollow)
	h(http.MethodPost, "/v1/users/{id}/block", s.hBlock)
	h(http.MethodDelete, "/v1/users/{id}/block", s.hUnblock)

	h(http.MethodGet, "/v1/feed/home", s.hFeed(s.HomeFeed))
	h(http.MethodGet, "/v1/feed/following", s.hFeed(s.FollowingFeed))
	h(http.MethodPost, "/v1/posts", s.hCreatePost)
	h(http.MethodGet, "/v1/posts/{id}", s.hGetPost)
	h(http.MethodPatch, "/v1/posts/{id}", s.hUpdatePost)
	h(http.MethodDelete, "/v1/posts/{id}", s.hDeletePost)
	h(http.MethodPut, "/v1/posts/{id}/like", s.hLike(true))
	h(http.MethodDelete, "/v1/posts/{id}/like", s.hLike(false))
	h(http.MethodGet, "/v1/posts/{id}/comments", s.hComments)
	h(http.MethodPost, "/v1/posts/{id}/comments", s.hAddComment)
	h(http.MethodDelete, "/v1/comments/{id}", s.hDeleteComment)
}

func viewer(r *http.Request) string {
	p, _ := auth.FromContext(r.Context())
	return p.Subject
}

func metaOf(r *http.Request) Meta {
	return Meta{IP: clientIP(r), UserAgent: r.UserAgent(), TraceID: httpx.TraceID(r.Context())}
}

func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i > 0 {
		host = host[:i]
	}
	return strings.Trim(host, "[]")
}

func limitParam(r *http.Request) int {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return n
}

// ifMatch parses the If-Match header carrying a resource version (doc 07 §1: optimistic concurrency).
func ifMatch(r *http.Request) (int, error) {
	v := strings.Trim(strings.TrimPrefix(r.Header.Get("If-Match"), "W/"), `"`)
	if v == "" {
		return 0, apperr.New(apperr.PreconditionRequired, "")
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, apperr.New(apperr.PreconditionFailed, "If-Match must be the resource version")
	}
	return n, nil
}

func writeVersioned(w http.ResponseWriter, version int, v any) {
	w.Header().Set("ETag", `"`+strconv.Itoa(version)+`"`)
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (s *Service) hMyProfile(w http.ResponseWriter, r *http.Request) {
	p, err := s.ProfileByID(r.Context(), viewer(r), viewer(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	writeVersioned(w, p.Version, p)
}

func (s *Service) hUpdateProfile(w http.ResponseWriter, r *http.Request) {
	version, err := ifMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var in ProfileUpdate
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := s.UpdateProfile(r.Context(), viewer(r), version, in, metaOf(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	writeVersioned(w, p.Version, p)
}

func (s *Service) hPrivacy(w http.ResponseWriter, r *http.Request) {
	p, err := s.Privacy(r.Context(), viewer(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (s *Service) hSetPrivacy(w http.ResponseWriter, r *http.Request) {
	var in PrivacySetting
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := s.SetPrivacy(r.Context(), viewer(r), in, metaOf(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (s *Service) hBlocks(w http.ResponseWriter, r *http.Request) {
	list, err := s.Blocks(r.Context(), viewer(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (s *Service) hFollowRequests(w http.ResponseWriter, r *http.Request) {
	page, err := s.FollowRequests(r.Context(), viewer(r), r.URL.Query().Get("cursor"), limitParam(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (s *Service) hDecideRequest(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.DecideFollowRequest(r.Context(), viewer(r), r.PathValue("id"), approve); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Service) hRemoveFollower(w http.ResponseWriter, r *http.Request) {
	if err := s.RemoveFollower(r.Context(), viewer(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hSearch(w http.ResponseWriter, r *http.Request) {
	list, err := s.SearchUsers(r.Context(), viewer(r), r.URL.Query().Get("q"), limitParam(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (s *Service) hProfileByHandle(w http.ResponseWriter, r *http.Request) {
	p, err := s.ProfileByHandle(r.Context(), viewer(r), r.PathValue("handle"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (s *Service) hProfileByID(w http.ResponseWriter, r *http.Request) {
	p, err := s.ProfileByID(r.Context(), viewer(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (s *Service) hUserPosts(w http.ResponseWriter, r *http.Request) {
	page, err := s.UserPosts(r.Context(), viewer(r), r.PathValue("id"), r.URL.Query().Get("cursor"), limitParam(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (s *Service) hFollowers(w http.ResponseWriter, r *http.Request) {
	page, err := s.Followers(r.Context(), viewer(r), r.PathValue("id"), r.URL.Query().Get("cursor"), limitParam(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (s *Service) hFollowing(w http.ResponseWriter, r *http.Request) {
	page, err := s.Following(r.Context(), viewer(r), r.PathValue("id"), r.URL.Query().Get("cursor"), limitParam(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (s *Service) hFollow(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("id")
	if !ids.Valid(target) {
		httpx.WriteError(w, r, errNotFound)
		return
	}
	res, err := s.Follow(r.Context(), viewer(r), target, metaOf(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (s *Service) hUnfollow(w http.ResponseWriter, r *http.Request) {
	if err := s.Unfollow(r.Context(), viewer(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hBlock(w http.ResponseWriter, r *http.Request) {
	if err := s.Block(r.Context(), viewer(r), r.PathValue("id"), metaOf(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hUnblock(w http.ResponseWriter, r *http.Request) {
	if err := s.Unblock(r.Context(), viewer(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hFeed(fn func(ctx context.Context, viewer, cur string, n int) (Page[Post], error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := fn(r.Context(), viewer(r), r.URL.Query().Get("cursor"), limitParam(r))
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}

func (s *Service) hCreatePost(w http.ResponseWriter, r *http.Request) {
	var in PostInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := s.CreatePost(r.Context(), viewer(r), in, metaOf(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.Header().Set("Location", "/v1/posts/"+p.ID)
	w.Header().Set("ETag", `"`+strconv.Itoa(p.Version)+`"`)
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (s *Service) hGetPost(w http.ResponseWriter, r *http.Request) {
	p, err := s.GetPost(r.Context(), viewer(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	writeVersioned(w, p.Version, p)
}

func (s *Service) hUpdatePost(w http.ResponseWriter, r *http.Request) {
	version, err := ifMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var in PostUpdate
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := s.UpdatePost(r.Context(), viewer(r), r.PathValue("id"), version, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	writeVersioned(w, p.Version, p)
}

func (s *Service) hDeletePost(w http.ResponseWriter, r *http.Request) {
	if err := s.DeletePost(r.Context(), viewer(r), r.PathValue("id"), metaOf(r)); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) hLike(like bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := s.SetLike(r.Context(), viewer(r), r.PathValue("id"), like)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, st)
	}
}

func (s *Service) hComments(w http.ResponseWriter, r *http.Request) {
	page, err := s.Comments(r.Context(), viewer(r), r.PathValue("id"), r.URL.Query().Get("cursor"), limitParam(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (s *Service) hAddComment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Body string `json:"body"`
	}
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := s.AddComment(r.Context(), viewer(r), r.PathValue("id"), in.Body)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (s *Service) hDeleteComment(w http.ResponseWriter, r *http.Request) {
	if err := s.DeleteComment(r.Context(), viewer(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SearchUsers finds visible accounts by handle prefix (Explore search arrives with discovery; P2 needs it for
// finding people to follow).
func (s *Service) SearchUsers(ctx context.Context, viewer, q string, n int) ([]Author, error) {
	q = strings.TrimSpace(q)
	if len([]rune(q)) < 2 {
		return []Author{}, nil
	}
	idsFound, err := contract.SearchUserIDs(ctx, s.Pool, q, pageSize(n)*2)
	if err != nil {
		return nil, err
	}
	out := []Author{}
	if len(idsFound) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT pr.user_id, pr.handle, pr.display_name, COALESCE(pr.avatar_media_id::text, ''), pr.is_private
		FROM social.profiles pr WHERE pr.user_id = ANY($2::uuid[]) AND pr.account_status IN ('active', 'restricted')
		  AND NOT EXISTS (SELECT 1 FROM social.blocks b WHERE (b.blocker_id = pr.user_id AND b.blocked_id = $1)
		                                                 OR (b.blocker_id = $1 AND b.blocked_id = pr.user_id))
		ORDER BY pr.handle_norm LIMIT $3`, viewer, idsFound, pageSize(n))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Author
		var avatar string
		if err := rows.Scan(&a.ID, &a.Handle, &a.DisplayName, &avatar, &a.IsPrivate); err != nil {
			return nil, err
		}
		a.AvatarURL = mediaURL(avatar)
		out = append(out, a)
	}
	return out, rows.Err()
}
