BEGIN;

create table if not exists account
(
    id         bigserial
    primary key,
    address    varchar not null
    unique,
    created_at double precision default EXTRACT(epoch FROM now())
    );

create table if not exists h_account_stats_30m
(
    id              bigserial
    primary key,
    year_utc        integer          not null,
    month_utc       integer          not null,
    day_utc         integer          not null,
    hour_utc        integer          not null,
    minute_utc      integer          not null,
    ts              double precision not null,
    chain_id        varchar          not null,
    account_id      bigint           not null,
    pair_id         bigint           not null,
    tx_cnt          integer          not null,
    asset0_amount   numeric(40)      not null,
    asset1_amount   numeric(40)      not null,
    total_lp_amount numeric(40)      not null,
    created_at      double precision default EXTRACT(epoch FROM now())
    );

create index if not exists h_account_stats_30m_ts_idx
    on h_account_stats_30m (ts);

create index if not exists h_account_stats_30m_account_id_pair_id_time_unit_idx
    on h_account_stats_30m (account_id, pair_id, year_utc, month_utc, day_utc, hour_utc, minute_utc);

create table if not exists h_pair_stats_30m
(
    id                       bigserial
    primary key,
    year_utc                 integer          not null,
    month_utc                integer          not null,
    day_utc                  integer          not null,
    hour_utc                 integer          not null,
    minute_utc               integer          not null,
    ts                       double precision not null,
    chain_id                 varchar          not null,
    pair_id                  bigint           not null,
    tx_cnt                   integer          not null,
    provider_cnt             integer          not null,
    asset0_amount            numeric(40)      not null,
    asset1_amount            numeric(40)      not null,
    asset0_commission_amount numeric(40)      not null,
    asset1_commission_amount numeric(40)      not null,
    lp_amount                numeric(40)      not null,
    created_at               double precision default EXTRACT(epoch FROM now())
    );

create index if not exists h_pair_stats_30m_ts_idx
    on h_pair_stats_30m (ts);

create index if not exists h_pair_stats_30m_pair_id_time_unit_idx
    on h_pair_stats_30m (pair_id, year_utc, month_utc, day_utc, hour_utc, minute_utc);

COMMIT;