BEGIN;

CREATE TABLE "parse_quarantine" (
  "id"          BIGSERIAL NOT NULL PRIMARY KEY,
  "chain_id"    VARCHAR NOT NULL, CHECK("chain_id" <> ''),
  "height"      BIGINT NOT NULL, CHECK("height" >= 0),
  "hash"        VARCHAR NOT NULL, CHECK("hash" <> ''),
  "stage"       VARCHAR NOT NULL, CHECK("stage" <> ''),
  "contract"    VARCHAR NOT NULL DEFAULT '',
  "action"      VARCHAR NOT NULL DEFAULT '',
  "error"       TEXT NOT NULL, CHECK("error" <> ''),
  "raw_tx"      JSONB NOT NULL,
  "status"      VARCHAR NOT NULL DEFAULT 'pending',
  "resolved_at" DOUBLE PRECISION,
  "created_at"  DOUBLE PRECISION NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  "updated_at"  DOUBLE PRECISION NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT parse_quarantine_chain_hash_unique UNIQUE ("chain_id", "hash"),
  CONSTRAINT parse_quarantine_status_check CHECK ("status" IN ('pending', 'resolved'))
);

CREATE INDEX parse_quarantine_pending_height_idx
  ON parse_quarantine ("chain_id", "status", "height");

COMMIT;
