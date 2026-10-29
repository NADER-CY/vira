"""Phase 2 additions to the OpenAPI contracts (identity, social, media, moderation; admin users and reports).

Imported by gen_openapi.py; edit here, then run `python3 scripts/gen_openapi.py`.
"""

def _ref(n): return {"$ref": f"#/components/schemas/{n}"}
def _problem(d): return {"description": d, "content": {"application/problem+json": {"schema": _ref("Problem")}}}
def _ok(schema, desc="OK", headers=None):
    r = {"description": desc, "content": {"application/json": {"schema": schema}}}
    if headers:
        r["headers"] = headers
    return r
def _body(schema): return {"required": True, "content": {"application/json": {"schema": schema}}}
def _path(name="id", desc="UUID"): return {"name": name, "in": "path", "required": True, "schema": {"type": "string"}, "description": desc}
def _q(name, schema, desc=None):
    p = {"name": name, "in": "query", "required": False, "schema": schema}
    if desc:
        p["description"] = desc
    return p
PAGING = [_q("cursor", {"type": "string"}, "Opaque cursor from next_cursor"), _q("limit", {"type": "integer", "minimum": 1, "maximum": 50})]
def _page(item): return {"type": "object", "required": ["data", "next_cursor"], "properties": {
    "data": {"type": "array", "items": _ref(item)}, "next_cursor": {"type": ["string", "null"]}}}
def _list(item): return {"type": "object", "required": ["data"], "properties": {"data": {"type": "array", "items": _ref(item)}}}
IF_MATCH = {"name": "If-Match", "in": "header", "required": True, "schema": {"type": "string"}, "description": "Current resource version (ETag)"}
IDEM = {"name": "Idempotency-Key", "in": "header", "required": False, "schema": {"type": "string", "maxLength": 64},
        "description": "Retries with the same key return the original response (doc 07 §1.3)"}
ETAG = {"ETag": {"schema": {"type": "string"}, "description": "Resource version for If-Match"}}

STD = {"401": _problem("Unauthenticated or session revoked"), "403": _problem("Forbidden / account restricted"),
       "404": _problem("Not found (also: blocked or not visible)"), "422": _problem("Validation failed"),
       "429": _problem("Rate limited (retry_after)")}
def _std(*codes): return {c: STD[c] for c in codes}

