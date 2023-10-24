package datastore

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPairListTypeUnmarshal(t *testing.T) {
	ret := &PairListResponse{}
	err := json.Unmarshal([]byte(factoryResponse), ret)

	assert.NoError(t, err)
	assert.NotNil(t, ret.Pairs[0].AssetInfo[0].GetInfo())
	assert.NotNil(t, ret.Pairs[0].AssetInfo[1].GetInfo())

	for _, unit := range ret.Pairs {
		convertedUnit := unit.Convert()
		assert.NotNil(t, convertedUnit)
	}
}

func TestUnitPoolUnmarshal(t *testing.T) {
	ret := &PoolInfo{}
	err := json.Unmarshal([]byte(poolResponse), ret)

	assert.NoError(t, err)
	assert.NotNil(t, ret.Assets[0].GetInfo())
	assert.NotNil(t, ret.Assets[1].GetInfo())

	convertedRet := ret.Convert()
	assert.NotNil(t, convertedRet)
}

func TestBlockWithTxDTOUnmarshal(t *testing.T) {
	body, err := os.ReadFile("../../collector/datastore/block_1000033.json")
	if err != nil {
		panic(err)
	}

	block := &BlockTxsDTO{}
	err = json.Unmarshal(body, block)
	assert.NoError(t, err)
}

