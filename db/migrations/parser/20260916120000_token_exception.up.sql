BEGIN;

-- Every rename below checks the source and the target: IF EXISTS alone still fails on a
-- hand repaired schema holding both names, and RENAME CONSTRAINT has no IF EXISTS form
-- at all. to_regclass yields NULL for a missing relation rather than erroring.
DO $$
BEGIN
  IF to_regclass('token_parse_exception') IS NOT NULL
     AND to_regclass('token_exception') IS NULL THEN
    ALTER TABLE "token_parse_exception" RENAME TO "token_exception";
  END IF;

  IF to_regclass('token_parse_exception_id_seq') IS NOT NULL
     AND to_regclass('token_exception_id_seq') IS NULL THEN
    ALTER SEQUENCE "token_parse_exception_id_seq" RENAME TO "token_exception_id_seq";
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = to_regclass('token_exception')
      AND conname = 'token_parse_exception_unique'
  ) AND NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = to_regclass('token_exception')
      AND conname = 'token_exception_unique'
  ) THEN
    ALTER TABLE "token_exception"
      RENAME CONSTRAINT "token_parse_exception_unique" TO "token_exception_unique";
  END IF;
END $$;

ALTER TABLE "token_exception" ADD COLUMN IF NOT EXISTS "skip_parse" BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE "token_exception" ADD COLUMN IF NOT EXISTS "hidden"     BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
