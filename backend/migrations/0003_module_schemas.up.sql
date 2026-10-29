-- 0003 module schemas (doc 06 §2): one schema per module, created empty so that ownership and
-- boundaries exist from day one. Tables arrive with each module's phase. Schemas are owned by migrator;
-- runtime roles receive USAGE + table grants in the migration that creates each table.
CREATE SCHEMA identity;   COMMENT ON SCHEMA identity   IS 'P2 identity: users, phone (encrypted + blind index), devices';
CREATE SCHEMA kyc;        COMMENT ON SCHEMA kyc        IS 'P4 KYC levels and decisions (highly sensitive)';
CREATE SCHEMA social;     COMMENT ON SCHEMA social     IS 'P2 social graph, posts, engagement';
CREATE SCHEMA feed;       COMMENT ON SCHEMA feed       IS 'P2 derived feeds (rebuildable)';
CREATE SCHEMA moderation; COMMENT ON SCHEMA moderation IS 'P2 reports and moderation decisions';
CREATE SCHEMA media;      COMMENT ON SCHEMA media      IS 'P3 media metadata';
CREATE SCHEMA messaging;  COMMENT ON SCHEMA messaging  IS 'P5 private communications';
CREATE SCHEMA notify;     COMMENT ON SCHEMA notify     IS 'P5 notifications (recipient_id, store_id, category)';
CREATE SCHEMA live;       COMMENT ON SCHEMA live       IS 'P8 live shopping and calls';
CREATE SCHEMA store;      COMMENT ON SCHEMA store      IS 'P4 stores and sellers (RLS by store_id)';
CREATE SCHEMA catalog;    COMMENT ON SCHEMA catalog    IS 'P4 catalog and inventory (RLS by store_id)';
CREATE SCHEMA checkout;   COMMENT ON SCHEMA checkout   IS 'P6 cart and checkout';
CREATE SCHEMA orders;     COMMENT ON SCHEMA orders     IS 'P6/P7 orders, fulfilment, returns';
CREATE SCHEMA payments;   COMMENT ON SCHEMA payments   IS 'P6 payments (sandbox until FG-1b)';
CREATE SCHEMA ledger;     COMMENT ON SCHEMA ledger     IS 'P6 double-entry ledger in integer Rial; append-only';
CREATE SCHEMA settlement; COMMENT ON SCHEMA settlement IS 'P7 settlement and payouts (sandbox until FG-1b)';
CREATE SCHEMA risk;       COMMENT ON SCHEMA risk       IS 'P4+ fraud and risk';
CREATE SCHEMA support;    COMMENT ON SCHEMA support    IS 'P7 support tickets';
