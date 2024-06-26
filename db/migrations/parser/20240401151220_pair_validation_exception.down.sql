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

DROP TABLE IF EXISTS "pair_validation_exception";

COMMIT;
