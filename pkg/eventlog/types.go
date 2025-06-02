package eventlog

type LogType string

const (
	//TODO add if need more
	Message         LogType = "message"
	ExecuteType     LogType = "execute"
	WasmType        LogType = "wasm"
	TransferType    LogType = "transfer"
	InstantiateType LogType = "instantiate"
	ReplyType       LogType = "reply"

	//columbus 4
	FromContract LogType = "from_contract"
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
