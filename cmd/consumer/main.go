package main

import (
	"context"
	"time"

	"github.com/grenmon2202/greninvestor/charts"
	"github.com/grenmon2202/greninvestor/config"
	"github.com/grenmon2202/greninvestor/db"
	"github.com/grenmon2202/greninvestor/logging"
	"github.com/grenmon2202/greninvestor/symbols"
	"go.uber.org/zap"
)

func processExchange(s symbols.SymbolsStore, exchange string, y *charts.YahooClient, ctx context.Context) {
	logging.Init()
	defer logging.L.Sync()

	logging.L.Info("Processing symbols for exchange", zap.String("exchange", exchange))

	var passedCount int
	var failedCount int

	for _, sym := range s.Symbols {
		logging.L.Info("Processing symbol", zap.String("symbol", sym.Code), zap.String("exchange", exchange))
		if err := db.InsertStockIfNotExists(sym); err != nil {
			logging.L.Warn("Failed to upsert stock metadata", zap.String("symbol", sym.Code), zap.Error(err))
			logging.L.Info("Skipping symbol due to error", zap.String("symbol", sym.Code))
			time.Sleep(time.Duration(config.CANDLE_SCRAPER_SLEEP_S) * time.Second)
			failedCount++
			continue
		}

		candles, err := y.FetchCandles(ctx, sym.Code, config.CANDLE_RANGE, config.CANDLE_INTERVAL)
		if err != nil {
			logging.L.Warn("Failed to fetch candles", zap.String("symbol", sym.Code), zap.Error(err))
			logging.L.Info("Skipping symbol due to error", zap.String("symbol", sym.Code))
			time.Sleep(time.Duration(config.CANDLE_SCRAPER_SLEEP_S) * time.Second)
			failedCount++
			continue
		}

		candlesCleaned := charts.RemoveIncompleteLastCandle(candles, exchange)

		if err := db.InsertMarketData(sym, candlesCleaned); err != nil {
			logging.L.Warn("Failed to insert market data", zap.String("symbol", sym.Code), zap.Error(err))
			logging.L.Info("Skipping symbol due to error", zap.String("symbol", sym.Code))
			time.Sleep(time.Duration(config.CANDLE_SCRAPER_SLEEP_S) * time.Second)
			failedCount++
			continue
		}

		logging.L.Info("Completed processing symbol", zap.String("symbol", sym.Code))
		passedCount++

		time.Sleep(time.Duration(config.CANDLE_SCRAPER_SLEEP_S) * time.Second)
	}

	logging.L.Info("Completed processing exchange", zap.String("exchange", exchange), zap.Int("passed", passedCount), zap.Int("failed", failedCount))
}

func main() {
	ctx := context.Background()
	y := charts.NewYahooClient()

	niftySymbols := symbols.RetrieveSymbolsNifty500()
	processExchange(niftySymbols, "NSE", y, ctx)

	sp500Symbols := symbols.RetrieveSymbolsSP500()
	processExchange(sp500Symbols, "NASDAQ", y, ctx)
}
