BEGIN;

CREATE TABLE "public"."collector_blocks" (
    "chain_id" varchar NOT NULL,
    "height" int8 NOT NULL,
    "block_time" timestamp NOT NULL,
    "txs" jsonb NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT now(),
    "updated_at" timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY ("chain_id", "height")
);

CREATE TABLE "public"."collector_pool_snapshots" (
    "chain_id" varchar NOT NULL,
    "height" int8 NOT NULL,
    "pool_infos" jsonb NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT now(),
    "updated_at" timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY ("chain_id", "height")
);

CREATE TABLE "public"."collector_synced_heights" (
    "chain_id" varchar NOT NULL PRIMARY KEY,
    "height" int8 NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT now(),
    "updated_at" timestamp NOT NULL DEFAULT now()
);

CREATE INDEX collector_blocks_height_idx ON collector_blocks(height);
CREATE INDEX collector_pool_snapshots_height_idx ON collector_pool_snapshots(height);

COMMIT;
