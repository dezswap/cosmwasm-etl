package fcd

import (
	"fmt"
	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/dezswap/cosmwasm-etl/pkg/terra/cosmos45"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBlock(t *testing.T) {
	expected := FcdBlockRes{
		Height: 10335499,
		Txs: []FcdBlockTxRes{
			{
				Code: 0,
				Data: "122E0A2C2F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374526573706F6E7365",
				Logs: []FcdBlockTxLogRes{
					{
						Log: "",
						Events: eventlog.LogResults{
							{Type: eventlog.Message, Attributes: eventlog.Attributes{}},
							{Type: eventlog.ExecuteType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.WasmType, Attributes: eventlog.Attributes{}},
						},
						MsgIndex: 0,
					},
				},
			},
			{
				Code: 0,
				Data: "122E0A2C2F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374526573706F6E7365",
				Logs: []FcdBlockTxLogRes{
					{
						Log: "",
						Events: eventlog.LogResults{
							{Type: eventlog.Message, Attributes: eventlog.Attributes{
								{Key: "", Value: ""},
							}},
							{Type: eventlog.ExecuteType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.WasmType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.InstantiateType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.InstantiateType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.ReplyType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.WasmType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.ReplyType, Attributes: eventlog.Attributes{}},
							{Type: eventlog.WasmType, Attributes: eventlog.Attributes{}},
						},
						MsgIndex: 0,
					},
				},
			},
		},
	}

	// https://fcd-terra.tfl.foundation/v1/blocks/10335499?chainId=phoenix-1
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `
{
  "chainId": "phoenix-1",
  "height": 10335499,
  "timestamp": "2024-05-15T13:19:07.719Z",
  "proposer": {
    "moniker": "SCV-Security",
    "identity": "135E966C9A604B66",
    "operatorAddress": "terravaloper10c04ysz9uznx2mkenuk3j3esjczyqh0j783nzt"
  },
  "txs": [
    {
      "id": "21441410",
      "tx": {
        "body": {
          "memo": "",
          "messages": [
            {
              "msg": {
                "feed_price": {
                  "prices": [
                    [
                      "terra170e8mepwmndwfgs5897almdewrt6phnkksktlf958s90eh055xvsrndvku",
                      "0.68797612720902013"
                    ],
                    [
                      "terra173z5ggu6k6slyumrrf59rd3ywmpu6hdfftwpqlkc7fp549yk9fmqzqyepj",
                      "0.68797612720902013"
                    ],
                    [
                      "terra1uq59f5lhzg6ut605ntevvf2a8kg9t2xk2873lgx6pweagkw76r4sdzj6ap",
                      "0.68797612720902013"
                    ],
                    [
                      "terra18mls96hhatg6k03zg29tz02a76q3w66z4qsa8pfww6hupszlhqns6fm9ad",
                      "0.68797612720902013"
                    ],
                    [
                      "terra1kd85952285xfdlp5ck8nt62vuvur8cem9h3svm7yptpsvmr9tuusqpm2sw",
                      "0.01766643517505207"
                    ],
                    [
                      "terra1ze3c86la6wynenrqewhq4j9hw24yrvardudsl5mkq3mhgs6ag4cqrva0pg",
                      "0.01766177667429403"
                    ],
                    [
                      "ibc/08095CEDEA29977C9DD0CE9A48329FDA622C183359D5F90CF04CC4FF80CBE431",
                      "0.84593552251565707"
                    ],
                    [
                      "ibc/B3F639855EE7478750CC8F82072307ED6E131A8EFF20345E1D136B50C4E5EC36",
                      "0.01935243672130604"
                    ],
                    [
                      "ibc/517E13F14A1245D4DE8CF467ADD4DA0058974CDCC880FA6AE536DBCA1D16D84E",
                      "0.01873046652379174"
                    ],
                    [
                      "uluna",
                      "0.68792624646329259"
                    ],
                    [
                      "ibc/36A02FFC4E74DF4F64305130C3DFA1B06BEAC775648927AA44467C76A77AB8DB",
                      "0.01766644143693975"
                    ],
                    [
                      "terra1v697322n7fny777xke4zkq8stcct2rn9v2esfpfs9xl98upvs98s4k7y3l",
                      "3.34313530548892679"
                    ],
                    [
                      "terra1hl4tqxa99w9ee2qs3umu9udmaq30yzz5cscqcpe3l60lvtqf4qxsdswgdh",
                      "3.35216881013557311"
                    ]
                  ]
                }
              },
              "@type": "/cosmwasm.wasm.v1.MsgExecuteContract",
              "funds": [],
              "sender": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877",
              "contract": "terra1gp3a4cz9magxuvj6n0x8ra8jqc79zqvquw85xrn0suwvml2cqs4q4l7ss7"
            }
          ],
          "timeout_height": "0",
          "extension_options": [],
          "non_critical_extension_options": []
        },
        "@type": "/cosmos.tx.v1beta1.Tx",
        "auth_info": {
          "fee": {
            "payer": "",
            "amount": [
              {
                "denom": "uluna",
                "amount": "4015"
              }
            ],
            "granter": "",
            "gas_limit": "267650"
          },
          "tip": null,
          "signer_infos": [
            {
              "sequence": "679729",
              "mode_info": {
                "single": {
                  "mode": "SIGN_MODE_DIRECT"
                }
              },
              "public_key": {
                "key": "A2se4OMvZZBR1onUdspKEIHsbbW6ojuHtvJpoOmGXhYx",
                "@type": "/cosmos.crypto.secp256k1.PubKey"
              }
            }
          ]
        },
        "signatures": [
          "daWectbi0dSJ8SoyhaLT/p7t2uo+wZVi/Q0eqErGd75ifPN6xSd4nRddYDqzmMVPSj/KvDAV8Q/Sc2oV3tp+qA=="
        ]
      },
      "code": 0,
      "data": "122E0A2C2F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374526573706F6E7365",
      "info": "",
      "logs": [
        {
          "log": "",
          "events": [
            {
              "type": "message",
              "attributes": [
                {
                  "key": "action",
                  "value": "/cosmwasm.wasm.v1.MsgExecuteContract"
                },
                {
                  "key": "sender",
                  "value": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877"
                },
                {
                  "key": "module",
                  "value": "wasm"
                }
              ]
            },
            {
              "type": "execute",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1gp3a4cz9magxuvj6n0x8ra8jqc79zqvquw85xrn0suwvml2cqs4q4l7ss7"
                }
              ]
            },
            {
              "type": "wasm",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1gp3a4cz9magxuvj6n0x8ra8jqc79zqvquw85xrn0suwvml2cqs4q4l7ss7"
                },
                {
                  "key": "action",
                  "value": "feed_prices"
                },
                {
                  "key": "asset",
                  "value": "terra170e8mepwmndwfgs5897almdewrt6phnkksktlf958s90eh055xvsrndvku"
                },
                {
                  "key": "price",
                  "value": "0.68797612720902013"
                },
                {
                  "key": "asset",
                  "value": "terra173z5ggu6k6slyumrrf59rd3ywmpu6hdfftwpqlkc7fp549yk9fmqzqyepj"
                },
                {
                  "key": "price",
                  "value": "0.68797612720902013"
                },
                {
                  "key": "asset",
                  "value": "terra1uq59f5lhzg6ut605ntevvf2a8kg9t2xk2873lgx6pweagkw76r4sdzj6ap"
                },
                {
                  "key": "price",
                  "value": "0.68797612720902013"
                },
                {
                  "key": "asset",
                  "value": "terra18mls96hhatg6k03zg29tz02a76q3w66z4qsa8pfww6hupszlhqns6fm9ad"
                },
                {
                  "key": "price",
                  "value": "0.68797612720902013"
                },
                {
                  "key": "asset",
                  "value": "terra1kd85952285xfdlp5ck8nt62vuvur8cem9h3svm7yptpsvmr9tuusqpm2sw"
                },
                {
                  "key": "price",
                  "value": "0.01766643517505207"
                },
                {
                  "key": "asset",
                  "value": "terra1ze3c86la6wynenrqewhq4j9hw24yrvardudsl5mkq3mhgs6ag4cqrva0pg"
                },
                {
                  "key": "price",
                  "value": "0.01766177667429403"
                },
                {
                  "key": "asset",
                  "value": "ibc/08095CEDEA29977C9DD0CE9A48329FDA622C183359D5F90CF04CC4FF80CBE431"
                },
                {
                  "key": "price",
                  "value": "0.84593552251565707"
                },
                {
                  "key": "asset",
                  "value": "ibc/B3F639855EE7478750CC8F82072307ED6E131A8EFF20345E1D136B50C4E5EC36"
                },
                {
                  "key": "price",
                  "value": "0.01935243672130604"
                },
                {
                  "key": "asset",
                  "value": "ibc/517E13F14A1245D4DE8CF467ADD4DA0058974CDCC880FA6AE536DBCA1D16D84E"
                },
                {
                  "key": "price",
                  "value": "0.01873046652379174"
                },
                {
                  "key": "asset",
                  "value": "uluna"
                },
                {
                  "key": "price",
                  "value": "0.68792624646329259"
                },
                {
                  "key": "asset",
                  "value": "ibc/36A02FFC4E74DF4F64305130C3DFA1B06BEAC775648927AA44467C76A77AB8DB"
                },
                {
                  "key": "price",
                  "value": "0.01766644143693975"
                },
                {
                  "key": "asset",
                  "value": "terra1v697322n7fny777xke4zkq8stcct2rn9v2esfpfs9xl98upvs98s4k7y3l"
                },
                {
                  "key": "price",
                  "value": "3.34313530548892679"
                },
                {
                  "key": "asset",
                  "value": "terra1hl4tqxa99w9ee2qs3umu9udmaq30yzz5cscqcpe3l60lvtqf4qxsdswgdh"
                },
                {
                  "key": "price",
                  "value": "3.35216881013557311"
                }
              ]
            }
          ],
          "msg_index": 0
        }
      ],
      "events": [
        {
          "type": "coin_spent",
          "attributes": [
            {
              "key": "spender",
              "index": true,
              "value": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877"
            },
            {
              "key": "amount",
              "index": true,
              "value": "4015uluna"
            }
          ]
        },
        {
          "type": "coin_received",
          "attributes": [
            {
              "key": "receiver",
              "index": true,
              "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa"
            },
            {
              "key": "amount",
              "index": true,
              "value": "4015uluna"
            }
          ]
        },
        {
          "type": "transfer",
          "attributes": [
            {
              "key": "recipient",
              "index": true,
              "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa"
            },
            {
              "key": "sender",
              "index": true,
              "value": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877"
            },
            {
              "key": "amount",
              "index": true,
              "value": "4015uluna"
            }
          ]
        },
        {
          "type": "message",
          "attributes": [
            {
              "key": "sender",
              "index": true,
              "value": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877"
            }
          ]
        },
        {
          "type": "tx",
          "attributes": [
            {
              "key": "fee",
              "index": true,
              "value": "4015uluna"
            },
            {
              "key": "fee_payer",
              "index": true,
              "value": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877"
            }
          ]
        },
        {
          "type": "tx",
          "attributes": [
            {
              "key": "acc_seq",
              "index": true,
              "value": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877/679729"
            }
          ]
        },
        {
          "type": "tx",
          "attributes": [
            {
              "key": "signature",
              "index": true,
              "value": "daWectbi0dSJ8SoyhaLT/p7t2uo+wZVi/Q0eqErGd75ifPN6xSd4nRddYDqzmMVPSj/KvDAV8Q/Sc2oV3tp+qA=="
            }
          ]
        },
        {
          "type": "message",
          "attributes": [
            {
              "key": "action",
              "index": true,
              "value": "/cosmwasm.wasm.v1.MsgExecuteContract"
            },
            {
              "key": "sender",
              "index": true,
              "value": "terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877"
            },
            {
              "key": "module",
              "index": true,
              "value": "wasm"
            }
          ]
        },
        {
          "type": "execute",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1gp3a4cz9magxuvj6n0x8ra8jqc79zqvquw85xrn0suwvml2cqs4q4l7ss7"
            }
          ]
        },
        {
          "type": "wasm",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1gp3a4cz9magxuvj6n0x8ra8jqc79zqvquw85xrn0suwvml2cqs4q4l7ss7"
            },
            {
              "key": "action",
              "index": true,
              "value": "feed_prices"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra170e8mepwmndwfgs5897almdewrt6phnkksktlf958s90eh055xvsrndvku"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.68797612720902013"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra173z5ggu6k6slyumrrf59rd3ywmpu6hdfftwpqlkc7fp549yk9fmqzqyepj"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.68797612720902013"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra1uq59f5lhzg6ut605ntevvf2a8kg9t2xk2873lgx6pweagkw76r4sdzj6ap"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.68797612720902013"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra18mls96hhatg6k03zg29tz02a76q3w66z4qsa8pfww6hupszlhqns6fm9ad"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.68797612720902013"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra1kd85952285xfdlp5ck8nt62vuvur8cem9h3svm7yptpsvmr9tuusqpm2sw"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.01766643517505207"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra1ze3c86la6wynenrqewhq4j9hw24yrvardudsl5mkq3mhgs6ag4cqrva0pg"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.01766177667429403"
            },
            {
              "key": "asset",
              "index": true,
              "value": "ibc/08095CEDEA29977C9DD0CE9A48329FDA622C183359D5F90CF04CC4FF80CBE431"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.84593552251565707"
            },
            {
              "key": "asset",
              "index": true,
              "value": "ibc/B3F639855EE7478750CC8F82072307ED6E131A8EFF20345E1D136B50C4E5EC36"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.01935243672130604"
            },
            {
              "key": "asset",
              "index": true,
              "value": "ibc/517E13F14A1245D4DE8CF467ADD4DA0058974CDCC880FA6AE536DBCA1D16D84E"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.01873046652379174"
            },
            {
              "key": "asset",
              "index": true,
              "value": "uluna"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.68792624646329259"
            },
            {
              "key": "asset",
              "index": true,
              "value": "ibc/36A02FFC4E74DF4F64305130C3DFA1B06BEAC775648927AA44467C76A77AB8DB"
            },
            {
              "key": "price",
              "index": true,
              "value": "0.01766644143693975"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra1v697322n7fny777xke4zkq8stcct2rn9v2esfpfs9xl98upvs98s4k7y3l"
            },
            {
              "key": "price",
              "index": true,
              "value": "3.34313530548892679"
            },
            {
              "key": "asset",
              "index": true,
              "value": "terra1hl4tqxa99w9ee2qs3umu9udmaq30yzz5cscqcpe3l60lvtqf4qxsdswgdh"
            },
            {
              "key": "price",
              "index": true,
              "value": "3.35216881013557311"
            }
          ]
        }
      ],
      "height": "10335499",
      "txhash": "61B4433CD0F2E4FC8A1F1E954ECFCDC47421DE79926242E4A595944C62B82170",
      "raw_log": "[{\"msg_index\":0,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmwasm.wasm.v1.MsgExecuteContract\"},{\"key\":\"sender\",\"value\":\"terra14uhlzf6v5c84y79ussxfc5lqmpc3f242yqr877\"},{\"key\":\"module\",\"value\":\"wasm\"}]},{\"type\":\"execute\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1gp3a4cz9magxuvj6n0x8ra8jqc79zqvquw85xrn0suwvml2cqs4q4l7ss7\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1gp3a4cz9magxuvj6n0x8ra8jqc79zqvquw85xrn0suwvml2cqs4q4l7ss7\"},{\"key\":\"action\",\"value\":\"feed_prices\"},{\"key\":\"asset\",\"value\":\"terra170e8mepwmndwfgs5897almdewrt6phnkksktlf958s90eh055xvsrndvku\"},{\"key\":\"price\",\"value\":\"0.68797612720902013\"},{\"key\":\"asset\",\"value\":\"terra173z5ggu6k6slyumrrf59rd3ywmpu6hdfftwpqlkc7fp549yk9fmqzqyepj\"},{\"key\":\"price\",\"value\":\"0.68797612720902013\"},{\"key\":\"asset\",\"value\":\"terra1uq59f5lhzg6ut605ntevvf2a8kg9t2xk2873lgx6pweagkw76r4sdzj6ap\"},{\"key\":\"price\",\"value\":\"0.68797612720902013\"},{\"key\":\"asset\",\"value\":\"terra18mls96hhatg6k03zg29tz02a76q3w66z4qsa8pfww6hupszlhqns6fm9ad\"},{\"key\":\"price\",\"value\":\"0.68797612720902013\"},{\"key\":\"asset\",\"value\":\"terra1kd85952285xfdlp5ck8nt62vuvur8cem9h3svm7yptpsvmr9tuusqpm2sw\"},{\"key\":\"price\",\"value\":\"0.01766643517505207\"},{\"key\":\"asset\",\"value\":\"terra1ze3c86la6wynenrqewhq4j9hw24yrvardudsl5mkq3mhgs6ag4cqrva0pg\"},{\"key\":\"price\",\"value\":\"0.01766177667429403\"},{\"key\":\"asset\",\"value\":\"ibc/08095CEDEA29977C9DD0CE9A48329FDA622C183359D5F90CF04CC4FF80CBE431\"},{\"key\":\"price\",\"value\":\"0.84593552251565707\"},{\"key\":\"asset\",\"value\":\"ibc/B3F639855EE7478750CC8F82072307ED6E131A8EFF20345E1D136B50C4E5EC36\"},{\"key\":\"price\",\"value\":\"0.01935243672130604\"},{\"key\":\"asset\",\"value\":\"ibc/517E13F14A1245D4DE8CF467ADD4DA0058974CDCC880FA6AE536DBCA1D16D84E\"},{\"key\":\"price\",\"value\":\"0.01873046652379174\"},{\"key\":\"asset\",\"value\":\"uluna\"},{\"key\":\"price\",\"value\":\"0.68792624646329259\"},{\"key\":\"asset\",\"value\":\"ibc/36A02FFC4E74DF4F64305130C3DFA1B06BEAC775648927AA44467C76A77AB8DB\"},{\"key\":\"price\",\"value\":\"0.01766644143693975\"},{\"key\":\"asset\",\"value\":\"terra1v697322n7fny777xke4zkq8stcct2rn9v2esfpfs9xl98upvs98s4k7y3l\"},{\"key\":\"price\",\"value\":\"3.34313530548892679\"},{\"key\":\"asset\",\"value\":\"terra1hl4tqxa99w9ee2qs3umu9udmaq30yzz5cscqcpe3l60lvtqf4qxsdswgdh\"},{\"key\":\"price\",\"value\":\"3.35216881013557311\"}]}]}]",
      "gas_used": "247568",
      "codespace": "",
      "timestamp": "2024-05-15T13:19:07Z",
      "gas_wanted": "267650"
    },
    {
      "id": "21441411",
      "tx": {
        "body": {
          "memo": "",
          "messages": [
            {
              "msg": {
                "create_pair": {
                  "assets": [
                    {
                      "info": {
                        "token": {
                          "contract_addr": "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y"
                        }
                      },
                      "amount": "0"
                    },
                    {
                      "info": {
                        "token": {
                          "contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
                        }
                      },
                      "amount": "0"
                    }
                  ]
                }
              },
              "@type": "/cosmwasm.wasm.v1.MsgExecuteContract",
              "funds": [],
              "sender": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3",
              "contract": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            }
          ],
          "timeout_height": "0",
          "extension_options": [],
          "non_critical_extension_options": []
        },
        "@type": "/cosmos.tx.v1beta1.Tx",
        "auth_info": {
          "fee": {
            "payer": "",
            "amount": [
              {
                "denom": "uluna",
                "amount": "28045"
              }
            ],
            "granter": "",
            "gas_limit": "1869632"
          },
          "tip": null,
          "signer_infos": [
            {
              "sequence": "2010",
              "mode_info": {
                "single": {
                  "mode": "SIGN_MODE_DIRECT"
                }
              },
              "public_key": {
                "key": "A0Su422qKTJHyJgMQlxUIBlCf/id9lX6/WJYxH3VPbQz",
                "@type": "/cosmos.crypto.secp256k1.PubKey"
              }
            }
          ]
        },
        "signatures": [
          "skYg/B1KwxmLItkCefJWQbFAtBFspXAsAB2DEWHBsQcJ82/7MRQHe4pQ1pF+Ty0EtpmXhQCX/+ssVuUPfk91mg=="
        ]
      },
      "code": 0,
      "data": "122E0A2C2F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374526573706F6E7365",
      "info": "",
      "logs": [
        {
          "log": "",
          "events": [
            {
              "type": "message",
              "attributes": [
                {
                  "key": "action",
                  "value": "/cosmwasm.wasm.v1.MsgExecuteContract"
                },
                {
                  "key": "sender",
                  "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
                },
                {
                  "key": "module",
                  "value": "wasm"
                }
              ]
            },
            {
              "type": "execute",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
                }
              ]
            },
            {
              "type": "wasm",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
                },
                {
                  "key": "action",
                  "value": "create_pair"
                },
                {
                  "key": "pair",
                  "value": "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
                }
              ]
            },
            {
              "type": "instantiate",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
                },
                {
                  "key": "code_id",
                  "value": "1723"
                }
              ]
            },
            {
              "type": "instantiate",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
                },
                {
                  "key": "code_id",
                  "value": "4"
                }
              ]
            },
            {
              "type": "reply",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
                }
              ]
            },
            {
              "type": "wasm",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
                },
                {
                  "key": "liquidity_token_addr",
                  "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
                }
              ]
            },
            {
              "type": "reply",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
                }
              ]
            },
            {
              "type": "wasm",
              "attributes": [
                {
                  "key": "_contract_address",
                  "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
                },
                {
                  "key": "pair_contract_addr",
                  "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
                },
                {
                  "key": "liquidity_token_addr",
                  "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
                }
              ]
            }
          ],
          "msg_index": 0
        }
      ],
      "events": [
        {
          "type": "coin_spent",
          "attributes": [
            {
              "key": "spender",
              "index": true,
              "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
            },
            {
              "key": "amount",
              "index": true,
              "value": "28045uluna"
            }
          ]
        },
        {
          "type": "coin_received",
          "attributes": [
            {
              "key": "receiver",
              "index": true,
              "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa"
            },
            {
              "key": "amount",
              "index": true,
              "value": "28045uluna"
            }
          ]
        },
        {
          "type": "transfer",
          "attributes": [
            {
              "key": "recipient",
              "index": true,
              "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa"
            },
            {
              "key": "sender",
              "index": true,
              "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
            },
            {
              "key": "amount",
              "index": true,
              "value": "28045uluna"
            }
          ]
        },
        {
          "type": "message",
          "attributes": [
            {
              "key": "sender",
              "index": true,
              "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
            }
          ]
        },
        {
          "type": "tx",
          "attributes": [
            {
              "key": "fee",
              "index": true,
              "value": "28045uluna"
            },
            {
              "key": "fee_payer",
              "index": true,
              "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
            }
          ]
        },
        {
          "type": "tx",
          "attributes": [
            {
              "key": "acc_seq",
              "index": true,
              "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3/2010"
            }
          ]
        },
        {
          "type": "tx",
          "attributes": [
            {
              "key": "signature",
              "index": true,
              "value": "skYg/B1KwxmLItkCefJWQbFAtBFspXAsAB2DEWHBsQcJ82/7MRQHe4pQ1pF+Ty0EtpmXhQCX/+ssVuUPfk91mg=="
            }
          ]
        },
        {
          "type": "message",
          "attributes": [
            {
              "key": "action",
              "index": true,
              "value": "/cosmwasm.wasm.v1.MsgExecuteContract"
            },
            {
              "key": "sender",
              "index": true,
              "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
            },
            {
              "key": "module",
              "index": true,
              "value": "wasm"
            }
          ]
        },
        {
          "type": "execute",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            }
          ]
        },
        {
          "type": "wasm",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            },
            {
              "key": "action",
              "index": true,
              "value": "create_pair"
            },
            {
              "key": "pair",
              "index": true,
              "value": "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
            }
          ]
        },
        {
          "type": "instantiate",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            },
            {
              "key": "code_id",
              "index": true,
              "value": "1723"
            }
          ]
        },
        {
          "type": "instantiate",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
            },
            {
              "key": "code_id",
              "index": true,
              "value": "4"
            }
          ]
        },
        {
          "type": "reply",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            }
          ]
        },
        {
          "type": "wasm",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            },
            {
              "key": "liquidity_token_addr",
              "index": true,
              "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
            }
          ]
        },
        {
          "type": "reply",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            }
          ]
        },
        {
          "type": "wasm",
          "attributes": [
            {
              "key": "_contract_address",
              "index": true,
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            },
            {
              "key": "pair_contract_addr",
              "index": true,
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            },
            {
              "key": "liquidity_token_addr",
              "index": true,
              "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
            }
          ]
        }
      ],
      "height": "10335499",
      "txhash": "6B0D0AC8684F43F145A1A2F2F00DD7BD5EC6509043254FDAAE0435E2D93241E4",
      "raw_log": "[{\"msg_index\":0,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmwasm.wasm.v1.MsgExecuteContract\"},{\"key\":\"sender\",\"value\":\"terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3\"},{\"key\":\"module\",\"value\":\"wasm\"}]},{\"type\":\"execute\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"},{\"key\":\"action\",\"value\":\"create_pair\"},{\"key\":\"pair\",\"value\":\"terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg\"}]},{\"type\":\"instantiate\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"code_id\",\"value\":\"1723\"}]},{\"type\":\"instantiate\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"},{\"key\":\"code_id\",\"value\":\"4\"}]},{\"type\":\"reply\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"liquidity_token_addr\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"}]},{\"type\":\"reply\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"},{\"key\":\"pair_contract_addr\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"liquidity_token_addr\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"}]}]}]",
      "gas_used": "632612",
      "codespace": "",
      "timestamp": "2024-05-15T13:19:07Z",
      "gas_wanted": "1869632"
    }
  ]
}`)
	}))
	defer mockServer.Close()

	client := http.Client{}
	fcd := New(mockServer.URL, &client)
	tx, err := fcd.Block(10335499, "phoenix-1")

	assert.NoError(t, err)
	assert.Equal(t, expected, *tx)
}

