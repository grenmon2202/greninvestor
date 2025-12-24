package charts

import (
	"time"

	"github.com/grenmon2202/greninvestor/config"
)

type Candle struct {
	T time.Time
	O float64
	H float64
	L float64
	C float64
	V float64
}

func RemoveIncompleteLastCandle(candles []Candle, exchange string) []Candle {
	if len(candles) == 0 {
		return candles
	}

	minutePadding := config.GetMinutePadding(exchange)

	lastCandle := candles[len(candles)-1]

	if lastCandle.T.Minute() != minutePadding && lastCandle.T.Second() != 0 {
		return candles[:len(candles)-1]
	}
	return candles
}
