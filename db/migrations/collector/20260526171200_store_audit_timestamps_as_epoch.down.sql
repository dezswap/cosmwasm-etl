BEGIN;

ALTER TABLE "public"."collector_blocks"
    ALTER COLUMN "created_at" DROP DEFAULT,
    ALTER COLUMN "updated_at" DROP DEFAULT;

ALTER TABLE "public"."collector_blocks"
    ALTER COLUMN "created_at" TYPE timestamp
        USING to_timestamp("created_at") AT TIME ZONE 'UTC',
    ALTER COLUMN "created_at" SET DEFAULT now(),
    ALTER COLUMN "updated_at" TYPE timestamp
        USING to_timestamp("updated_at") AT TIME ZONE 'UTC',
    ALTER COLUMN "updated_at" SET DEFAULT now();

ALTER TABLE "public"."collector_pool_snapshots"
    ALTER COLUMN "created_at" DROP DEFAULT,
    ALTER COLUMN "updated_at" DROP DEFAULT;

ALTER TABLE "public"."collector_pool_snapshots"
    ALTER COLUMN "created_at" TYPE timestamp
        USING to_timestamp("created_at") AT TIME ZONE 'UTC',
    ALTER COLUMN "created_at" SET DEFAULT now(),
    ALTER COLUMN "updated_at" TYPE timestamp
        USING to_timestamp("updated_at") AT TIME ZONE 'UTC',
    ALTER COLUMN "updated_at" SET DEFAULT now();

ALTER TABLE "public"."collector_synced_heights"
    ALTER COLUMN "created_at" DROP DEFAULT,
    ALTER COLUMN "updated_at" DROP DEFAULT;

ALTER TABLE "public"."collector_synced_heights"
    ALTER COLUMN "created_at" TYPE timestamp
        USING to_timestamp("created_at") AT TIME ZONE 'UTC',
    ALTER COLUMN "created_at" SET DEFAULT now(),
    ALTER COLUMN "updated_at" TYPE timestamp
        USING to_timestamp("updated_at") AT TIME ZONE 'UTC',
    ALTER COLUMN "updated_at" SET DEFAULT now();

COMMIT;