const (
	factoryResponse = `
		{
			"pairs": [
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8",
					"liquidity_token": "terra1ttaekgrc60xc3xcflq069m49lwu79m5t552rjcws48rzhxcr4g6shmdw2v",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1spkm49wd9dqkranhrks4cupecl3rtgeqqljq3qrvrrts2ev2gw6sy5vz3k"
							}
						},
						{
							"token": {
								"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							}
						}
					],
					"contract_addr": "terra1wm9dlwgtufjnjzuuee8ftqy3t9vq728vhyxv0tuqgzk7dt3fmwwsecqh8j",
					"liquidity_token": "terra18jq3nawedex6s67f2rwkz3e7a57a5nxq40r9905yvyran5x7dh6sm7krj2",
					"asset_decimals": [
						0,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1sdglum2dt4f3fmq7jrt2phf2tegmnudc7qqqqujkpqcm9ujuxxkqakv5u8"
							}
						},
						{
							"token": {
								"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							}
						}
					],
					"contract_addr": "terra1qg85dekl59jv723ce54s82v26rteknru5645lfm3n9eytr53570ssrz6js",
					"liquidity_token": "terra1xgkjhtn4d5csgwchma73gtx5yzjxuq0eywz25lj56eqgq4d4r37sygw8wn",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							}
						},
						{
							"token": {
								"contract_addr": "terra1cmf8ytutcwrjrv08zskj9phuh46a3w3nkjax7en4hxezsrdr58lqvzy05q"
							}
						}
					],
					"contract_addr": "terra1gu75wek7kq8h4ee6eztmfu73nr3esl6al0qjawkhya3g57sz6jvsukpj3z",
					"liquidity_token": "terra1a4hcacqpkqgmh94jpe290gekgv6f7euhfcl2fxm22s2r78uck9lqw33t99",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x",
					"liquidity_token": "terra1ma0g752dl0yujasnfs9yrk6uew7d0a2zrgvg62cfnlfftu2y0egqjpj90v",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1rwg5kt6kcyxtz69acjgpeut7dgr4y3r7tvntdxqt03dvpqktrfxq4jrvpq"
							}
						},
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						}
					],
					"contract_addr": "terra1yu58twelefzpkgnzphc99q57kyqqyqhl6fmsc64cz6jy3s90gv8srlvtps",
					"liquidity_token": "terra1hhjh8hj0zdx2qh70gsuse4efer0dq93rwggkwd6w2jqpafa5gw4qhzn047",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"token": {
								"contract_addr": "terra1rcmvfsn77pd6m04ctqj3wcu66pvrw9p265cdl72w4zarfup2rv7qjxhkzl"
							}
						}
					],
					"contract_addr": "terra172v738ut05le2272gm6akv9hw2jqfwfkm7ej7ndy53skxq757s5sraz2ja",
					"liquidity_token": "terra1rwx6w02alc4kaz7xpyg3rlxpjl4g63x5jq292mkxgg65zqpn5llq9etvsk",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1y92wp8u58396mtaejwhtasz24vrcf0lhsznl7rj0nhw0khp8kchscuqgfs"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1mecfcj3fkmsgxqa4eaq5w285u6cn0wqtwzkp9gfhpm3dyt8e3cesrpg5hq",
					"liquidity_token": "terra1gxnp98ghg5mqddw3n0ve6uw3ay9hnt0ks9r3dyucjn0y007u64vqw03d6f",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1ykxe98arwtahe97h9y5nck8gw88kxn2z73gfwlnx5twcd96ct98sqzcsrk"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra15dpd6drrsxt785m4k8frxt088caelz37q3tkpveekh4lvt6j79kq3jrvqs",
					"liquidity_token": "terra1nwju0tcdykx037phw8qh7jzwqal5uk7ekqjkxzpuymkpmc7a4sjs3d3t9a",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1yetl2gafkhtanr6utpmxp0zqtkhkc05k4dgjg0zfyf86p9fzw3ssslv0fj"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1k08qteme5x0gm4932usuet89zzcv3z9kp6jzzn8wgy4qgwk293sqjm09mz",
					"liquidity_token": "terra1a0s7u0zej7jpzuq90yglms75ng5yvq9q20awu44h395hc9k249hsry5d2j",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1xumzh893lfa7ak5qvpwmnle5m5xp47t3suwwa9s0ydqa8d8s5faqn6x7al"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1zdpq84j8ex29wz9tmygqtftplrw87x8wmuyfh0rsy60uq7nadtsq5pjr7y",
					"liquidity_token": "terra1gte4eejaw3hrs2d8pt0zhp0yfd34xp24qdgqumjul29jt5hwl5tsx3qmw7",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra12ezq5402h5n3skhdshjp4f49zzg0saxum8fvvhjhauzas2ezyyrqpznqny"
							}
						},
						{
							"token": {
								"contract_addr": "terra129zjaxquhh3h5upn0clqzdawnze43a9z34ktt4l6um2hf5w0xqjsta42u5"
							}
						}
					],
					"contract_addr": "terra16m2fp6mlt2qtlgv30z4xln8q0grlxr9c0ylwelknnl7rklasw24qkag3za",
					"liquidity_token": "terra14avgfstw3th676cy3mg7kwqd5a09yfwwk02uztqzq4lzu78e738saktxrk",
					"asset_decimals": [
						8,
						8
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1vzd98s9kqdkatahxs7rsd8m474lf2f8ct39zdgd6shj4nh5e6kuskaz2gy"
							}
						},
						{
							"token": {
								"contract_addr": "terra129zjaxquhh3h5upn0clqzdawnze43a9z34ktt4l6um2hf5w0xqjsta42u5"
							}
						}
					],
					"contract_addr": "terra1twpp0zndevlxx3mfn047qmvl824zm5mnqwaye07ts6x5tqyyf9dqx9a63n",
					"liquidity_token": "terra1nvd25azm56h9fk7qn9r0usjsn3g2n34w7dypw52gam8ytgvrcrfslrq2ff",
					"asset_decimals": [
						8,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						},
						{
							"token": {
								"contract_addr": "terra129zjaxquhh3h5upn0clqzdawnze43a9z34ktt4l6um2hf5w0xqjsta42u5"
							}
						}
					],
					"contract_addr": "terra1vrxe77hvfl4k98n9rdv4n47u8hy59tqv9eyrvl0u445ym543g93svz89m3",
					"liquidity_token": "terra1erku46zd9ac98ts6mzj0fxv3rv37c4al5afff4xejhx5cveqefrqa9js3n",
					"asset_decimals": [
						6,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"token": {
								"contract_addr": "terra129zjaxquhh3h5upn0clqzdawnze43a9z34ktt4l6um2hf5w0xqjsta42u5"
							}
						}
					],
					"contract_addr": "terra1n9gqryt5sqlt9rexp569anwqw8end0tqj2jdauu7cwv4jrfkq7eqj70d0n",
					"liquidity_token": "terra1zmlgklwyrzvhgvn5sfrq28fad69wcevjymkea7jgpz8gheuk6qas2w8krk",
					"asset_decimals": [
						6,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"token": {
								"contract_addr": "terra129gzxm65ckt7p9tp3rnq8q0zvaz6m48e5l7qpxtmy2s3fnhcjd0sag3tm3"
							}
						}
					],
					"contract_addr": "terra1gfv5f3r5e9ykhsgz92hx5qa5wtxc6222w7nnj5f26a9ekw3mdzmsa95h0v",
					"liquidity_token": "terra1uvqk5vj9vn4gjemrp0myz4ku49aaemulgaqw7pfe0nuvfwp3gukq4rz3fj",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1vzd98s9kqdkatahxs7rsd8m474lf2f8ct39zdgd6shj4nh5e6kuskaz2gy"
							}
						},
						{
							"token": {
								"contract_addr": "terra12ezq5402h5n3skhdshjp4f49zzg0saxum8fvvhjhauzas2ezyyrqpznqny"
							}
						}
					],
					"contract_addr": "terra1t5ddufyxl6yjedr6me5dusmm29jfgvmxstjk6ul57zk6d0trptjsju2xw2",
					"liquidity_token": "terra14m6nt5dwnxsmvt6pv6am8dkmqrf263acgjq6ex00un3t0qdd9pts2fy2cq",
					"asset_decimals": [
						8,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						},
						{
							"token": {
								"contract_addr": "terra12ezq5402h5n3skhdshjp4f49zzg0saxum8fvvhjhauzas2ezyyrqpznqny"
							}
						}
					],
					"contract_addr": "terra130w7wjnj4ngzwz2x3qd4w2pq99ynzgw9fdxrsjg0cqhg5pw0dphsqg2jtk",
					"liquidity_token": "terra1qcax3h3cadtpqua2uev06euuse8wxsjzfrtlmt0nyp2aw0a7hctq0ww6p9",
					"asset_decimals": [
						6,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"token": {
								"contract_addr": "terra12ezq5402h5n3skhdshjp4f49zzg0saxum8fvvhjhauzas2ezyyrqpznqny"
							}
						}
					],
					"contract_addr": "terra128mfavj89an674grxygf7eyqwqc5y9z849nklvd2v65gs6g2r5hqpyht0u",
					"liquidity_token": "terra1h7kgccvtdf6z2udech63qlvn5j06pw0vny9uq2wl8eqad63vt95q2pkmrq",
					"asset_decimals": [
						6,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						},
						{
							"token": {
								"contract_addr": "terra1vzd98s9kqdkatahxs7rsd8m474lf2f8ct39zdgd6shj4nh5e6kuskaz2gy"
							}
						}
					],
					"contract_addr": "terra13vxj8sffahcwjl02mkw065sydparj086q4vcnw4n5cvqg6wd4r8qd3ngnx",
					"liquidity_token": "terra1nf8q6w0tl3hmnvs3llakr52f9gtlnyfmtt0ynmwweppx4g2y42dqxaqrek",
					"asset_decimals": [
						6,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"token": {
								"contract_addr": "terra1vzd98s9kqdkatahxs7rsd8m474lf2f8ct39zdgd6shj4nh5e6kuskaz2gy"
							}
						}
					],
					"contract_addr": "terra1zrajvdc5yx0fsp429j6ej4nvrq68jjv078n94r00nl67d8cj6kmsalxtwt",
					"liquidity_token": "terra1ep6pquxwgfljxzvgs7l7d0epp8spx3erysymhd8m6u4x94ztjxhqyhmyrp",
					"asset_decimals": [
						6,
						8
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						},
						{
							"native_token": {
								"denom": "ibc/CBF67A2BCF6CAE343FDF251E510C8E18C361FC02B23430C121116E0811835DEF"
							}
						}
					],
					"contract_addr": "terra1req03gy0eyeeg9e7nwjyn0pct6hdqtphy837j784492l4hcsh0vqx2n8az",
					"liquidity_token": "terra1mpyp9t48q2dy6s4lkxwjpy8sgg4r823hwam2tap2ra86hmgrrqyqcf6ehy",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						}
					],
					"contract_addr": "terra1zrs8p04zctj0a0f9azakwwennrqfrkh3l6zkttz9x89e7vehjzmqzg8v7n",
					"liquidity_token": "terra1a0fyanyqm496fpgneqawhlsug6uqfvqg2epnw39q0jdenw3zs8zqlykdyd",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1564y9uxzhast8sk5n47teypy4mxy7fg5vne2msuztsft7qk3pj9sxxuxmc"
							}
						},
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						}
					],
					"contract_addr": "terra1qe36wap4lrwx4yanhvst33lvvxfdryve8c6uwhvks36p07z5qvlq0cx202",
					"liquidity_token": "terra10nnsamvtc5yux6m9utwc6dtee20h8fe8gp06jfqy0ffqtxrk384s4l0rru",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1ee4g63c3sus9hnyyp3p2u3tulzdv5ag68l55q8ej64y4qpwswvus5mtag2"
							}
						},
						{
							"native_token": {
								"denom": "ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							}
						}
					],
					"contract_addr": "terra12jlsxqs89ytrtpm86mc0ey8yl902zhk2vy7e3h9xzfppk3mdd3qqdj9c5t",
					"liquidity_token": "terra1qatnqunnama825l0xf6nmxgts6j27vqfhnzadwecld4mnlumhkkq9q7cn7",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"native_token": {
								"denom": "ibc/CBF67A2BCF6CAE343FDF251E510C8E18C361FC02B23430C121116E0811835DEF"
							}
						}
					],
					"contract_addr": "terra1u3wd9gu7weezw6vwfaaa4q589zjlazg6wt6gyer3lc42tgqrpggqv90c2c",
					"liquidity_token": "terra1w2l4w5p66l5t2nmrmsvz7k4cu50s7e8dc6h59gcxsnmp2tgy7q7smfaxql",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1ee4g63c3sus9hnyyp3p2u3tulzdv5ag68l55q8ej64y4qpwswvus5mtag2"
							}
						},
						{
							"native_token": {
								"denom": "ibc/CBF67A2BCF6CAE343FDF251E510C8E18C361FC02B23430C121116E0811835DEF"
							}
						}
					],
					"contract_addr": "terra160lewlf0ygzvjkjar5n8wxulnh8phsu6vsq4sk8e3ln3pqz58juq22ywwy",
					"liquidity_token": "terra1vz29w25qu5lzfghz89yy6cq7jaj5snjf5p66qcmp4hza87jcstfqylf5er",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1dtaqwlmzlk3jku5un6h6rfunttmwsqnfz7evvdf4pwr0wypsl68q6nuam0"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1dtaakf99dllanxn0sg0ryft4j9fsewypgns5gavev6tz49mw0wds8cg89y",
					"liquidity_token": "terra1q3647qp780u7y2zvau5fn748zqxsfm4kr6lcvr5jjev5a77kchxsjy2m4x",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"native_token": {
								"denom": "uluna"
							}
						},
						{
							"token": {
								"contract_addr": "terra1wu5cts7zr3sfmwlxfh7an3nthrx9cuz8fx7xfesmdudg2kzcwmhsaw24r8"
							}
						}
					],
					"contract_addr": "terra1u9hwyy9yjjhh03hr4sqvk9trzrgjnmjesql9m05t03pz4yjr52gqgjlv8s",
					"liquidity_token": "terra1qzn7zc70c5npg5tyc6pvwrpvfp7p9utkpcwlaugn9avn6r49609suh5g0r",
					"asset_decimals": [
						6,
						6
					]
				},
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1spkm49wd9dqkranhrks4cupecl3rtgeqqljq3qrvrrts2ev2gw6sy5vz3k"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1w8246pdk9tf9d2dnu4lty5m8v3ptjltrm46vh8kd6yhr8k4ad2yskdqs6x",
					"liquidity_token": "terra1gdj85sxs0tqhap50pv6jr6vrku4vqfrx5k62x0fu4gxt4l66qjgqqyz386",
					"asset_decimals": [
						0,
						6
					]
				}
			]
		}
	`

	poolResponse = `
		{
			"assets": [
				{
					"info": {
						"token": {
							"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
						}
					},
					"amount": "50000000000000"
				},
				{
					"info": {
						"native_token": {
							"denom": "uluna"
						}
					},
					"amount": "5000000"
				}
			],
			"total_share": "15811388300"
		}
	`

	lightFactoryResp = `
		{
			"pairs": [
				{
					"asset_infos": [
						{
							"token": {
								"contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							}
						},
						{
							"native_token": {
								"denom": "uluna"
							}
						}
					],
					"contract_addr": "terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8",
					"liquidity_token": "terra1ttaekgrc60xc3xcflq069m49lwu79m5t552rjcws48rzhxcr4g6shmdw2v",
					"asset_decimals": [
						6,
						6
					]
				}
			]
		}
	`

	lightWholePoolResp = `
		{
			"pairs": {
				"terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							},
							"amount":"50000000000000"
						},
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"uluna"
							},
							"amount":"5000000"
						}
					],
					"total_share":"15811388300"
				}
			}
		}
	`

	unitPoolResp = `
		{
			"assets":[
				{
					"info":{
						"type":"token",
						"denom_or_address":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
					},
					"amount":"50000000000000"
				},
				{
					"info":{
						"type":"native_token",
						"denom_or_address":"uluna"
					},
					"amount":"5000000"
				}
			],
			"total_share":"15811388300"
		}
	`

	poolStatusFromS3 = `
		{
			"pairs":{
				"terra15dpd6drrsxt785m4k8frxt088caelz37q3tkpveekh4lvt6j79kq3jrvqs":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1ykxe98arwtahe97h9y5nck8gw88kxn2z73gfwlnx5twcd96ct98sqzcsrk"
							},
							"amount":"7859655667"
						},
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"uluna"
							},
							"amount":"12666690"
						}
					],
					"total_share":"315227766"
				},
				"terra172v738ut05le2272gm6akv9hw2jqfwfkm7ej7ndy53skxq757s5sraz2ja":{
					"assets":[
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"uluna"
							},
							"amount":"1000000000"
						},
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1rcmvfsn77pd6m04ctqj3wcu66pvrw9p265cdl72w4zarfup2rv7qjxhkzl"
							},
							"amount":"6000000000"
						}
					],
					"total_share":"2449489742"
				},
				"terra1gu75wek7kq8h4ee6eztmfu73nr3esl6al0qjawkhya3g57sz6jvsukpj3z":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							},
							"amount":"0"
						},
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1cmf8ytutcwrjrv08zskj9phuh46a3w3nkjax7en4hxezsrdr58lqvzy05q"
							},
							"amount":"0"
						}
					],
					"total_share":"0"
				},
				"terra1j08452mqwadp8xu25kn9rleyl2gufgfjnv0sn8dvynynakkjukcqsc244x":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1zwv6feuzhy6a9wekh96cd57lsarmqlwxdypdsplw6zhfncqw6ftqynf7kp"
							},
							"amount":"0"
						},
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"uluna"
							},
							"amount":"0"
						}
					],
					"total_share":"0"
				},
				"terra1k08qteme5x0gm4932usuet89zzcv3z9kp6jzzn8wgy4qgwk293sqjm09mz":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1yetl2gafkhtanr6utpmxp0zqtkhkc05k4dgjg0zfyf86p9fzw3ssslv0fj"
							},
							"amount":"0"
						},
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"uluna"
							},
							"amount":"0"
						}
					],
					"total_share":"0"
				},
				"terra1mecfcj3fkmsgxqa4eaq5w285u6cn0wqtwzkp9gfhpm3dyt8e3cesrpg5hq":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1y92wp8u58396mtaejwhtasz24vrcf0lhsznl7rj0nhw0khp8kchscuqgfs"
							},
							"amount":"0"
						},
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"uluna"
							},
							"amount":"0"
						}
					],
					"total_share":"0"
				},
				"terra1qg85dekl59jv723ce54s82v26rteknru5645lfm3n9eytr53570ssrz6js":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1sdglum2dt4f3fmq7jrt2phf2tegmnudc7qqqqujkpqcm9ujuxxkqakv5u8"
							},
							"amount":"0"
						},
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							},
							"amount":"0"
						}
					],
					"total_share":"0"
				},
				"terra1wm9dlwgtufjnjzuuee8ftqy3t9vq728vhyxv0tuqgzk7dt3fmwwsecqh8j":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1spkm49wd9dqkranhrks4cupecl3rtgeqqljq3qrvrrts2ev2gw6sy5vz3k"
							},
							"amount":"0"
						},
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							},
							"amount":"0"
						}
					],
					"total_share":"0"
				},
				"terra1xjv2pmf26yaz3wqft7caafgckdg4eflzsw56aqhdcjw58qx0v2mqux87t8":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
							},
							"amount":"41691666666666"
						},
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"uluna"
							},
							"amount":"6000000"
						}
					],
					"total_share":"15811388300"
				},
				"terra1yu58twelefzpkgnzphc99q57kyqqyqhl6fmsc64cz6jy3s90gv8srlvtps":{
					"assets":[
						{
							"info":{
								"type":"token",
								"denom_or_address":"terra1rwg5kt6kcyxtz69acjgpeut7dgr4y3r7tvntdxqt03dvpqktrfxq4jrvpq"
							},
							"amount":"0"
						},
						{
							"info":{
								"type":"native_token",
								"denom_or_address":"ibc/B3504E092456BA618CC28AC671A71FB08C6CA0FD0BE7C8A5B5A3E2DD933CC9E4"
							},
							"amount":"0"
						}
					],
					"total_share":"0"
				}
			}
		}
	`
)
