package rules

type RuleType string

const (
	Terraswap RuleType = "terraswap"
	Dezswap   RuleType = "dezswap"
	Starfleit RuleType = "starfleit"
	Unknown   RuleType = ""
)

func ToRuleType(candidate string) RuleType {
	switch RuleType(candidate) {
	case Terraswap:
		return Terraswap
	case Dezswap:
		return Dezswap
	case Starfleit:
		return Starfleit
	default:
		return Unknown
	}
}
