BEGIN;

create table if not exists lp_history
(
    id          bigserial primary key,
    height      bigint                                                   not null,
    pair_id     bigint                                                   not null,
    chain_id    varchar                                                  not null,
    liquidity0  numeric                                                  not null,
    liquidity1  numeric                                                  not null,
    timestamp   double precision                                         not null,
    created_at  double precision default date_part('epoch'::text, now()) not null,
    modified_at double precision default date_part('epoch'::text, now()) not null
);

create index if not exists lp_history_chain_id_height_pair_id_uidx on lp_history (chain_id, height, pair_id);

create table if not exists route
(
    id          bigserial primary key,
    chain_id    text                                                     not null,
    asset0      varchar                                                  not null,
    asset1      varchar                                                  not null,
    hop_count   integer                                                  not null,
    route       character varying[]                                      not null,
    created_at  double precision default date_part('epoch'::text, now()) not null,
    modified_at double precision default date_part('epoch'::text, now()) not null
);

create unique index if not exists route_chain_id_asset0_asset1_uidx on route (chain_id, asset0, asset1, route);

create table if not exists price
(
    id             bigserial primary key,
    height         bigint                                                   not null,
    chain_id       text                                                     not null,
    token_id       bigint                                                   not null,
    price          numeric                                                     not null,
    price_token_id bigint                                                   not null,
    route_id       bigint                                                   not null,
    created_at     double precision default date_part('epoch'::text, now()) not null,
    modified_at    double precision default date_part('epoch'::text, now()) not null
);

create index if not exists price_token_id_height_uidx on price (token_id, height);

create table if not exists pair_stats_30m
(
    id                   bigserial primary key,
    year_utc             smallint                                                 not null,
    month_utc            smallint                                                 not null,
    day_utc              smallint                                                 not null,
    hour_utc             smallint                                                 not null,
    minute_utc           smallint                                                 not null,
    pair_id              bigint                                                   not null,
    chain_id             varchar                                                  not null,
    volume0              numeric                                                  not null,
    volume1              numeric                                                  not null,
    volume0_in_price     numeric                                                  not null,
    volume1_in_price     numeric                                                  not null,
    last_swap_price      numeric                                                  not null,
    liquidity0           numeric                                                  not null,
    liquidity1           numeric                                                  not null,
    liquidity0_in_price  numeric                                                  not null,
    liquidity1_in_price  numeric                                                  not null,
    commission0          numeric                                                  not null,
    commission1          numeric                                                  not null,
    commission0_in_price numeric                                                  not null,
    commission1_in_price numeric                                                  not null,
    price_token          varchar                                                  not null,
    tx_cnt               bigint                                                   not null,
    provider_cnt         bigint                                                   not null,
    timestamp            double precision                                         not null,
    created_at           double precision default date_part('epoch'::text, now()) not null,
    modified_at          double precision default date_part('epoch'::text, now()) not null
);

create index if not exists pair_stats_30m_chain_id_timestamp_uidx on pair_stats_30m (chain_id, timestamp);
create index if not exists pair_stats_30m_pair_id_timestamp_uidx on pair_stats_30m (pair_id, timestamp);

create table if not exists pair_stats_in_24h
(
    id                   bigserial primary key,
    pair_id              bigserial,
    chain_id             varchar                                                  not null,
    volume0              numeric                                                  not null,
    volume1              numeric                                                  not null,
    volume0_in_price     numeric                                                  not null,
    volume1_in_price     numeric                                                  not null,
    liquidity0           numeric                                                  not null,
    liquidity1           numeric                                                  not null,
    liquidity0_in_price  numeric                                                  not null,
    liquidity1_in_price  numeric                                                  not null,
    commission0          numeric                                                  not null,
    commission1          numeric                                                  not null,
    commission0_in_price numeric                                                  not null,
    commission1_in_price numeric                                                  not null,
    price_token          varchar                                                  not null,
    height               bigint                                                   not null,
    timestamp            double precision                                         not null,
    created_at           double precision default date_part('epoch'::text, now()) not null,
    modified_at          double precision default date_part('epoch'::text, now()) not null
);

create index if not exists pair_stats_in_24h_chain_id_height_uidx on pair_stats_in_24h (chain_id, height);
create index if not exists pair_stats_in_24h_timestamp_uidx on pair_stats_in_24h (timestamp);

COMMIT;