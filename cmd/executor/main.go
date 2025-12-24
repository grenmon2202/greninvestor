package main

import (
	"time"

	"github.com/grenmon2202/greninvestor/config"
	"github.com/grenmon2202/greninvestor/db"
	"github.com/grenmon2202/greninvestor/logging"
	"github.com/grenmon2202/greninvestor/scripts"
	"github.com/grenmon2202/greninvestor/symbols"
	"go.uber.org/zap"
)

func processExchange(s symbols.SymbolsStore, exchange string) {
	logging.Init()
	defer logging.L.Sync()

	logging.L.Info("Processing symbols for exchange", zap.String("exchange", exchange))

	currTs := time.Unix(1766131200, 0)

	switch exchange {
	case "NSE":
		if currTs.Hour() > config.NSE_END_HOUR || (currTs.Hour() == config.NSE_END_HOUR && currTs.Minute() >= config.NSE_END_MINUTE) {
			logging.L.Info("Market closed for NSE today")
		}
	case "NASDAQ":
		if currTs.Hour() > config.NASDAQ_END_HOUR || (currTs.Hour() == config.NASDAQ_END_HOUR && currTs.Minute() >= config.NASDAQ_END_MINUTE) {
			logging.L.Info("Market closed for NASDAQ today")
		}
	default:
		logging.L.Error("Unsupported exchange", zap.String("exchange", exchange))
		return
	}

	for _, sym := range s.Symbols {
		logging.L.Info("Processing symbol", zap.String("symbol", sym.Code), zap.String("exchange", exchange))

		startTs := currTs.AddDate(0, 0, -config.DS_DATA_RANGE_D)
		candles, err := db.FetchMarketData(sym.Code, startTs, currTs)

		if err != nil {
			logging.L.Error("Failed to fetch market data", zap.String("symbol", sym.Code), zap.Error(err))
			continue
		}

		if len(candles) == 0 {
			logging.L.Warn("No market data found, skipping symbol", zap.String("symbol", sym.Code))
			continue
		}

		logging.L.Info("Fetched market data", zap.String("symbol", sym.Code), zap.Int("num_candles", len(candles)))

		if startTs.Unix()-candles[len(candles)-1].T.Unix() > config.MAX_DELTA_FROM_LAST_CANDLE_S {
			logging.L.Warn("Insufficient recent market data, skipping symbol", zap.String("symbol", sym.Code))
			continue
		}

		logging.L.Info("Executing trend following strategy", zap.String("symbol", sym.Code))
		decision, confidence, err := scripts.ExecuteTrendFollowingStrategy(candles, "buy", sym.Code)
		if err != nil {
			logging.L.Error("Strategy execution failed", zap.String("symbol", sym.Code), zap.Error(err))
			continue
		}

		logging.L.Info("Strategy executed", zap.String("symbol", sym.Code), zap.Bool("decision", decision), zap.Float32("confidence", confidence))
	}
}

func main() {
	niftySymbols := symbols.RetrieveSymbolsNifty500()
	processExchange(niftySymbols, "NSE")

	sp500Symbols := symbols.RetrieveSymbolsSP500()
	processExchange(sp500Symbols, "NASDAQ")
}
