BEGIN;

-- The old schema cannot express "hidden but still parsed", and keeping those rows would
-- make the reverted parser drop the token's transfers.
DELETE FROM "token_exception" WHERE NOT "skip_parse";

ALTER TABLE "token_exception" DROP COLUMN "hidden";
ALTER TABLE "token_exception" DROP COLUMN "skip_parse";

ALTER SEQUENCE "token_exception_id_seq" RENAME TO "token_parse_exception_id_seq";
ALTER TABLE "token_exception" RENAME CONSTRAINT "token_exception_unique" TO "token_parse_exception_unique";
ALTER TABLE "token_exception" RENAME TO "token_parse_exception";

COMMIT;