def public_paths(paths, schemas):
    sec = [{"bearer": []}]
    def op(method, path, oid, tag, summary, responses, params=None, body=None, security=sec):
        o = {"operationId": oid, "tags": [tag], "summary": summary, "security": security, "responses": responses}
        if params:
            o["parameters"] = params
        if body:
            o["requestBody"] = body
        paths.setdefault(path, {})[method] = o

    # ---- auth (doc 07 §4.1) ----
    op("post", "/v1/auth/otp", "requestOtp", "auth", "Send a 6-digit sign-in code by SMS (never reveals whether the number has an account)",
       {"202": _ok(_ref("OtpChallenge")), "422": STD["422"], "429": STD["429"], "503": _problem("SMS provider unavailable")},
       params=[{"name": "X-Device-Id", "in": "header", "required": False, "schema": {"type": "string"}}],
       body=_body(_ref("OtpRequest")), security=[])
    op("post", "/v1/auth/otp/verify", "verifyOtp", "auth", "Verify the code; signs in or creates the account; issues tokens",
       {"200": _ok(_ref("SignIn")), "401": _problem("otp_invalid / otp_expired"), "403": _problem("account_restricted"),
        "422": STD["422"], "429": _problem("otp_locked or rate limited")}, body=_body(_ref("OtpVerifyRequest")), security=[])
    op("post", "/v1/auth/token/refresh", "refreshToken", "auth",
       "Rotate the refresh token (a reused token revokes the session); devices with a key send X-Vira-PoP",
       {"200": _ok(_ref("Tokens")), "401": STD["401"], "429": STD["429"]},
       params=[{"name": "X-Vira-PoP", "in": "header", "required": False, "schema": {"type": "string"},
                "description": "<unix seconds>.<base64url ECDSA P-256 signature of VIRA-REFRESH\\n<ts>\\n<refresh token>>"}],
       body=_body({"type": "object", "required": ["refresh_token"], "properties": {"refresh_token": {"type": "string"}}}), security=[])
    op("post", "/v1/auth/recovery", "requestRecovery", "auth", "File an account-recovery request for a lost number (foundation)",
       {"202": _ok({"type": "object", "properties": {"status": {"type": "string"}}}), "422": STD["422"], "429": STD["429"]},
       body=_body(_ref("RecoveryRequest")), security=[])
    op("post", "/v1/auth/logout", "logout", "auth", "Revoke the current session", {"204": {"description": "Signed out"}, "401": STD["401"]})
    op("get", "/v1/me/account", "getAccount", "account", "Account status, masked phone, deletion date",
       {"200": _ok(_ref("Account")), "401": STD["401"]})
    op("get", "/v1/me/sessions", "listSessions", "account", "Active sessions (devices)", {"200": _ok(_list("Session")), "401": STD["401"]})
    op("delete", "/v1/me/sessions/{id}", "revokeSession", "account", "Sign out one device", {"204": {"description": "Revoked"}, **_std("401", "404")},
       params=[_path()])
    op("post", "/v1/me/sessions/revoke-others", "revokeOtherSessions", "account", "Sign out every other device",
       {"204": {"description": "Revoked"}, "401": STD["401"]})
    op("post", "/v1/me/deletion", "requestDeletion", "account", "Start the 30-day deletion grace period (signing in again cancels it)",
       {"202": _ok({"type": "object", "properties": {"status": {"type": "string"}, "effective_at": {"type": "string", "format": "date-time"}}}),
        "401": STD["401"], "409": _problem("Already requested"), "422": STD["422"]},
       body=_body({"type": "object", "required": ["confirm"], "properties": {"confirm": {"type": "boolean", "const": True}}}))

    # ---- profiles, privacy, graph ----
    op("get", "/v1/me/profile", "getMyProfile", "profiles", "Own profile with privacy settings and version",
       {"200": _ok(_ref("Profile"), headers=ETAG), "401": STD["401"]})
    op("patch", "/v1/me/profile", "updateMyProfile", "profiles", "Edit own profile (handle, name, bio, website, avatar)",
       {"200": _ok(_ref("Profile"), headers=ETAG), **_std("401", "422"), "412": _problem("Version mismatch"), "428": _problem("If-Match required")},
       params=[IF_MATCH], body=_body(_ref("ProfileUpdate")))
    op("get", "/v1/me/privacy", "getPrivacy", "profiles", "Privacy settings", {"200": _ok(_ref("PrivacySetting")), "401": STD["401"]})
    op("put", "/v1/me/privacy", "setPrivacy", "profiles", "Replace privacy settings (making an account public approves pending requests)",
       {"200": _ok(_ref("PrivacySetting")), **_std("401", "422")}, body=_body(_ref("PrivacySetting")))
    op("get", "/v1/me/blocks", "listBlocks", "graph", "Accounts I blocked", {"200": _ok(_list("ListEntry")), "401": STD["401"]})
    op("get", "/v1/me/follow-requests", "listFollowRequests", "graph", "Pending follow requests to me",
       {"200": _ok(_page("ListEntry")), **_std("401", "422")}, params=PAGING)
    for action in ("approve", "decline"):
        op("post", "/v1/me/follow-requests/{id}/" + action, action + "FollowRequest", "graph", action.capitalize() + " a follow request",
           {"204": {"description": "Done"}, **_std("401", "404")}, params=[_path(desc="Requester user id")])
    op("delete", "/v1/me/followers/{id}", "removeFollower", "graph", "Remove a follower", {"204": {"description": "Removed"}, **_std("401", "404")},
       params=[_path()])
    op("get", "/v1/users/search", "searchUsers", "profiles", "Find accounts by handle prefix",
       {"200": _ok(_list("Author")), "401": STD["401"]}, params=[_q("q", {"type": "string"}), _q("limit", {"type": "integer"})])
    op("get", "/v1/handles/{handle}", "getProfileByHandle", "profiles", "Public profile by handle",
       {"200": _ok(_ref("Profile")), **_std("401", "404")}, params=[_path("handle", "Handle without @")])
    op("get", "/v1/users/{id}", "getProfile", "profiles", "Public profile by user id", {"200": _ok(_ref("Profile")), **_std("401", "404")},
       params=[_path()])
    op("get", "/v1/users/{id}/posts", "listUserPosts", "posts", "A user's posts (private accounts: followers only)",
       {"200": _ok(_page("Post")), **_std("401", "403", "404", "422")}, params=[_path()] + PAGING)
    for rel in ("followers", "following"):
        op("get", "/v1/users/{id}/" + rel, "list" + rel.capitalize(), "graph", rel.capitalize() + " of a user",
           {"200": _ok(_page("ListEntry")), **_std("401", "403", "404", "422")}, params=[_path()] + PAGING)
    op("post", "/v1/users/{id}/follow", "follow", "graph", "Follow (or request to follow a private account)",
       {"200": _ok(_ref("FollowResult")), **_std("401", "403", "404", "422", "429")}, params=[_path()])
    op("delete", "/v1/users/{id}/follow", "unfollow", "graph", "Unfollow or cancel a request", {"204": {"description": "Done"}, **_std("401", "404")},
       params=[_path()])
    op("post", "/v1/users/{id}/block", "block", "graph", "Block (removes follows both ways; neither side sees the other)",
       {"204": {"description": "Blocked"}, **_std("401", "404", "422")}, params=[_path()])
    op("delete", "/v1/users/{id}/block", "unblock", "graph", "Unblock", {"204": {"description": "Unblocked"}, **_std("401", "404")}, params=[_path()])

    # ---- feed, posts, engagement ----
    for name, summary in (("home", "Home feed (برای شما): visible posts, newest first"), ("following", "Following feed: followed accounts and own posts")):
        op("get", "/v1/feed/" + name, name + "Feed", "feed", summary, {"200": _ok(_page("Post")), **_std("401", "422")}, params=PAGING)
    op("post", "/v1/posts", "createPost", "posts", "Publish a post (text and/or up to 10 uploaded images)",
       {"201": _ok(_ref("Post"), headers=ETAG), **_std("401", "403", "422", "429")}, params=[IDEM], body=_body(_ref("PostInput")))
    op("get", "/v1/posts/{id}", "getPost", "posts", "Post detail", {"200": _ok(_ref("Post"), headers=ETAG), **_std("401", "404")}, params=[_path()])
    op("patch", "/v1/posts/{id}", "updatePost", "posts", "Edit own post (caption, visibility)",
       {"200": _ok(_ref("Post"), headers=ETAG), **_std("401", "403", "404", "422"), "412": _problem("Version mismatch"),
        "428": _problem("If-Match required")}, params=[_path(), IF_MATCH], body=_body(_ref("PostUpdate")))
    op("delete", "/v1/posts/{id}", "deletePost", "posts", "Delete own post", {"204": {"description": "Deleted"}, **_std("401", "403", "404")},
       params=[_path()])
    op("put", "/v1/posts/{id}/like", "likePost", "engagement", "Like (idempotent)", {"200": _ok(_ref("LikeState")), **_std("401", "403", "404", "429")},
       params=[_path()])
    op("delete", "/v1/posts/{id}/like", "unlikePost", "engagement", "Unlike (idempotent)", {"200": _ok(_ref("LikeState")), **_std("401", "404")},
       params=[_path()])
    op("get", "/v1/posts/{id}/comments", "listComments", "engagement", "Comments, oldest first",
       {"200": _ok(_page("Comment")), **_std("401", "404", "422")}, params=[_path()] + PAGING)
    op("post", "/v1/posts/{id}/comments", "addComment", "engagement", "Comment (honours the author's comments_from setting)",
       {"201": _ok(_ref("Comment")), **_std("401", "403", "404", "422", "429")}, params=[_path(), IDEM],
       body=_body({"type": "object", "required": ["body"], "properties": {"body": {"type": "string", "minLength": 1, "maxLength": 1000}}}))
    op("delete", "/v1/comments/{id}", "deleteComment", "engagement", "Delete own comment, or any comment on own post",
       {"204": {"description": "Deleted"}, **_std("401", "403", "404")}, params=[_path()])

    # ---- media, reports ----
    op("post", "/v1/media/images", "uploadImage", "media", "Upload an image (JPEG/PNG/GIF/WebP ≤ 10 MB; re-encoded, metadata stripped)",
       {"201": _ok(_ref("UploadedImage")), **_std("401", "403", "422", "429")},
       body={"required": True, "content": {"multipart/form-data": {"schema": {"type": "object", "required": ["file", "purpose"], "properties": {
           "file": {"type": "string", "format": "binary"}, "purpose": {"type": "string", "enum": ["post", "avatar"]}}}}}})
    op("get", "/v1/media/{id}", "getMedia", "media", "Image bytes (visible to the uploader and to viewers of the post/profile)",
       {"200": {"description": "image/jpeg", "content": {"image/jpeg": {"schema": {"type": "string", "format": "binary"}}}}, **_std("401", "404")},
       params=[_path()])
    op("post", "/v1/reports", "fileReport", "moderation", "Report a post, comment or account",
       {"201": _ok({"type": "object", "required": ["id", "state"], "properties": {"id": {"type": "string"}, "state": {"type": "string"},
                                                                              "created_at": {"type": "string", "format": "date-time"}}}),
        **_std("401", "404", "422", "429")}, body=_body(_ref("ReportRequest")))

    s = schemas
    s["OtpRequest"] = {"type": "object", "required": ["phone"], "properties": {
        "phone": {"type": "string", "description": "Iranian mobile; Persian/Arabic/ASCII digits, with or without +98 / 0"},
        "purpose": {"type": "string", "enum": ["login"]}}}
    s["OtpChallenge"] = {"type": "object", "required": ["challenge_id", "expires_in", "resend_after", "code_length"], "properties": {
        "challenge_id": {"type": "string"}, "expires_in": {"type": "integer"}, "resend_after": {"type": "integer"},
        "code_length": {"type": "integer", "const": 6}}}
    s["Device"] = {"type": "object", "required": ["id", "platform"], "properties": {
        "id": {"type": "string", "minLength": 8, "maxLength": 128}, "platform": {"type": "string", "enum": ["android", "ios", "web"]},
        "model": {"type": "string", "maxLength": 100}, "app_version": {"type": "string", "maxLength": 40},
        "public_key": {"type": "string", "description": "base64 SPKI of a P-256 key held by the device (proof of possession)"}}}
    s["Consents"] = {"type": "object", "required": ["terms", "privacy", "age_confirmation", "text_version"],
        "description": "Explicit, unticked-by-default choices (doc 19 UC-5); all must be true",
        "properties": {"terms": {"type": "boolean"}, "privacy": {"type": "boolean"}, "age_confirmation": {"type": "boolean"},
                       "text_version": {"type": "string", "maxLength": 40}}}
    s["OtpVerifyRequest"] = {"type": "object", "required": ["challenge_id", "code", "device", "consents"], "properties": {
        "challenge_id": {"type": "string"}, "code": {"type": "string", "description": "6 digits (Persian digits accepted)"},
        "device": _ref("Device"), "intends_to_sell": {"type": "boolean", "description": "Account-type choice; seller onboarding is P4, same identity"},
        "consents": _ref("Consents")}}
    s["Tokens"] = {"type": "object", "required": ["access_token", "token_type", "expires_in", "refresh_token", "session_id"], "properties": {
        "access_token": {"type": "string"}, "token_type": {"type": "string", "const": "Bearer"}, "expires_in": {"type": "integer"},
        "refresh_token": {"type": "string"}, "session_id": {"type": "string"}}}
    s["SignIn"] = {"allOf": [_ref("Tokens"), {"type": "object", "required": ["user_id", "handle", "is_new"], "properties": {
        "user_id": {"type": "string"}, "handle": {"type": "string"}, "is_new": {"type": "boolean"}}}]}
    s["RecoveryRequest"] = {"type": "object", "required": ["old_phone", "new_phone"], "properties": {
        "old_phone": {"type": "string"}, "new_phone": {"type": "string"}, "note": {"type": "string", "maxLength": 500}}}
    s["Account"] = {"type": "object", "required": ["user_id", "status", "phone_masked"], "properties": {
        "user_id": {"type": "string"}, "status": {"type": "string", "enum": ["active", "restricted", "suspended", "deletion_pending", "deleted"]},
        "phone_masked": {"type": "string"}, "intends_to_sell": {"type": "boolean"}, "created_at": {"type": "string", "format": "date-time"},
        "deletion_effective_at": {"type": ["string", "null"], "format": "date-time"}}}
    s["Session"] = {"type": "object", "required": ["id", "platform", "created_at", "last_used_at", "current"], "properties": {
        "id": {"type": "string"}, "platform": {"type": "string"}, "model": {"type": "string"}, "app_version": {"type": "string"},
        "created_at": {"type": "string", "format": "date-time"}, "last_used_at": {"type": "string", "format": "date-time"}, "current": {"type": "boolean"}}}
    s["Author"] = {"type": "object", "required": ["id", "handle", "display_name", "is_private"], "properties": {
        "id": {"type": "string"}, "handle": {"type": "string"}, "display_name": {"type": "string"},
        "avatar_url": {"type": "string", "description": "API path of the avatar image"}, "is_private": {"type": "boolean"}}}
    s["Stats"] = {"type": "object", "required": ["posts", "followers", "following"], "properties": {
        "posts": {"type": "integer"}, "followers": {"type": "integer"}, "following": {"type": "integer"}}}
    s["Relationship"] = {"type": "object", "required": ["is_self", "following", "requested", "follows_you", "blocking"], "properties": {
        k: {"type": "boolean"} for k in ("is_self", "following", "requested", "follows_you", "blocking")}}
    s["PrivacySetting"] = {"type": "object", "required": ["is_private", "comments_from", "hide_like_counts"], "properties": {
        "is_private": {"type": "boolean"}, "comments_from": {"type": "string", "enum": ["everyone", "followers", "nobody"]},
        "hide_like_counts": {"type": "boolean"},
        "settings": {"type": "object", "additionalProperties": True,
                     "description": "Preferences of later-phase features (story_replies_from, mentions_from, messages_from, live_comments_from, allow_product_tags, show_activity, allow_reshare), stored now and enforced when each feature ships"}}}
    s["Profile"] = {"allOf": [_ref("Author"), {"type": "object", "required": ["bio", "website", "stats", "relationship", "can_view_posts"], "properties": {
        "bio": {"type": "string"}, "website": {"type": "string"}, "stats": _ref("Stats"), "relationship": _ref("Relationship"),
        "can_view_posts": {"type": "boolean"}, "version": {"type": "integer", "description": "own profile only"},
        "privacy": _ref("PrivacySetting")}}]}
    s["ProfileUpdate"] = {"type": "object", "properties": {
        "handle": {"type": "string", "pattern": "^@?[A-Za-z0-9_.]{3,30}$"}, "display_name": {"type": "string", "maxLength": 50},
        "bio": {"type": "string", "maxLength": 300}, "website": {"type": "string", "maxLength": 200},
        "avatar_media_id": {"type": "string", "description": "uploaded image (purpose avatar); empty string removes"}}}
    s["FollowResult"] = {"type": "object", "required": ["state"], "properties": {"state": {"type": "string", "enum": ["following", "requested"]}}}
    s["ListEntry"] = {"allOf": [_ref("Author"), {"type": "object", "required": ["since", "following"], "properties": {
        "since": {"type": "string", "format": "date-time"}, "following": {"type": "boolean", "description": "the viewer follows this account"}}}]}
    s["MediaInfo"] = {"type": "object", "required": ["id", "url", "width", "height"], "properties": {
        "id": {"type": "string"}, "url": {"type": "string"}, "width": {"type": "integer"}, "height": {"type": "integer"}}}
    s["UploadedImage"] = {"allOf": [_ref("MediaInfo"), {"type": "object", "required": ["purpose"], "properties": {"purpose": {"type": "string"}}}]}
    s["PostInput"] = {"type": "object", "properties": {
        "caption": {"type": "string", "maxLength": 2200}, "visibility": {"type": "string", "enum": ["public", "followers"]},
        "media_ids": {"type": "array", "maxItems": 10, "items": {"type": "string"}}}}
    s["PostUpdate"] = {"type": "object", "properties": {
        "caption": {"type": "string", "maxLength": 2200}, "visibility": {"type": "string", "enum": ["public", "followers"]}}}
    s["Post"] = {"type": "object", "required": ["id", "author", "caption", "visibility", "media", "like_count", "comment_count", "viewer_liked",
                                               "is_owner", "comments_allowed", "published_at", "version"], "properties": {
        "id": {"type": "string"}, "author": _ref("Author"), "caption": {"type": "string"},
        "visibility": {"type": "string", "enum": ["public", "followers"]}, "media": {"type": "array", "items": _ref("MediaInfo")},
        "like_count": {"type": ["integer", "null"], "description": "null when the author hides like counts"},
        "comment_count": {"type": "integer"}, "viewer_liked": {"type": "boolean"}, "is_owner": {"type": "boolean"},
        "comments_allowed": {"type": "boolean"}, "published_at": {"type": "string", "format": "date-time"},
        "edited_at": {"type": ["string", "null"], "format": "date-time"}, "version": {"type": "integer"},
        "viewer_follows_author": {"type": "boolean"}}}
    s["LikeState"] = {"type": "object", "required": ["liked", "like_count"], "properties": {
        "liked": {"type": "boolean"}, "like_count": {"type": ["integer", "null"]}}}
    s["Comment"] = {"type": "object", "required": ["id", "post_id", "author", "body", "created_at", "can_delete", "is_post_author"], "properties": {
        "id": {"type": "string"}, "post_id": {"type": "string"}, "author": _ref("Author"), "body": {"type": "string"},
        "created_at": {"type": "string", "format": "date-time"}, "can_delete": {"type": "boolean"}, "is_post_author": {"type": "boolean"}}}
    s["ReportRequest"] = {"type": "object", "required": ["target_type", "target_id", "reason"], "properties": {
        "target_type": {"type": "string", "enum": ["post", "comment", "user"]}, "target_id": {"type": "string"},
        "reason": {"type": "string", "enum": ["spam", "harassment", "hate", "violence", "nudity", "scam", "counterfeit", "misinformation", "self_harm", "other"]},
        "details": {"type": "string", "maxLength": 1000}}}


