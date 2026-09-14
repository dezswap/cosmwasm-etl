package dex

// ChainName is the chain a chain id belongs to, mainnet and testnets alike. Log finders,
// source stores and query clients are all chain specific, so a chain id missing from
// chainNameByChainId has no parser at all.
type ChainName string

// Values are cosmos chain-registry chain_names, which predate the CONX and ASI Alliance
// rebrands and which list each testnet under its own name; here a testnet shares the name
// of the chain it mirrors.
const (
	ChainNameTerraClassic ChainName = "terra"
	ChainNameTerra2       ChainName = "terra2"
	ChainNameConx         ChainName = "xpla"
	ChainNameAsiAlliance  ChainName = "fetchhub"
	ChainNameUnknown      ChainName = ""
)

var chainNameByChainId = map[string]ChainName{
	"columbus-5":     ChainNameTerraClassic,
	"phoenix-1":      ChainNameTerra2,
	"cube_47-5":      ChainNameConx,
	"dimension_37-1": ChainNameConx,
	"dorado-1":       ChainNameAsiAlliance,
	"fetchhub-4":     ChainNameAsiAlliance,
}

func ChainNameOf(chainId string) ChainName {
	return chainNameByChainId[chainId]
}
