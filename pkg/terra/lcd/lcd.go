package lcd

type Lcd[R any] interface {
	ContractState(address string, query string, height ...uint64) ([]byte, error)
	Tx(hash string) (*R, error)
}
