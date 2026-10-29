-- 0001 platform foundations (doc 06 §2, §3, §8; ADR-006, ADR-007).
-- Schema `platform`: RLS context helpers, outbox, inbox, job queue, idempotency keys, audit log,
-- feature flags (regulated flags locked off), remote configuration.

CREATE SCHEMA platform;
COMMENT ON SCHEMA platform IS 'Cross-cutting infrastructure: outbox, inbox, jobs, idempotency, audit, flags, config';

-- ---------------------------------------------------------------------------------------------
-- RLS context. Values are written per transaction with set_config(..., true) (= SET LOCAL).
-- NULLIF(...,'') matters: on a pooled connection an ended SET LOCAL leaves '' behind; ''::uuid would raise.
-- With no context, policies evaluate to false (fail closed).
-- ---------------------------------------------------------------------------------------------
CREATE FUNCTION platform.current_audience() RETURNS text
  LANGUAGE sql STABLE PARALLEL SAFE
  AS $$ SELECT NULLIF(current_setting('app.audience', true), '') $$;

CREATE FUNCTION platform.current_store_id() RETURNS uuid
  LANGUAGE sql STABLE PARALLEL SAFE
  AS $$ SELECT NULLIF(current_setting('app.store_id', true), '')::uuid $$;

CREATE FUNCTION platform.current_actor_id() RETURNS text
  LANGUAGE sql STABLE PARALLEL SAFE
  AS $$ SELECT NULLIF(current_setting('app.actor_id', true), '') $$;

-- ---------------------------------------------------------------------------------------------
-- Transactional outbox (single table, ADR-006). Written in the same transaction as the state change;
-- the worker's relay publishes to NATS and marks dispatched_at. Delivery is at-least-once.
-- ---------------------------------------------------------------------------------------------
CREATE TABLE platform.outbox (
  event_id          uuid        PRIMARY KEY,
  module            text        NOT NULL CHECK (module ~ '^[a-z][a-z_]*$'),
  aggregate_type    text        NOT NULL,
  aggregate_id      text        NOT NULL,
  aggregate_version bigint      NOT NULL DEFAULT 0,
  type              text        NOT NULL CHECK (type ~ '^vira\.[a-z_]+\.[a-z_]+\.v[0-9]+$'),
  payload           jsonb       NOT NULL,
  traceparent       text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  available_at      timestamptz NOT NULL DEFAULT now(),
  attempts          integer     NOT NULL DEFAULT 0,
  last_error        text,
  dispatched_at     timestamptz
);
CREATE INDEX outbox_pending_idx ON platform.outbox (available_at, created_at) WHERE dispatched_at IS NULL;
CREATE INDEX outbox_dispatched_idx ON platform.outbox (dispatched_at) WHERE dispatched_at IS NOT NULL;

-- Consumer de-duplication (at-least-once delivery → exactly-once effects).
CREATE TABLE platform.inbox (
  consumer     text        NOT NULL,
  event_id     uuid        NOT NULL,
  processed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (consumer, event_id)
);
CREATE INDEX inbox_processed_idx ON platform.inbox (processed_at);

-- ---------------------------------------------------------------------------------------------
-- PostgreSQL-backed job queue (ADR-007). Jobs are enqueued in the caller's transaction.
-- ---------------------------------------------------------------------------------------------
CREATE TABLE platform.jobs (
  id            uuid        PRIMARY KEY,
  kind          text        NOT NULL CHECK (kind ~ '^[a-z][a-z0-9_.]*$'),
  payload       jsonb       NOT NULL DEFAULT '{}',
  state         text        NOT NULL DEFAULT 'available'
                            CHECK (state IN ('available', 'running', 'succeeded', 'dead', 'cancelled')),
  attempt       integer     NOT NULL DEFAULT 0,
  max_attempts  integer     NOT NULL DEFAULT 25 CHECK (max_attempts BETWEEN 1 AND 1000),
  run_at        timestamptz NOT NULL DEFAULT now(),
  locked_until  timestamptz,
  locked_by     text,
  unique_key    text,
  last_error    text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  finished_at   timestamptz
);
CREATE INDEX jobs_ready_idx ON platform.jobs (run_at) WHERE state = 'available';
CREATE INDEX jobs_running_idx ON platform.jobs (locked_until) WHERE state = 'running';
CREATE UNIQUE INDEX jobs_unique_active_idx ON platform.jobs (kind, unique_key)
  WHERE unique_key IS NOT NULL AND state IN ('available', 'running');