def admin_paths(paths, schemas):
    sec = [{"session": []}]
    csrf = {"$ref": "#/components/parameters/CSRF"}
    def op(method, path, oid, tag, perm, summary, responses, params=None, body=None):
        o = {"operationId": oid, "tags": [tag], "security": sec, "x-permission": perm, "summary": summary,
             "responses": {**responses, "401": _problem("Unauthenticated"), "403": _problem("Forbidden")}}
        if params:
            o["parameters"] = params
        if body:
            o["requestBody"] = body
        paths.setdefault(path, {})[method] = o
    page = [_q("cursor", {"type": "string"}), _q("limit", {"type": "integer", "minimum": 1, "maximum": 100})]
    op("get", "/v1/admin/users", "adminListUsers", "users", "users.read", "Users (q: handle prefix, user id or exact phone; status filter)",
       {"200": _ok(_page("AdminUser")), "422": _problem("Bad filter")},
       params=[_q("q", {"type": "string"}), _q("status", {"type": "string", "enum": ["active", "restricted", "suspended", "deletion_pending", "deleted"]})] + page)
    op("get", "/v1/admin/users/{id}", "adminGetUser", "users", "users.read", "User detail: profile, status history, sessions",
       {"200": _ok(_ref("AdminUserDetail")), "404": _problem("Not found")}, params=[_path()])
    op("post", "/v1/admin/users/{id}/status", "adminSetUserStatus", "users", "users.write",
       "Change account status (active, restricted, suspended); suspension revokes all sessions",
       {"200": _ok({"type": "object", "properties": {"id": {"type": "string"}, "previous_status": {"type": "string"}, "status": {"type": "string"}}}),
        "404": _problem("Not found"), "409": _problem("Unchanged"), "422": _problem("Validation failed")},
       params=[_path(), csrf], body=_body({"type": "object", "required": ["status", "reason"], "properties": {
           "status": {"type": "string", "enum": ["active", "restricted", "suspended"]}, "reason": {"type": "string", "minLength": 1, "maxLength": 500}}}))
    op("post", "/v1/admin/users/{id}/sessions/revoke", "adminRevokeUserSessions", "users", "users.write", "Sign the user out on every device",
       {"200": _ok({"type": "object", "properties": {"revoked": {"type": "integer"}}}), "404": _problem("Not found")}, params=[_path(), csrf])
    op("get", "/v1/admin/reports", "adminListReports", "reports", "reports.read", "Moderation report queue",
       {"200": _ok(_page("AdminReport")), "422": _problem("Bad filter")},
       params=[_q("state", {"type": "string", "enum": ["open", "actioned", "dismissed"]}),
               _q("target_type", {"type": "string", "enum": ["post", "comment", "user"]})] + page)
    op("post", "/v1/admin/reports/{id}/resolve", "adminResolveReport", "reports", "reports.write",
       "Resolve a report: dismiss, remove content, restrict or suspend the owner (closes other open reports on the target)",
       {"200": _ok(_ref("AdminReport")), "404": _problem("Not found"), "409": _problem("Already resolved"), "422": _problem("Validation failed")},
       params=[_path(), csrf], body=_body({"type": "object", "required": ["resolution", "note"], "properties": {
           "resolution": {"type": "string", "enum": ["dismiss", "remove_content", "restrict_user", "suspend_user"]},
           "note": {"type": "string", "minLength": 1, "maxLength": 500}}}))

    s = schemas
    s["AdminUser"] = {"type": "object", "required": ["id", "handle", "status", "created_at"], "properties": {
        "id": {"type": "string"}, "handle": {"type": "string"}, "display_name": {"type": "string"},
        "phone_masked": {"type": "string", "description": "never the full number"}, "status": {"type": "string"},
        "is_private": {"type": "boolean"}, "created_at": {"type": "string", "format": "date-time"},
        "last_login_at": {"type": ["string", "null"], "format": "date-time"}, "posts": {"type": "integer"}, "followers": {"type": "integer"},
        "open_reports": {"type": "integer"}, "active_sessions": {"type": "integer"}}}
    s["AdminUserDetail"] = {"allOf": [_ref("AdminUser"), {"type": "object", "properties": {
        "bio": {"type": "string"}, "status_reason": {"type": "string"}, "intends_to_sell": {"type": "boolean"}, "following": {"type": "integer"},
        "status_history": {"type": "array", "items": {"type": "object", "properties": {
            "from": {"type": "string"}, "to": {"type": "string"}, "reason": {"type": "string"}, "actor_type": {"type": "string"},
            "created_at": {"type": "string", "format": "date-time"}}}},
        "sessions": {"type": "array", "items": {"type": "object", "properties": {
            "id": {"type": "string"}, "platform": {"type": "string"}, "model": {"type": "string"},
            "created_at": {"type": "string", "format": "date-time"}, "last_used_at": {"type": "string", "format": "date-time"},
            "revoked_at": {"type": ["string", "null"], "format": "date-time"}, "revoke_reason": {"type": "string"}}}}}}]}
    s["AdminReport"] = {"type": "object", "required": ["id", "target_type", "target_id", "reason", "state", "created_at"], "properties": {
        "id": {"type": "string"}, "reporter_id": {"type": "string"}, "reporter_handle": {"type": "string"},
        "target_type": {"type": "string", "enum": ["post", "comment", "user"]}, "target_id": {"type": "string"},
        "target_owner_id": {"type": "string"}, "target_owner_handle": {"type": "string"}, "target_preview": {"type": "string"},
        "reason": {"type": "string"}, "details": {"type": "string"}, "state": {"type": "string", "enum": ["open", "actioned", "dismissed"]},
        "resolution": {"type": "string"}, "resolution_note": {"type": "string"}, "created_at": {"type": "string", "format": "date-time"},
        "resolved_at": {"type": ["string", "null"], "format": "date-time"}, "open_reports_on_target": {"type": "integer"}}}
