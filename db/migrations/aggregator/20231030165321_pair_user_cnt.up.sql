BEGIN;

create table if not exists account_stats_30m
(
    id                   bigserial primary key,
    year_utc             smallint                                                 not null,
    month_utc            smallint                                                 not null,
    day_utc              smallint                                                 not null,
    hour_utc             smallint                                                 not null,
    minute_utc           smallint                                                 not null,
    address              varchar                                                  not null,
    pair_id              bigint                                                   not null,
    chain_id             varchar                                                  not null,
    tx_cnt               bigint                                                   not null,
    timestamp            double precision                                         not null,
    created_at           double precision default date_part('epoch'::text, now()) not null,
    modified_at          double precision default date_part('epoch'::text, now()) not null
);

create index if not exists account_stats_30m_chain_id_address_idx on account_stats_30m (chain_id, address);
create index if not exists account_stats_30m_chain_id_pair_id_idx on account_stats_30m (chain_id, pair_id);

COMMIT;