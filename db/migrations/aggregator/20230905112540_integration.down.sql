BEGIN;

drop table if exists h_pair_stats_30m cascade;

drop table if exists lp_history cascade;

drop table if exists route cascade;

drop table if exists price cascade;

drop table if exists pair_stats_30m cascade;

drop table if exists pair_stats_in_24h cascade;

COMMIT;