-- ---------------------------------------------------------------------------------------------
-- Idempotency keys (doc 06 §8, audit F-23). Transient failures are never stored as the final answer.
-- ---------------------------------------------------------------------------------------------
CREATE TABLE platform.idempotency_keys (
  principal_id     text        NOT NULL,
  key              text        NOT NULL CHECK (length(key) BETWEEN 8 AND 128),
  request_hash     text        NOT NULL,
  method           text        NOT NULL,
  path             text        NOT NULL,
  state            text        NOT NULL CHECK (state IN ('in_progress', 'completed', 'failed_retryable')),
  lease_until      timestamptz NOT NULL,
  response_status  integer,
  response_headers jsonb,
  response_body    bytea,
  created_at       timestamptz NOT NULL DEFAULT now(),
  expires_at       timestamptz NOT NULL,
  PRIMARY KEY (principal_id, key)
);
CREATE INDEX idempotency_keys_expires_idx ON platform.idempotency_keys (expires_at);

-- ---------------------------------------------------------------------------------------------
-- Audit log: append-only, partitioned monthly, PK (created_at, id) (doc 06 §8).
-- Hourly sealing (signed Merkle root to WORM storage, audit F-27) is added before launch (gate L).
-- ---------------------------------------------------------------------------------------------
CREATE TABLE platform.audit_events (
  id          uuid        NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  actor_type  text        NOT NULL CHECK (actor_type IN ('admin', 'user', 'seller_staff', 'system', 'anonymous')),
  actor_id    text,
  action      text        NOT NULL CHECK (action ~ '^[a-z][a-z0-9_.]*$'),
  target_type text,
  target_id   text,
  outcome     text        NOT NULL CHECK (outcome IN ('success', 'failure', 'denied')),
  ip          inet,
  user_agent  text,
  trace_id    text,
  metadata    jsonb       NOT NULL DEFAULT '{}',
  PRIMARY KEY (created_at, id)
) PARTITION BY RANGE (created_at);

CREATE TABLE platform.audit_events_default PARTITION OF platform.audit_events DEFAULT;
CREATE INDEX audit_events_actor_idx ON platform.audit_events (actor_type, actor_id, created_at DESC);
CREATE INDEX audit_events_action_idx ON platform.audit_events (action, created_at DESC);

CREATE FUNCTION platform.reject_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'platform.audit_events is append-only' USING ERRCODE = 'insufficient_privilege';
END $$;
CREATE TRIGGER audit_events_append_only
  BEFORE UPDATE OR DELETE ON platform.audit_events
  FOR EACH ROW EXECUTE FUNCTION platform.reject_audit_mutation();
CREATE TRIGGER audit_events_no_truncate
  BEFORE TRUNCATE ON platform.audit_events
  FOR EACH STATEMENT EXECUTE FUNCTION platform.reject_audit_mutation();

-- Creates monthly partitions from the current month through `months_ahead`. Runs as its owner (migrator)
-- so the worker can call it without DDL privileges. Never creates a partition that would overlap rows in
-- the default partition (those months stay in default).
CREATE FUNCTION platform.ensure_audit_partitions(months_ahead integer DEFAULT 2) RETURNS integer
  LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, platform AS $$
DECLARE
  m_start date := date_trunc('month', now())::date;
  m_end   date;
  name    text;
  created integer := 0;
BEGIN
  FOR i IN 0..GREATEST(months_ahead, 0) LOOP
    m_end := (m_start + interval '1 month')::date;
    name := format('audit_events_%s', to_char(m_start, 'YYYY_MM'));
    IF NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
                   WHERE n.nspname = 'platform' AND c.relname = name)
       AND NOT EXISTS (SELECT 1 FROM platform.audit_events_default
                       WHERE created_at >= m_start AND created_at < m_end) THEN
      EXECUTE format('CREATE TABLE platform.%I PARTITION OF platform.audit_events FOR VALUES FROM (%L) TO (%L)',
                     name, m_start, m_end);
      created := created + 1;
    END IF;
    m_start := m_end;
  END LOOP;
  RETURN created;
END $$;
SELECT platform.ensure_audit_partitions(2);

