-- WRITE YOUR MIGRATION CODES FOR UP or DOWN

-- Convention
-- Write snake_case for your tables and columns

-- Index name recommendation {tablename}_{columnname(s)}_{suffix} columnames should be alphabetical order
-- {suffix}
-- pkey for a Primary Key constraint
-- key for a Unique constraint
-- excl for an Exclusion constraint
-- idx for any other kind of index
-- fkey for a Foreign key
-- check for a Check constraint
BEGIN;
CREATE TABLE "pair_validation_exception"(
  "id" BIGSERIAL NOT NULL PRIMARY KEY,
  "chain_id" VARCHAR NOT NULL, CHECK("chain_id" <> ''),
  "contract" VARCHAR NOT NULL, CHECK("contract" <> ''),
  "meta" JSON
);

ALTER TABLE pair_validation_exception ADD CONSTRAINT pair_validation_exception_contract_pair_fkey FOREIGN KEY ("chain_id","contract") REFERENCES pair ("chain_id", "contract");

COMMIT;
