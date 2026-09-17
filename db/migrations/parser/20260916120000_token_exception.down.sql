BEGIN;

-- The old schema cannot express "hidden but still parsed", and keeping those rows would
-- make the reverted parser drop the token's transfers. Guarded because the column may
-- already be gone; plpgsql parses the branch only when it runs.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_attribute
    WHERE attrelid = to_regclass('token_exception')
      AND attname = 'skip_parse'
      AND NOT attisdropped
  ) THEN
    DELETE FROM "token_exception" WHERE NOT "skip_parse";
  END IF;
END $$;

ALTER TABLE IF EXISTS "token_exception" DROP COLUMN IF EXISTS "hidden";
ALTER TABLE IF EXISTS "token_exception" DROP COLUMN IF EXISTS "skip_parse";

-- Every rename below checks the source and the target: IF EXISTS alone still fails on a
-- hand repaired schema holding both names, and RENAME CONSTRAINT has no IF EXISTS form
-- at all. to_regclass yields NULL for a missing relation rather than erroring.
DO $$
BEGIN
  IF to_regclass('token_exception_id_seq') IS NOT NULL
     AND to_regclass('token_parse_exception_id_seq') IS NULL THEN
    ALTER SEQUENCE "token_exception_id_seq" RENAME TO "token_parse_exception_id_seq";
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = to_regclass('token_exception')
      AND conname = 'token_exception_unique'
  ) AND NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = to_regclass('token_exception')
      AND conname = 'token_parse_exception_unique'
  ) THEN
    ALTER TABLE "token_exception"
      RENAME CONSTRAINT "token_exception_unique" TO "token_parse_exception_unique";
  END IF;

  IF to_regclass('token_exception') IS NOT NULL
     AND to_regclass('token_parse_exception') IS NULL THEN
    ALTER TABLE "token_exception" RENAME TO "token_parse_exception";
  END IF;
END $$;

COMMIT;
