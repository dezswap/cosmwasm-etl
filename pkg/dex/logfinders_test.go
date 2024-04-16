package dex

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dezswap/cosmwasm-etl/pkg/eventlog"
	"github.com/stretchr/testify/assert"
)

func Test_LogFinders(t *testing.T) {
	var logFinder eventlog.LogFinder
	var eventLogs eventlog.LogResults

	setUp := func(rawLogsStr string, pairs map[string]bool, finderFunc func(map[string]bool) (eventlog.LogFinder, error)) {
		var err error
		logFinder = nil
		eventLogs = eventlog.LogResults{}
		logFinder, err = finderFunc(pairs)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal([]byte(rawLogsStr), &eventLogs); err != nil {
			panic(err)
		}
	}

	tcs := []struct {
		rawLogStr         string
		pairs             map[string]bool
		finderFunc        func(map[string]bool) (eventlog.LogFinder, error)
		expectedResultLen int
		matchedLen        int
		errMsg            string
	}{
		//InitialProvide
		{PairInitialProvideRawLogStr, map[string]bool{"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp": true}, CreatePairInitialProvideRuleFinder, 1, PairInitialProvideMatchedLen, "must match once"},
	}

	for idx, tc := range tcs {
		errMsg := fmt.Sprintf("idx(%d): %s", idx, tc.errMsg)
		assert := assert.New(t)

		setUp(tc.rawLogStr, tc.pairs, tc.finderFunc)
		matchedResults := logFinder.FindFromLogs(eventLogs)
		assert.Len(matchedResults, tc.expectedResultLen, errMsg)
		if tc.expectedResultLen > 0 {
			assert.Len(matchedResults[0], tc.matchedLen, "must return all matched value")
		}
	}

}

const PairInitialProvideRawLogStr = `[
    {"type":"execute","attributes":[{"key":"_contract_address","value":"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp"},{"key":"_contract_address","value":"xpla1p3dsd5k7cl0p0jhtj0s00vrf45s4c6j3wtjsaaalatmr2jwl7p0sy3sd26"},{"key":"_contract_address","value":"xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5"},{"key":"_contract_address","value":"xpla1he85n9h0mcnzhpegj76wwcyjv626tced0zkp58wakjc7d3fm50xq8sywg6"},{"key":"_contract_address","value":"xpla1p3dsd5k7cl0p0jhtj0s00vrf45s4c6j3wtjsaaalatmr2jwl7p0sy3sd26"}]},
    {"type":"message","attributes":[{"key":"action","value":"/cosmwasm.wasm.v1.MsgExecuteContract"},{"key":"module","value":"wasm"},{"key":"sender","value":"xpla1g8hkzkgfa3uq0cg9d6h99jk5nlg92lwx2jme2l"}]},
    {"type":"wasm","attributes":[{"key":"_contract_address","value":"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp"},{"key":"action","value":"provide_liquidity"},{"key":"sender","value":"xpla1g8hkzkgfa3uq0cg9d6h99jk5nlg92lwx2jme2l"},{"key":"receiver","value":"xpla1g8hkzkgfa3uq0cg9d6h99jk5nlg92lwx2jme2l"},{"key":"assets","value":"500000000000xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5, 19605600000000xpla1he85n9h0mcnzhpegj76wwcyjv626tced0zkp58wakjc7d3fm50xq8sywg6"},{"key":"share","value":"3130942349155"},{"key":"refund_assets","value":"0xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5, 0xpla1he85n9h0mcnzhpegj76wwcyjv626tced0zkp58wakjc7d3fm50xq8sywg6"},{"key":"_contract_address","value":"xpla1p3dsd5k7cl0p0jhtj0s00vrf45s4c6j3wtjsaaalatmr2jwl7p0sy3sd26"},{"key":"action","value":"mint"},{"key":"amount","value":"1000"},{"key":"to","value":"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp"},{"key":"_contract_address","value":"xpla1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5"},{"key":"action","value":"transfer_from"},{"key":"amount","value":"500000000000"},{"key":"by","value":"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp"},{"key":"from","value":"xpla1g8hkzkgfa3uq0cg9d6h99jk5nlg92lwx2jme2l"},{"key":"to","value":"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp"},{"key":"_contract_address","value":"xpla1he85n9h0mcnzhpegj76wwcyjv626tced0zkp58wakjc7d3fm50xq8sywg6"},{"key":"action","value":"transfer_from"},{"key":"amount","value":"19605600000000"},{"key":"by","value":"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp"},{"key":"from","value":"xpla1g8hkzkgfa3uq0cg9d6h99jk5nlg92lwx2jme2l"},{"key":"to","value":"xpla10x8w4n9cvg4f63fjrm0yc6c54a4p3csuk0uv0hskz8g3aljufl6qlrs6gp"},{"key":"_contract_address","value":"xpla1p3dsd5k7cl0p0jhtj0s00vrf45s4c6j3wtjsaaalatmr2jwl7p0sy3sd26"},{"key":"action","value":"mint"},{"key":"amount","value":"3130942349155"},{"key":"to","value":"xpla1g8hkzkgfa3uq0cg9d6h99jk5nlg92lwx2jme2l"}]}]`
