BEGIN;

ALTER TABLE "public"."collector_blocks"
    ALTER COLUMN "created_at" DROP DEFAULT,
    ALTER COLUMN "updated_at" DROP DEFAULT;

ALTER TABLE "public"."collector_blocks"
    ALTER COLUMN "created_at" TYPE int8
        USING FLOOR(EXTRACT(EPOCH FROM "created_at" AT TIME ZONE 'UTC'))::int8,
    ALTER COLUMN "created_at" SET DEFAULT FLOOR(EXTRACT(EPOCH FROM now()))::int8,
    ALTER COLUMN "updated_at" TYPE int8
        USING FLOOR(EXTRACT(EPOCH FROM "updated_at" AT TIME ZONE 'UTC'))::int8,
    ALTER COLUMN "updated_at" SET DEFAULT FLOOR(EXTRACT(EPOCH FROM now()))::int8;

ALTER TABLE "public"."collector_pool_snapshots"
    ALTER COLUMN "created_at" DROP DEFAULT,
    ALTER COLUMN "updated_at" DROP DEFAULT;

ALTER TABLE "public"."collector_pool_snapshots"
    ALTER COLUMN "created_at" TYPE int8
        USING FLOOR(EXTRACT(EPOCH FROM "created_at" AT TIME ZONE 'UTC'))::int8,
    ALTER COLUMN "created_at" SET DEFAULT FLOOR(EXTRACT(EPOCH FROM now()))::int8,
    ALTER COLUMN "updated_at" TYPE int8
        USING FLOOR(EXTRACT(EPOCH FROM "updated_at" AT TIME ZONE 'UTC'))::int8,
    ALTER COLUMN "updated_at" SET DEFAULT FLOOR(EXTRACT(EPOCH FROM now()))::int8;

ALTER TABLE "public"."collector_synced_heights"
    ALTER COLUMN "created_at" DROP DEFAULT,
    ALTER COLUMN "updated_at" DROP DEFAULT;

ALTER TABLE "public"."collector_synced_heights"
    ALTER COLUMN "created_at" TYPE int8
        USING FLOOR(EXTRACT(EPOCH FROM "created_at" AT TIME ZONE 'UTC'))::int8,
    ALTER COLUMN "created_at" SET DEFAULT FLOOR(EXTRACT(EPOCH FROM now()))::int8,
    ALTER COLUMN "updated_at" TYPE int8
        USING FLOOR(EXTRACT(EPOCH FROM "updated_at" AT TIME ZONE 'UTC'))::int8,
    ALTER COLUMN "updated_at" SET DEFAULT FLOOR(EXTRACT(EPOCH FROM now()))::int8;

COMMIT;
