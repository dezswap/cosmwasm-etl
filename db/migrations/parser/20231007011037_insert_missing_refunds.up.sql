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
DO $$
DECLARE selected_synced_height synced_height%rowtype;

BEGIN

SELECT
  *
FROM
  synced_height INTO selected_synced_height
WHERE
  chain_id = 'dimension_37-1';

IF found AND selected_synced_height.height > 3048255 THEN
    INSERT INTO
    "public"."parsed_tx" (
        "chain_id",
        "height",
        "timestamp",
        "hash",
        "type",
        "sender",
        "contract",
        "asset0",
        "asset0_amount",
        "asset1",
        "asset1_amount",
        "lp",
        "lp_amount",
        "commission_amount",
        "created_at",
        "meta",
        "commission0_amount",
        "commission1_amount"
    )
    VALUES
    ('dimension_37-1', 1949193, 1672824066, 'F2DCBA292C1DB467BAB5CE86404A1FD7AA104FC2A8E07B6EF86E9C0A7D3D3AA3', 'transfer', 'xpla1d85zdhcptw52rw36mv9h3lp5f6m6ce6exhjey3', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-709059,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5'	, -54797390,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1673517506,	NULL,	0,	0),
    ('dimension_37-1', 2044485, 1673407182, '9F5AB8C2418F21C604FE5B2E093B8A1C3529138E8A93CC46BAE8E5FC54F21E2D', 'transfer', 'xpla1d85zdhcptw52rw36mv9h3lp5f6m6ce6exhjey3', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-1140812,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1674118671,	NULL,	0,	0),
    ('dimension_37-1', 2262496, 1674738715, '51A29105B603F9027ABAA60789464454081FFC4BEC5935EAECE1AE59A78AEAAE', 'transfer', 'xpla1064ufs0cjmjx520xm0af9phj63ym3yk2ur9g2h', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-259599,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1674738722,	NULL,	0,	0),
    ('dimension_37-1', 2272733, 1674801200, '98FA090CA6220A458271D0B53B59A886ACE8863C2CC0A0DE72DE3E266F66FCE4', 'transfer', 'xpla18nvle2r9lxgw24fdls2qmdlemd5a6724qn8hr0', 'xpla1mqneytdhcuznwsuj4hd97nfs6s6vl5ar556nrt6jgjpfrqnzfx4q33ryuh', 'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5',	0,	'xpla1r8td9cjggx9f5ek2gzfh34kguxuz5j9rfjemamvjw2he2y84t3xqf3lte4'	, -8,	'xpla17dez7atlgwrl7lxzszxjy7gzuj325n8r07k85v5zf5hsgq6f6qzq2ql0ax',	0,	0,	1674801210,	NULL,	0,	0),
    ('dimension_37-1', 2272737, 1674801224, '422F1417A1B4BAD86EAD2A99D53135F7CEFF937F1020E3986890C987BC00C8C2', 'transfer', 'xpla18nvle2r9lxgw24fdls2qmdlemd5a6724qn8hr0', 'xpla1j2t9s7xgmudw74pcdtrqj9t2pyg80dasts2ey3edl9u3vdqqwp8qlug95s', 'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5',	0,	'xpla19377ce3k2893s98xp58vg0jtgtvj5qv405y36l4wf4z2xhnjwh6sd8pvxr'	, -3,	'xpla18knfkcehsh5txvcql54j6pvru3ey7svuc0gq2wn4l4rm48cup3csa4p5ag',	0,	0,	1674801235,	NULL,	0,	0),
    ('dimension_37-1', 2272742, 1674801255, 'F1A57C7952CF911A821A3925A181A33637C2FCC445EE00E422EF82CEC02B9231', 'transfer', 'xpla18nvle2r9lxgw24fdls2qmdlemd5a6724qn8hr0', 'xpla18570mzun47xwcddt3f7ryrr6ujn0r0vdavl9zgzlfwkp5xw9357sd7e5qw', 'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5',	0,	'xpla1ddstvl38skwpm284gfaqukn3e8c4mlf26mssy5hppeq6ar2nnw0sr8vh6m'	, -8,	'xpla1lg8plsdphya2mutm8vwf6yn5kg9vt6jrda5tfexuktz6mhfkerfss8xcg5',	0,	0,	1674801265,	NULL,	0,	0),
    ('dimension_37-1', 2272750, 1674801303, 'AF699377B1FC87B9A81A14A73597DDFFA59DCD4C1EC2A0FBCD2B237C0476E90C', 'transfer', 'xpla18nvle2r9lxgw24fdls2qmdlemd5a6724qn8hr0', 'xpla1uz7hfc5tthhg3s8rr2axj9c4w405zdnl9ywjwde0m4qhcse8sy3sakpzal', 'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5',	0,	'xpla1hdnu502uecmddk9w48kxvekgp43mjdpr3mza9kj2tfvjpgef5grszl8rur'	, -2,	'xpla1ewzlxcavclq56gnmg6g0rqmwszce3fm7c8pf0cwtxnjhl86ssmyssqckcg',	0,	0,	1674801310,	NULL,	0,	0),
    ('dimension_37-1', 2275511, 1674818183, 'B76FBB4BBE943DC2502D9BAE1C8C5B690A5F7E9762DF4F4E8B8BF607BF3F794B', 'transfer', 'xpla1hnvmgpxw9y0q7k60eucu9hgrd2w6pkasrw5lnl', 'xpla1sdzaas0068n42xk8ndm6959gpu6n09tajmeuq7vak8t9qt5jrp6szltsnk', 'axpla',	-857091,	'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5', 	0,	'xpla14ndtyhlyqtla6tf2qlqvhk2amkq2fvct49suy9nr7nfzw42tvqxqkynqk5',	0,	0,	1674818194,	NULL,	0,	0),
    ('dimension_37-1', 2375043, 1675425231, '31D4A4373D8F84DC6FFA9EA2DCF438351D229496A52BC4EA39AE55616F2A7CDF', 'transfer', 'xpla1d85zdhcptw52rw36mv9h3lp5f6m6ce6exhjey3', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-944964,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1675425240,	NULL,	0,	0),
    ('dimension_37-1', 2512630, 1676267192, '6D0618355F7128C76B72B695DE79C6D4CF36E522AF5487D395654444D0341983', 'transfer', 'xpla1d85zdhcptw52rw36mv9h3lp5f6m6ce6exhjey3', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-1028970,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1676267204,	NULL,	0,	0),
    ('dimension_37-1', 2527922, 1676360394, 'ACC5BCBCCF5FC780F504A9E9AC31779B3B93CC1762C290ABAC6C2F8A13228991', 'transfer', 'xpla1qxth4g4jv037eruuykc6q2nahvprc26zze7rms', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-796090,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5'	, -6,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1676360402,	NULL,	0,	0),
    ('dimension_37-1', 2569674, 1676614819, '990E4DA2DF5ADFC8F386AECE8D8A6DACDCB22DF957EB025A7B8A25AB01A620E7', 'transfer', 'xpla1g960px7gstnmycv46zmdl6yeqn0ga5p4wjplme', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-1391312,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5'	, -19276,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1676614836,	NULL,	0,	0),
    ('dimension_37-1', 2707775, 1677456672, 'D268484D8F5D4B8A2895656721F3230B42680F8B5A62CD22273B68C64D5FABE9', 'transfer', 'xpla1064ufs0cjmjx520xm0af9phj63ym3yk2ur9g2h', 'xpla1sdzaas0068n42xk8ndm6959gpu6n09tajmeuq7vak8t9qt5jrp6szltsnk', 'axpla',	-299482942074299,	'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5', 	0,	'xpla14ndtyhlyqtla6tf2qlqvhk2amkq2fvct49suy9nr7nfzw42tvqxqkynqk5',	0,	0,	1677456683,	NULL,	0,	0),
    ('dimension_37-1', 2711543, 1677479660, 'F825D788ED2480D62E9B7F9E1416EE82CC7B36626F0C6DBE67308CE136DA5D29', 'transfer', 'xpla1s7ajt6ylzf6wlwdstfv8a6y4lrav8g94y8c2nx', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-1230680,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1677479668,	NULL,	0,	0),
    ('dimension_37-1', 2727682, 1677578090, '12FC1AE97F52B441B6B31D70EBC327403CBA0A7A0625AABE5CCB5408B71E5C97', 'transfer', 'xpla1ple80mwedgcsalmuyslx3wrfwgt08gjrp7k30n', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-538898,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5'	, -787682,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1677578098,	NULL,	0,	0),
    ('dimension_37-1', 2782540, 1677912727, 'FCDA0413C2ACA1F9C288D55EC25C0B934822A9CE6E67A1CEB17AFE4729AB1609', 'transfer', 'xpla1ujuc32u95ztxsg6xplsjm69jkmu8j882sc9xp8', 'xpla1sdzaas0068n42xk8ndm6959gpu6n09tajmeuq7vak8t9qt5jrp6szltsnk', 'axpla',	-371378506186,	'xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5', 	0,	'xpla14ndtyhlyqtla6tf2qlqvhk2amkq2fvct49suy9nr7nfzw42tvqxqkynqk5',	0,	0,	1677912735,	NULL,	0,	0),
    ('dimension_37-1', 2883413, 1678529888, 'AC47FDC6C749F0FAA23A23E9A86820A5CD1C3334CD196F2C14591555FBCA7235', 'transfer', 'xpla1dpld9d6erqwrss9n9t7gcnexma6573vj0mevhw', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-1150191,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1678529898,	NULL,	0,	0),
    ('dimension_37-1', 2883923, 1678533005, 'DA1A29EC11B6E3040F622465F6FB05F762E9E055EE5B2DAEDA8A3E02F9D2CECB', 'transfer', 'xpla1dpld9d6erqwrss9n9t7gcnexma6573vj0mevhw', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-1166373,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1678533013,	NULL,	0,	0),
    ('dimension_37-1', 3048255, 1679537440, '0F5BF6CFBB6EC5F6CC4BB6479DBDE0FF428C89AF7E58CEC12BFB08A30230970A', 'transfer', 'xpla1u4pyzq6mv5z8nvqns7j38qltlp638z7vccwzlr', 'xpla1sms4u3vra5wem5dufl7wwttzyrcgfe529u9rp2rqdst60skllzgsx0ka9f', 'axpla',	-349221,	'ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5', 	0,	'xpla1kqfpraxuvnzql99ngzasgueppp5s7yru7klc5kjwtrwlx2jswm5s9m222w',	0,	0,	1679537451,	NULL,	0,	0);
ELSE
    RAISE NOTICE 'target DB does not contain dimension';
END IF;
END $$;
