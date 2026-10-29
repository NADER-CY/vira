-- 0005 social graph, profiles, posts, engagement (P2) and basic image media (doc 06 §4.2).
-- Counters are computed from indexed tables in P2; separate counter tables arrive when load requires (doc 04 §9).

CREATE TABLE media.assets (
  id           uuid        PRIMARY KEY,
  owner_id     uuid        NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
  kind         text        NOT NULL DEFAULT 'image' CHECK (kind IN ('image')),
  purpose      text        NOT NULL CHECK (purpose IN ('post', 'avatar')),
  content_type text        NOT NULL CHECK (content_type IN ('image/jpeg')),
  bytes        bigint      NOT NULL CHECK (bytes > 0),
  width        int         NOT NULL CHECK (width > 0),
  height       int         NOT NULL CHECK (height > 0),
  sha256       bytea       NOT NULL,
  object_key   text        NOT NULL UNIQUE,
  state        text        NOT NULL DEFAULT 'ready' CHECK (state IN ('ready', 'deleted')),
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX assets_owner ON media.assets (owner_id, created_at DESC);

CREATE TABLE social.profiles (
  user_id          uuid        PRIMARY KEY REFERENCES identity.users (id) ON DELETE CASCADE,
  handle           text        NOT NULL CHECK (handle ~ '^[A-Za-z0-9_.]{3,30}$'),
  handle_norm      text        NOT NULL UNIQUE,
  display_name     text        NOT NULL DEFAULT '' CHECK (char_length(display_name) <= 50),
  bio              text        NOT NULL DEFAULT '' CHECK (char_length(bio) <= 300),
  website          text        NOT NULL DEFAULT '' CHECK (char_length(website) <= 200),
  avatar_media_id  uuid        REFERENCES media.assets (id) ON DELETE SET NULL,
  is_private       boolean     NOT NULL DEFAULT false,
  comments_from    text        NOT NULL DEFAULT 'everyone' CHECK (comments_from IN ('everyone', 'followers', 'nobody')),
  hide_like_counts boolean     NOT NULL DEFAULT false,
  -- Preferences of features delivered in later phases (stories, mentions, product tags, messages): stored now,
  -- enforced when the feature ships.
  settings         jsonb       NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(settings) = 'object'),
  account_status   text        NOT NULL DEFAULT 'active'
                   CHECK (account_status IN ('active', 'restricted', 'suspended', 'deletion_pending', 'deleted')),
  version          int         NOT NULL DEFAULT 1,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX profiles_handle_prefix ON social.profiles (handle_norm text_pattern_ops);

CREATE TABLE social.follows (
  follower_id uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  followee_id uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (follower_id, followee_id),
  CHECK (follower_id <> followee_id)
);
CREATE INDEX follows_followee ON social.follows (followee_id, created_at DESC, follower_id);
CREATE INDEX follows_follower ON social.follows (follower_id, created_at DESC, followee_id);

CREATE TABLE social.follow_requests (
  requester_id uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  target_id    uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  created_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (requester_id, target_id),
  CHECK (requester_id <> target_id)
);
CREATE INDEX follow_requests_target ON social.follow_requests (target_id, created_at DESC);

CREATE TABLE social.blocks (
  blocker_id uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  blocked_id uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (blocker_id, blocked_id),
  CHECK (blocker_id <> blocked_id)
);
CREATE INDEX blocks_blocked ON social.blocks (blocked_id, blocker_id);

CREATE TABLE social.posts (
  id             uuid        PRIMARY KEY,
  author_id      uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  caption        text        NOT NULL DEFAULT '' CHECK (char_length(caption) <= 2200),
  visibility     text        NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'followers')),
  state          text        NOT NULL DEFAULT 'published' CHECK (state IN ('published', 'removed', 'deleted')),
  removed_reason text,
  version        int         NOT NULL DEFAULT 1,
  published_at   timestamptz NOT NULL DEFAULT now(),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  edited_at      timestamptz
);
CREATE INDEX posts_author_recent ON social.posts (author_id, published_at DESC, id DESC) WHERE state = 'published';
CREATE INDEX posts_recent ON social.posts (published_at DESC, id DESC) WHERE state = 'published';

CREATE TABLE social.post_media (
  post_id  uuid     NOT NULL REFERENCES social.posts (id) ON DELETE CASCADE,
  position smallint NOT NULL CHECK (position BETWEEN 0 AND 9),
  media_id uuid     NOT NULL UNIQUE REFERENCES media.assets (id),
  PRIMARY KEY (post_id, position)
);

CREATE TABLE social.likes (
  post_id    uuid        NOT NULL REFERENCES social.posts (id) ON DELETE CASCADE,
  user_id    uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (post_id, user_id)
);
CREATE INDEX likes_user ON social.likes (user_id, created_at DESC);

CREATE TABLE social.comments (
  id         uuid        PRIMARY KEY,
  post_id    uuid        NOT NULL REFERENCES social.posts (id) ON DELETE CASCADE,
  author_id  uuid        NOT NULL REFERENCES social.profiles (user_id) ON DELETE CASCADE,
  body       text        NOT NULL CHECK (char_length(body) BETWEEN 1 AND 1000),
  state      text        NOT NULL DEFAULT 'visible' CHECK (state IN ('visible', 'deleted', 'removed')),
  created_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX comments_post ON social.comments (post_id, created_at, id) WHERE state = 'visible';
CREATE INDEX comments_author ON social.comments (author_id, created_at DESC);

GRANT USAGE ON SCHEMA social, media TO app_public, app_admin, app_worker;
GRANT SELECT, INSERT, UPDATE ON media.assets TO app_public;
GRANT SELECT, INSERT, UPDATE ON social.profiles, social.posts, social.comments TO app_public;
GRANT SELECT, INSERT, DELETE ON social.follows, social.follow_requests, social.blocks, social.likes, social.post_media TO app_public;
GRANT SELECT ON ALL TABLES IN SCHEMA social, media TO app_admin;
GRANT UPDATE ON social.profiles, social.posts, social.comments TO app_admin;
GRANT SELECT, UPDATE ON social.profiles TO app_worker;
