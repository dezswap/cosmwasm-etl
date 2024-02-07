BEGIN;

update pair_stats_30m ps set last_swap_price = pow(10, t0.decimals - t1.decimals)/last_swap_price
    from pair p
    join tokens t0 on p.chain_id = t0.chain_id and p.asset0 = t0.address
    join tokens t1 on p.chain_id = t1.chain_id and p.asset1 = t1.address
where ps.pair_id = p.id;

COMMIT;