func TestTx(t *testing.T) {
	expected := FcdTxRes{
		Tx: cosmos45.LcdTx{Body: cosmos45.LcdTxBody{Messages: []cosmos45.LcdTxMessage{
			{Type: "/cosmwasm.wasm.v1.MsgExecuteContract", Sender: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
		}}},
		Code: 0,
		Logs: []cosmos45.LcdTxLogRes{
			{MsgIndex: 0, Events: []cosmos45.LcdTxEventRes{
				{Type: "message", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "action", Value: "/cosmwasm.wasm.v1.MsgExecuteContract"},
					{Key: "sender", Value: "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"},
					{Key: "module", Value: "wasm"},
				}},
				{Type: "execute", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
				}},
				{Type: "wasm", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
					{Key: "action", Value: "create_pair"},
					{Key: "pair", Value: "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"},
				}},
				{Type: "instantiate", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
					{Key: "code_id", Value: "1723"},
				}},
				{Type: "instantiate", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"},
					{Key: "code_id", Value: "4"},
				}},
				{Type: "reply", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
				}},
				{Type: "wasm", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
					{Key: "liquidity_token_addr", Value: "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"},
				}},
				{Type: "reply", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
				}},
				{Type: "wasm", Attributes: []cosmos45.LcdTxAttributeRes{
					{Key: "_contract_address", Value: "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"},
					{Key: "pair_contract_addr", Value: "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"},
					{Key: "liquidity_token_addr", Value: "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"},
				}},
			}},
		},
		Height: "10335499",
		TxHash: "6B0D0AC8684F43F145A1A2F2F00DD7BD5EC6509043254FDAAE0435E2D93241E4",
		RawLog: "[{\"msg_index\":0,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmwasm.wasm.v1.MsgExecuteContract\"},{\"key\":\"sender\",\"value\":\"terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3\"},{\"key\":\"module\",\"value\":\"wasm\"}]},{\"type\":\"execute\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"},{\"key\":\"action\",\"value\":\"create_pair\"},{\"key\":\"pair\",\"value\":\"terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg\"}]},{\"type\":\"instantiate\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"code_id\",\"value\":\"1723\"}]},{\"type\":\"instantiate\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"},{\"key\":\"code_id\",\"value\":\"4\"}]},{\"type\":\"reply\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"liquidity_token_addr\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"}]},{\"type\":\"reply\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"},{\"key\":\"pair_contract_addr\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"liquidity_token_addr\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"}]}]}]",
	}

	// https://fcd-terra.tfl.foundation/v1/tx/6B0D0AC8684F43F145A1A2F2F00DD7BD5EC6509043254FDAAE0435E2D93241E4
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `
{
  "tx": {
    "body": {
      "memo": "",
      "messages": [
        {
          "msg": {
            "create_pair": {
              "assets": [
                {
                  "info": {
                    "token": {
                      "contract_addr": "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y"
                    }
                  },
                  "amount": "0"
                },
                {
                  "info": {
                    "token": {
                      "contract_addr": "terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
                    }
                  },
                  "amount": "0"
                }
              ]
            }
          },
          "@type": "/cosmwasm.wasm.v1.MsgExecuteContract",
          "funds": [],
          "sender": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3",
          "contract": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
        }
      ],
      "timeout_height": "0",
      "extension_options": [],
      "non_critical_extension_options": []
    },
    "@type": "/cosmos.tx.v1beta1.Tx",
    "auth_info": {
      "fee": {
        "payer": "",
        "amount": [
          {
            "denom": "uluna",
            "amount": "28045"
          }
        ],
        "granter": "",
        "gas_limit": "1869632"
      },
      "tip": null,
      "signer_infos": [
        {
          "sequence": "2010",
          "mode_info": {
            "single": {
              "mode": "SIGN_MODE_DIRECT"
            }
          },
          "public_key": {
            "key": "A0Su422qKTJHyJgMQlxUIBlCf/id9lX6/WJYxH3VPbQz",
            "@type": "/cosmos.crypto.secp256k1.PubKey"
          }
        }
      ]
    },
    "signatures": [
      "skYg/B1KwxmLItkCefJWQbFAtBFspXAsAB2DEWHBsQcJ82/7MRQHe4pQ1pF+Ty0EtpmXhQCX/+ssVuUPfk91mg=="
    ]
  },
  "code": 0,
  "data": "122E0A2C2F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374526573706F6E7365",
  "info": "",
  "logs": [
    {
      "log": "",
      "events": [
        {
          "type": "message",
          "attributes": [
            {
              "key": "action",
              "value": "/cosmwasm.wasm.v1.MsgExecuteContract"
            },
            {
              "key": "sender",
              "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
            },
            {
              "key": "module",
              "value": "wasm"
            }
          ]
        },
        {
          "type": "execute",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            }
          ]
        },
        {
          "type": "wasm",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            },
            {
              "key": "action",
              "value": "create_pair"
            },
            {
              "key": "pair",
              "value": "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
            }
          ]
        },
        {
          "type": "instantiate",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            },
            {
              "key": "code_id",
              "value": "1723"
            }
          ]
        },
        {
          "type": "instantiate",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
            },
            {
              "key": "code_id",
              "value": "4"
            }
          ]
        },
        {
          "type": "reply",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            }
          ]
        },
        {
          "type": "wasm",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            },
            {
              "key": "liquidity_token_addr",
              "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
            }
          ]
        },
        {
          "type": "reply",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            }
          ]
        },
        {
          "type": "wasm",
          "attributes": [
            {
              "key": "_contract_address",
              "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
            },
            {
              "key": "pair_contract_addr",
              "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
            },
            {
              "key": "liquidity_token_addr",
              "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
            }
          ]
        }
      ],
      "msg_index": 0
    }
  ],
  "events": [
    {
      "type": "coin_spent",
      "attributes": [
        {
          "key": "spender",
          "index": true,
          "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
        },
        {
          "key": "amount",
          "index": true,
          "value": "28045uluna"
        }
      ]
    },
    {
      "type": "coin_received",
      "attributes": [
        {
          "key": "receiver",
          "index": true,
          "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa"
        },
        {
          "key": "amount",
          "index": true,
          "value": "28045uluna"
        }
      ]
    },
    {
      "type": "transfer",
      "attributes": [
        {
          "key": "recipient",
          "index": true,
          "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa"
        },
        {
          "key": "sender",
          "index": true,
          "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
        },
        {
          "key": "amount",
          "index": true,
          "value": "28045uluna"
        }
      ]
    },
    {
      "type": "message",
      "attributes": [
        {
          "key": "sender",
          "index": true,
          "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
        }
      ]
    },
    {
      "type": "tx",
      "attributes": [
        {
          "key": "fee",
          "index": true,
          "value": "28045uluna"
        },
        {
          "key": "fee_payer",
          "index": true,
          "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
        }
      ]
    },
    {
      "type": "tx",
      "attributes": [
        {
          "key": "acc_seq",
          "index": true,
          "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3/2010"
        }
      ]
    },
    {
      "type": "tx",
      "attributes": [
        {
          "key": "signature",
          "index": true,
          "value": "skYg/B1KwxmLItkCefJWQbFAtBFspXAsAB2DEWHBsQcJ82/7MRQHe4pQ1pF+Ty0EtpmXhQCX/+ssVuUPfk91mg=="
        }
      ]
    },
    {
      "type": "message",
      "attributes": [
        {
          "key": "action",
          "index": true,
          "value": "/cosmwasm.wasm.v1.MsgExecuteContract"
        },
        {
          "key": "sender",
          "index": true,
          "value": "terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3"
        },
        {
          "key": "module",
          "index": true,
          "value": "wasm"
        }
      ]
    },
    {
      "type": "execute",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
        }
      ]
    },
    {
      "type": "wasm",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
        },
        {
          "key": "action",
          "index": true,
          "value": "create_pair"
        },
        {
          "key": "pair",
          "index": true,
          "value": "terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg"
        }
      ]
    },
    {
      "type": "instantiate",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
        },
        {
          "key": "code_id",
          "index": true,
          "value": "1723"
        }
      ]
    },
    {
      "type": "instantiate",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
        },
        {
          "key": "code_id",
          "index": true,
          "value": "4"
        }
      ]
    },
    {
      "type": "reply",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
        }
      ]
    },
    {
      "type": "wasm",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
        },
        {
          "key": "liquidity_token_addr",
          "index": true,
          "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
        }
      ]
    },
    {
      "type": "reply",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
        }
      ]
    },
    {
      "type": "wasm",
      "attributes": [
        {
          "key": "_contract_address",
          "index": true,
          "value": "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul"
        },
        {
          "key": "pair_contract_addr",
          "index": true,
          "value": "terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3"
        },
        {
          "key": "liquidity_token_addr",
          "index": true,
          "value": "terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut"
        }
      ]
    }
  ],
  "height": "10335499",
  "txhash": "6B0D0AC8684F43F145A1A2F2F00DD7BD5EC6509043254FDAAE0435E2D93241E4",
  "raw_log": "[{\"msg_index\":0,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmwasm.wasm.v1.MsgExecuteContract\"},{\"key\":\"sender\",\"value\":\"terra1vzpwguqcsg9ejmjz0paqw2ekgm73v6apn3vsr3\"},{\"key\":\"module\",\"value\":\"wasm\"}]},{\"type\":\"execute\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"},{\"key\":\"action\",\"value\":\"create_pair\"},{\"key\":\"pair\",\"value\":\"terra1ysd87nayjuelxj4wvp4wnp9as0mwszzkje6a9z6f3xx2903ghnsq4hm50y-terra1qj5hs3e86qn4vm9dvtgtlkdp550r0rayk9wpay44mfw3gn3tr8nq5jw3dg\"}]},{\"type\":\"instantiate\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"code_id\",\"value\":\"1723\"}]},{\"type\":\"instantiate\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"},{\"key\":\"code_id\",\"value\":\"4\"}]},{\"type\":\"reply\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"liquidity_token_addr\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"}]},{\"type\":\"reply\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul\"},{\"key\":\"pair_contract_addr\",\"value\":\"terra1vs6ywu3h37353a0mtjdjsv4w4cv3ejzlhplj8qdd5suhqrhdwn4qafymj3\"},{\"key\":\"liquidity_token_addr\",\"value\":\"terra14nln3d42h0wz8xxhsws026j69fau35glhngyw3g36p6n8v3zx4fsnx63ut\"}]}]}]",
  "gas_used": "632612",
  "codespace": "",
  "timestamp": "2024-05-15T13:19:07Z",
  "gas_wanted": "1869632",
  "chainId": "phoenix-1"
}`)
	}))
	defer mockServer.Close()

	client := http.Client{}
	fcd := New(mockServer.URL, &client)
	tx, err := fcd.Tx("61B4433CD0F2E4FC8A1F1E954ECFCDC47421DE79926242E4A595944C62B82170")

	assert.NoError(t, err)
	assert.Equal(t, expected, *tx)
}
