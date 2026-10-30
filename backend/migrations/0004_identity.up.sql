-- 0004 identity (P2): users, phone identities (encrypted + blind index), OTP challenges, devices, sessions with
-- rotating refresh tokens, account status history and recovery requests (doc 06 §4.1, doc 07 §4.1, doc 08).
-- Phone numbers never appear in clear text: phone_enc is AES-256-GCM (key outside the database),
-- phone_bidx is HMAC-SHA256 (separate key) for exact lookups, phone_last4 supports masked display.

CREATE TABLE identity.users (
  id                    uuid        PRIMARY KEY,
  status                text        NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'restricted', 'suspended', 'deletion_pending', 'deleted')),
  status_reason         text,
  status_changed_at     timestamptz,
  intends_to_sell       boolean     NOT NULL DEFAULT false, -- account-type choice at sign-up; seller onboarding is P4 (same identity)
  deletion_effective_at timestamptz,
  created_at            timestamptz NOT NULL DEFAULT now(),
  last_login_at         timestamptz
);
CREATE INDEX users_status ON identity.users (status, created_at DESC);

CREATE TABLE identity.phone_identities (
  id          uuid        PRIMARY KEY,
  user_id     uuid        NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
  phone_enc   bytea       NOT NULL,
  phone_bidx  bytea       NOT NULL UNIQUE,
  phone_last4 text        NOT NULL CHECK (phone_last4 ~ '^[0-9]{4}$'),
  verified_at timestamptz NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX phone_identities_user ON identity.phone_identities (user_id);

CREATE TABLE identity.otp_challenges (
  id           uuid        PRIMARY KEY,
  phone_bidx   bytea       NOT NULL,
  phone_enc    bytea       NOT NULL,
  phone_last4  text        NOT NULL,
  purpose      text        NOT NULL DEFAULT 'login' CHECK (purpose IN ('login')),
  code_hash    bytea       NOT NULL,
  salt         bytea       NOT NULL,
  attempts     int         NOT NULL DEFAULT 0,
  max_attempts int         NOT NULL DEFAULT 5,
  state        text        NOT NULL DEFAULT 'pending'
               CHECK (state IN ('pending', 'verified', 'locked', 'superseded', 'expired')),
  delivered    boolean     NOT NULL DEFAULT false,
  ip           inet,
  device_id    text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  expires_at   timestamptz NOT NULL,
  consumed_at  timestamptz
);
CREATE INDEX otp_challenges_phone ON identity.otp_challenges (phone_bidx, created_at DESC);
CREATE INDEX otp_challenges_expiry ON identity.otp_challenges (expires_at);

CREATE TABLE identity.devices (
  id               uuid        PRIMARY KEY,
  user_id          uuid        NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
  client_device_id text        NOT NULL CHECK (char_length(client_device_id) BETWEEN 8 AND 128),
  platform         text        NOT NULL CHECK (platform IN ('android', 'ios', 'web')),
  model            text        CHECK (char_length(model) <= 100),
  app_version      text        CHECK (char_length(app_version) <= 40),
  public_key       bytea,      -- P-256 SPKI (DER); refresh requests are signed with the private key (proof of possession)
  created_at       timestamptz NOT NULL DEFAULT now(),
  last_seen_at     timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, client_device_id)
);

CREATE TABLE identity.sessions (
  id            uuid        PRIMARY KEY,
  user_id       uuid        NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
  device_id     uuid        NOT NULL REFERENCES identity.devices (id) ON DELETE CASCADE,
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_used_at  timestamptz NOT NULL DEFAULT now(),
  expires_at    timestamptz NOT NULL,
  revoked_at    timestamptz,
  revoke_reason text        CHECK (revoke_reason IN ('logout', 'revoked_by_user', 'revoked_others', 'refresh_replay',
                                                     'account_status', 'admin', 'expired', 'deletion')),
  ip            inet,
  user_agent    text        CHECK (char_length(user_agent) <= 300)
);
CREATE INDEX sessions_user_active ON identity.sessions (user_id, created_at DESC) WHERE revoked_at IS NULL;

