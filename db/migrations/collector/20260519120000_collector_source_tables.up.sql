BEGIN;

CREATE TABLE IF NOT EXISTS "public"."collector_blocks" (
    "chain_id" varchar NOT NULL,
    "height" int8 NOT NULL,
    "block_time" timestamp NOT NULL,
    "txs" jsonb NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT now(),
    "updated_at" timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY ("chain_id", "height")
);

CREATE TABLE IF NOT EXISTS "public"."collector_pool_snapshots" (
    "chain_id" varchar NOT NULL,
    "height" int8 NOT NULL,
    "pool_infos" jsonb NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT now(),
    "updated_at" timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY ("chain_id", "height")
);

CREATE TABLE IF NOT EXISTS "public"."collector_synced_heights" (
    "chain_id" varchar NOT NULL PRIMARY KEY,
    "height" int8 NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT now(),
    "updated_at" timestamp NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS collector_blocks_height_idx ON collector_blocks(height);
CREATE INDEX IF NOT EXISTS collector_pool_snapshots_height_idx ON collector_pool_snapshots(height);

COMMIT;
