package phoenix

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/dex"
)

func Test_lcd(t *testing.T) {
	lcd := NewLcd("http://office-ubuntu:21317", &http.Client{})
	req := dex.FactoryPairsReq{}
	reqBase64, _ := dex.QueryToBase64Str(req)
	res, _ := lcd.ContractState("terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul", reqBase64, 0)
	fmt.Println(res)
	res2, _ := QueryContractState[dex.FactoryPairsRes](lcd, "terra1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqxl5qul", reqBase64, 0)
	fmt.Println(res2)

	tx, _ := lcd.Tx("3DE3BD9AAEE788551BD5432AB9B32A816720EC0B9E40BBC35788DFC207912511")
	fmt.Print(tx)
}