-- Refresh tokens rotate on every use. A token that was already rotated and is presented again is a replay:
-- the whole session is revoked (doc 08: refresh-token theft detection).
CREATE TABLE identity.refresh_tokens (
  token_hash bytea       PRIMARY KEY,
  session_id uuid        NOT NULL REFERENCES identity.sessions (id) ON DELETE CASCADE,
  state      text        NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'rotated', 'revoked')),
  created_at timestamptz NOT NULL DEFAULT now(),
  rotated_at timestamptz
);
CREATE INDEX refresh_tokens_session ON identity.refresh_tokens (session_id);
CREATE UNIQUE INDEX refresh_tokens_one_active ON identity.refresh_tokens (session_id) WHERE state = 'active';

CREATE TABLE identity.status_history (
  id          uuid        PRIMARY KEY,
  user_id     uuid        NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
  from_status text        NOT NULL,
  to_status   text        NOT NULL,
  reason      text        NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 500),
  actor_type  text        NOT NULL CHECK (actor_type IN ('user', 'admin', 'system')),
  actor_id    uuid,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX status_history_user ON identity.status_history (user_id, created_at DESC);

-- Account recovery foundations (doc 07 §4.1): a user who lost the old number files a request; the full flow
-- (Shahkar-class ownership check, liveness, admin decision) arrives with KYC in P4.
CREATE TABLE identity.recovery_requests (
  id               uuid        PRIMARY KEY,
  old_phone_bidx   bytea       NOT NULL,
  new_phone_enc    bytea       NOT NULL,
  new_phone_bidx   bytea       NOT NULL,
  new_phone_last4  text        NOT NULL,
  note             text        CHECK (char_length(note) <= 500),
  state            text        NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'in_review', 'approved', 'rejected')),
  ip               inet,
  created_at       timestamptz NOT NULL DEFAULT now(),
  decided_at       timestamptz
);
CREATE INDEX recovery_requests_state ON identity.recovery_requests (state, created_at DESC);

-- Consent log (doc 19 UC-5): every purpose is a separate, explicit, unticked-by-default choice; the text version
-- shown is recorded with the surface and request attributes. Withdrawal sets withdrawn_at (Settings → Privacy).
CREATE TABLE identity.consents (
  id           uuid        PRIMARY KEY,
  user_id      uuid        NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
  purpose      text        NOT NULL CHECK (purpose IN ('terms', 'privacy', 'age_confirmation')),
  text_version text        NOT NULL CHECK (char_length(text_version) BETWEEN 1 AND 40),
  surface      text        NOT NULL CHECK (surface IN ('android', 'ios', 'web')),
  granted_at   timestamptz NOT NULL DEFAULT now(),
  withdrawn_at timestamptz,
  ip           inet,
  device_id    text
);
CREATE UNIQUE INDEX consents_current ON identity.consents (user_id, purpose, text_version) WHERE withdrawn_at IS NULL;

GRANT USAGE ON SCHEMA identity TO app_public, app_admin, app_worker;
GRANT SELECT, INSERT, UPDATE ON identity.users, identity.phone_identities, identity.otp_challenges, identity.devices,
  identity.sessions, identity.refresh_tokens, identity.consents TO app_public;
GRANT INSERT ON identity.status_history, identity.recovery_requests TO app_public;
GRANT SELECT ON ALL TABLES IN SCHEMA identity TO app_admin;
GRANT UPDATE ON identity.users, identity.sessions, identity.refresh_tokens, identity.recovery_requests TO app_admin;
GRANT INSERT ON identity.status_history TO app_admin;
GRANT SELECT, DELETE ON identity.otp_challenges TO app_worker;
GRANT SELECT, UPDATE ON identity.users, identity.sessions TO app_worker;
