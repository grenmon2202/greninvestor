package ledger

type Portfolio struct {
	Name           string
	Wallet         float64
	PortfolioValue float64
}

type Holdings struct {
	Name             string
	Symbol           string
	BuyTs            int64
	EntryPoint       float64
	NumShares        int
	QuantInvestedINR float64
	InrUsdConvRatio  float64
}
