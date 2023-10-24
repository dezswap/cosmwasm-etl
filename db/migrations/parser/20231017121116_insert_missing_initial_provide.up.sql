DO $$
DECLARE selected_synced_height synced_height%rowtype;

BEGIN
ALTER TYPE "tx_type" RENAME VALUE 'mint_token' TO 'initial_provide';
SELECT
  *
FROM
  synced_height INTO selected_synced_height
WHERE
  chain_id = 'dimension_37-1';

IF found AND selected_synced_height.height > 4524215 THEN
    INSERT INTO
    "public"."parsed_tx" (
        "chain_id",
        "height",
        "timestamp",
        "hash",
        "type",
        "sender",
        "contract",
        "lp",
        "lp_amount",
        "created_at",
        "asset0",
        "asset1",
        "asset0_amount",
        "asset1_amount"
    )
    VALUES
    ('dimension_37-1',   4079526,	1685829522, '9CBA5DB9007B9CD8D88987B209BD4B1183792C9805D9AA05442009EA2C15A76E', 'initial_provide', 'xpla1mjsme7n0v6qqkzvx8smm42va6zuh7wku2m0cpr', 'xpla1tj2uked2zfq958w6cugnm44qlllnrpaaxk09j39qqdqhhedjpljs80ffda', 'xpla1kytv5yjkkszr2zs8937lp6kvqhxxx2e429qwqk2dr7rh99pklf9szz7g86', 1000, 1685829532.171276, 'xpla1fajhd23hc7nq3m4p0vxrzk9yu5unp7xc7np0qg558qky4jhamjfqnem6dn', 'axpla', 0, 0),
    ('dimension_37-1',   4328499,	1687347087, 'B467C8115FBE34F5A799740F9A2490E8580AD438A6033743D2453CBFDB0E9114', 'initial_provide', 'xpla1km8w8npy53r5mrumjn9gp6ry75yf86mcz3hvrp', 'xpla1508ss54fyf4k2y2438h98s8z6ft670dr27ah0uyr2hxzr8dr5j6q0kycum', 'xpla17ap4g7kjnat0e4feh8exvmpw9ddceutyfatp8rk9f944ve72890sj2lfug', 1000, 1687347095.272097, 'axpla', 'xpla1hz3svgdhmv67lsqlduu0tcnd3f75c0xr0mu48l6ywuwlz43zssjqc0z2h4', 0, 0),
    ('dimension_37-1',   4524215,	1688539847, '4CD62E3B141DF94B3470730C49FE441C9C4F2462E4DCA085D8F6987A9AB3839B', 'initial_provide', 'xpla1g8hkzkgfa3uq0cg9d6h99jk5nlg92lwx2jme2l', 'xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp', 'xpla1p3dsd5k7cl0p0jhtj0s00vrf45s4c6j3wtjsaaalatmr2jwl7p0sy3sd26', 1000, 1688539856.778621, 'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5', 'xpla1he85n9h0mcnzhpegj76wwcyjv626tced0zkp58wakjc7d3fm50xq8sywg6', 0, 0);
ELSE
    RAISE NOTICE 'target DB does not contain dimension';
END IF;
END $$;
