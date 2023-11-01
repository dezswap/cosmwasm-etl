BEGIN;

alter table if exists pair_stats_in_24h rename to pair_stats_recent;

COMMIT;
