BEGIN;

alter table if exists price drop column tx_id;

COMMIT;