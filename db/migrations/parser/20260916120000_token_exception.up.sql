BEGIN;

ALTER TABLE "token_parse_exception" RENAME TO "token_exception";
ALTER TABLE "token_exception" RENAME CONSTRAINT "token_parse_exception_unique" TO "token_exception_unique";
ALTER SEQUENCE "token_parse_exception_id_seq" RENAME TO "token_exception_id_seq";

ALTER TABLE "token_exception" ADD COLUMN IF NOT EXISTS "skip_parse" BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE "token_exception" ADD COLUMN IF NOT EXISTS "hidden"     BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
