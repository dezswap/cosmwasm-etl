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

-- Table Definition
CREATE TABLE "public"."fcd_tx_log" (
    "id" BIGSERIAL NOT NULL PRIMARY KEY,
    "fcd_offset" int8 NOT NULL,
    "height" int8 NOT NULL,
    "timestamp" timestamp NOT NULL,
    "hash" varchar NOT NULL,
    "address" varchar NOT NULL,
    "event_log" varchar NOT NULL,
    "tx_index" int4
);

CREATE INDEX fcd_tx_log_timestamp_idx ON fcd_tx_log(timestamp);
CREATE INDEX fcd_tx_log_height_idx ON fcd_tx_log(height);
CREATE UNIQUE INDEX fcd_tx_log_height_hash_address_key ON fcd_tx_log(height,hash,address);
CREATE INDEX fcd_tx_log_hash_idx ON fcd_tx_log(hash);
CREATE INDEX fcd_tx_log_fcd_offset_idx ON fcd_tx_log(fcd_offset);
CREATE INDEX fcd_tx_log_address_idx ON fcd_tx_log(address);
COMMIT;