-- ---------------------------------------------------------------------------------------------
-- Feature flags and remote configuration (doc 06 §8). Every change is versioned and audited.
-- Regulated flags (real money, settlement, withdrawals, wallet, USDT, credit) are LOCKED OFF by a
-- CHECK constraint: enabling one requires a reviewed migration recording the approved gate
-- (FG-1b, G1, G2, G3 — doc 20 §2), not an admin click.
-- ---------------------------------------------------------------------------------------------
CREATE TABLE platform.feature_flags (
  key         text        PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$'),
  description text        NOT NULL,
  enabled     boolean     NOT NULL DEFAULT false,
  regulated   boolean     NOT NULL DEFAULT false,
  gate        text,
  audiences   text[]      NOT NULL DEFAULT '{public}',
  version     integer     NOT NULL DEFAULT 1,
  updated_at  timestamptz NOT NULL DEFAULT now(),
  updated_by  text,
  CONSTRAINT regulated_flags_locked_off CHECK (NOT (regulated AND enabled)),
  CONSTRAINT regulated_flags_name_gate CHECK (NOT regulated OR gate IS NOT NULL)
);

CREATE TABLE platform.feature_flag_changes (
  id          uuid        PRIMARY KEY,
  flag_key    text        NOT NULL REFERENCES platform.feature_flags (key),
  old_enabled boolean     NOT NULL,
  new_enabled boolean     NOT NULL,
  version     integer     NOT NULL,
  reason      text        NOT NULL CHECK (length(reason) >= 3),
  changed_by  text        NOT NULL,
  changed_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX feature_flag_changes_flag_idx ON platform.feature_flag_changes (flag_key, changed_at DESC);

INSERT INTO platform.feature_flags (key, description, enabled, regulated, gate, audiences) VALUES
  ('payments.real_money',      'Real-money card payments through a PSP',            false, true,  'FG-1b', '{public}'),
  ('settlement.live',          'Live seller settlement',                            false, true,  'FG-1b', '{public,seller}'),
  ('payouts.withdrawals',      'Seller withdrawals / payouts',                      false, true,  'FG-1b', '{seller}'),
  ('wallet.topup',             'Wallet stored-value top-up',                        false, true,  'G1',    '{public}'),
  ('usdt.deposits',            'USDT deposits / payments',                          false, true,  'G2',    '{public}'),
  ('usdt.withdrawals',         'Eligible seller USDT withdrawals',                  false, true,  'G2',    '{seller}'),
  ('usdt.conversion',          'USDT <-> Toman conversion (both directions)',      false, true,  'G2',    '{public,seller}'),
  ('credit.bnpl',              'Credit / buy-now-pay-later instalments',            false, true,  'G3',    '{public}'),
  ('app.maintenance_mode',     'Public API answers 503 maintenance (except health)', false, false, NULL,   '{public}'),
  ('seller.web_portal',        'Seller Admin web portal sign-in (Phase 4)',         false, false, NULL,    '{seller}');

CREATE TABLE platform.remote_config (
  key        text        PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$'),
  value      jsonb       NOT NULL,
  public     boolean     NOT NULL DEFAULT false,
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by text
);
INSERT INTO platform.remote_config (key, value, public) VALUES
  ('mobile.min_supported_version', '"1.0.0"', true),
  ('mobile.latest_version',        '"1.0.0"', true);

-- ---------------------------------------------------------------------------------------------
-- Grants. Runtime roles never own tables (doc 06 §1).
-- ---------------------------------------------------------------------------------------------
GRANT USAGE ON SCHEMA platform TO app_public, app_admin, app_worker;
GRANT EXECUTE ON FUNCTION platform.current_audience(), platform.current_store_id(), platform.current_actor_id()
  TO app_public, app_admin, app_worker;
REVOKE ALL ON FUNCTION platform.ensure_audit_partitions(integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION platform.ensure_audit_partitions(integer) TO app_worker;

GRANT INSERT ON platform.outbox TO app_public, app_admin, app_worker;
GRANT SELECT, UPDATE, DELETE ON platform.outbox TO app_worker;
GRANT SELECT, INSERT, DELETE ON platform.inbox TO app_worker;
GRANT INSERT ON platform.jobs TO app_public, app_admin, app_worker;
GRANT SELECT, UPDATE, DELETE ON platform.jobs TO app_worker;
GRANT SELECT ON platform.jobs TO app_admin;
GRANT SELECT, INSERT, UPDATE ON platform.idempotency_keys TO app_public, app_admin;
GRANT SELECT, DELETE ON platform.idempotency_keys TO app_worker;
GRANT INSERT ON platform.audit_events TO app_public, app_admin, app_worker;
GRANT SELECT ON platform.audit_events TO app_admin;
GRANT SELECT ON platform.feature_flags, platform.remote_config TO app_public, app_worker;
GRANT SELECT, UPDATE ON platform.feature_flags, platform.remote_config TO app_admin;
GRANT SELECT, INSERT ON platform.feature_flag_changes TO app_admin;
