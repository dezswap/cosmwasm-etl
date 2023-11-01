package schemas

func (LpHistory) TableName() string {
	return "lp_history"
}

func (Price) TableName() string {
	return "price"
}

func (Route) TableName() string {
	return "route"
}

func (Account) TableName() string {
	return "account"
}

func (PairStatsRecent) TableName() string {
	return "pair_stats_recent"
}

func (PairStats30m) TableName() string {
	return "pair_stats_30m"
}

func (HAccountStats30m) TableName() string {
	return "h_account_stats_30m"
}
