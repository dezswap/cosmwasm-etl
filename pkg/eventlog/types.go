package eventlog

type LogType string

const (
	//TODO add if need more
	Message      LogType = "message"
	WasmType     LogType = "wasm"
	TransferType LogType = "transfer"

	//columbus 4
	FromContract LogType = "from_contract"

	// columbus 5
	TaxPaymentType LogType = "tax_payment"
)

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type Attributes []Attribute

type LogResult struct {
	Type       LogType    `json:"type"`
	Attributes Attributes `json:"attributes"`
}
type LogResults []LogResult

type MatchedItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type MatchedResult []MatchedItem
type MatchedResults []MatchedResult
