BEGIN;
CREATE TABLE "token_parse_exception" (
  "id"       BIGSERIAL NOT NULL PRIMARY KEY,
  "chain_id" VARCHAR   NOT NULL, CHECK("chain_id" <> ''),
  "contract" VARCHAR   NOT NULL, CHECK("contract" <> ''),
  CONSTRAINT token_parse_exception_unique UNIQUE ("chain_id", "contract")
);
COMMIT;
