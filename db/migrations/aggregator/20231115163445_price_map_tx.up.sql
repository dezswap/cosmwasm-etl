BEGIN;

alter table if exists price add column tx_id bigint;

update price
set tx_id = t.tx_id
    from (
    select pr.id price_id, pt.id tx_id
    from (
        select id, chain_id, height, token_id,
               row_number() OVER (PARTITION BY chain_id, height ORDER BY id) rn
        from price
        ) pr
    join (
        select id, chain_id, height, contract,
               row_number() OVER (PARTITION BY chain_id, height ORDER BY id) rn
        from parsed_tx
        ) pt
        on pr.chain_id = pt.chain_id and pr.height = pt.height and pr.rn >= pt.rn
    join pair p on pt.chain_id = p.chain_id and pt.contract = p.contract
    join tokens t0 on p.chain_id = t0.chain_id and p.asset0 = t0.address
    join tokens t1 on p.chain_id = t1.chain_id and p.asset1 = t1.address
    where (pr.token_id = t0.id or pr.token_id = t1.id)
    ) t
where id = t.price_id;

COMMIT;