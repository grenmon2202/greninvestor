package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/grenmon2202/greninvestor/config"
	"github.com/grenmon2202/greninvestor/db"
	"github.com/grenmon2202/greninvestor/ledger"
	"github.com/grenmon2202/greninvestor/logging"
	"github.com/grenmon2202/greninvestor/scripts"
	"github.com/grenmon2202/greninvestor/symbols"
	"go.uber.org/zap"
)

const (
	currentTimeLayoutHint = time.RFC3339
)

type RunStats struct {
	ExchangesTotal        int
	ExchangesClosed       int
	ProcessedExchanges    []string
	ClosedExchanges       []string
	PortfoliosLoaded      int
	SymbolsProcessed      int
	HoldingsChecked       int
	BuySignals            int
	SellSignals           int
	BuyOrders             int
	SellOrders            int
	SharesBought          int
	SharesSold            int
	RequestedBuyShares    int
	AmountSpent           float64
	AmountEarned          float64
	RealizedPnL           float64
	SkippedNoData         int
	SkippedStaleData      int
	SkippedInvalidBuySize int
	SkippedUnaffordable   int
	StrategyErrors        int
}

func ensurePortfolio(name string, initialWallet float64) (ledger.Portfolio, error) {
	portfolio, err := db.FetchPortfolio(name)
	if err == nil {
		return portfolio, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		if err := db.CreateNewPortfolio(name, initialWallet); err != nil {
			return ledger.Portfolio{}, err
		}
		return db.FetchPortfolio(name)
	}

	return ledger.Portfolio{}, err
}

func exchangeLocalTime(currTs time.Time, exchange string) (time.Time, error) {
	location, err := time.LoadLocation(config.GetExchangeTimezone(exchange))
	if err != nil {
		return time.Time{}, err
	}
	return currTs.In(location), nil
}

func marketSessionBounds(currTs time.Time, exchange string) (time.Time, time.Time, error) {
	localTime, err := exchangeLocalTime(currTs, exchange)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	switch exchange {
	case "NSE":
		start := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), config.NSE_START_HOUR, config.NSE_START_MINUTE, 0, 0, localTime.Location())
		end := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), config.NSE_END_HOUR, config.NSE_END_MINUTE, 0, 0, localTime.Location())
		return start, end, nil
	case "NASDAQ":
		start := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), config.NASDAQ_START_HOUR, config.NASDAQ_START_MINUTE, 0, 0, localTime.Location())
		end := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), config.NASDAQ_END_HOUR, config.NASDAQ_END_MINUTE, 0, 0, localTime.Location())
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported exchange: %s", exchange)
	}
}

func isMarketClosed(currTs time.Time, exchange string) bool {
	localTime, err := exchangeLocalTime(currTs, exchange)
	if err != nil {
		logging.L.Error("Failed to resolve exchange timezone", zap.String("exchange", exchange), zap.Error(err))
		return true
	}

	if localTime.Weekday() == time.Saturday || localTime.Weekday() == time.Sunday {
		return true
	}

	sessionStart, sessionEnd, err := marketSessionBounds(currTs, exchange)
	if err != nil {
		logging.L.Error("Failed to compute market session", zap.String("exchange", exchange), zap.Error(err))
		return true
	}

	return localTime.Before(sessionStart) || !localTime.Before(sessionEnd)
}

func parseCurrentTime() (time.Time, error) {
	currentTimeRaw := flag.String("current-time", "", "Override the current executor time in RFC3339 format")
	flag.Parse()

	if *currentTimeRaw == "" {
		return time.Now(), nil
	}

	parsedTime, err := time.Parse(time.RFC3339, *currentTimeRaw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --current-time value %q: expected format %s: %w", *currentTimeRaw, currentTimeLayoutHint, err)
	}

	return parsedTime, nil
}

func computeBuyShares(wallet float64, buyPrice float64, requestedShares int) int {
	if buyPrice <= 0 || requestedShares <= 0 || wallet < buyPrice {
		return 0
	}

	affordableShares := int(wallet / buyPrice)
	if affordableShares <= 0 {
		return 0
	}

	if requestedShares < affordableShares {
		return requestedShares
	}

	return affordableShares
}

func refreshPortfolioValue(portfolio ledger.Portfolio, initialWallet float64, asOf time.Time) (ledger.Portfolio, db.PortfolioHistoryRecord, error) {
	holdings, err := db.FetchHoldings(portfolio.Name)
	if err != nil {
		return ledger.Portfolio{}, db.PortfolioHistoryRecord{}, err
	}

	totalValue := portfolio.Wallet
	holdingsValue := 0.0
	totalShares := 0
	for _, holding := range holdings {
		latestPrice, err := db.FetchLatestMarketPrice(holding.Symbol, asOf)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				latestPrice = holding.EntryPoint
			} else {
				return ledger.Portfolio{}, db.PortfolioHistoryRecord{}, err
			}
		}

		holdingValue := latestPrice * float64(holding.NumShares)
		totalValue += holdingValue
		holdingsValue += holdingValue
		totalShares += holding.NumShares
	}

	if err := db.UpdatePortfolioValue(portfolio.Name, totalValue); err != nil {
		return ledger.Portfolio{}, db.PortfolioHistoryRecord{}, err
	}

	previousPortfolioValue, err := db.FetchPreviousPortfolioValue(portfolio.Name, asOf.Unix())
	pnlFromLastRun := 0.0
	if err == nil {
		pnlFromLastRun = totalValue - previousPortfolioValue
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ledger.Portfolio{}, db.PortfolioHistoryRecord{}, err
	}

	portfolio.PortfolioValue = totalValue
	historyRecord := db.PortfolioHistoryRecord{
		RunTs:          asOf.Unix(),
		Portfolio:      portfolio.Name,
		Wallet:         portfolio.Wallet,
		HoldingsValue:  holdingsValue,
		PortfolioValue: portfolio.PortfolioValue,
		CumulativePnL:  totalValue - initialWallet,
		PnLFromLastRun: pnlFromLastRun,
		HoldingsCount:  len(holdings),
		TotalShares:    totalShares,
	}

	return portfolio, historyRecord, nil
}

func logRunSummary(currTs time.Time, stats RunStats, startedAt time.Time, finishedAt time.Time) {
	processedExchanges := strings.Join(stats.ProcessedExchanges, ",")
	closedExchanges := strings.Join(stats.ClosedExchanges, ",")
	logging.L.Info(
		"Executor run summary",
		zap.Time("simulation_time", currTs),
		zap.Time("started_at", startedAt),
		zap.Time("finished_at", finishedAt),
		zap.Duration("duration", finishedAt.Sub(startedAt)),
		zap.String("processed_exchanges", processedExchanges),
		zap.String("closed_exchanges", closedExchanges),
		zap.Int("exchanges_total", stats.ExchangesTotal),
		zap.Int("exchanges_closed", stats.ExchangesClosed),
		zap.Int("portfolios_loaded", stats.PortfoliosLoaded),
		zap.Int("symbols_processed", stats.SymbolsProcessed),
		zap.Int("holdings_checked", stats.HoldingsChecked),
		zap.Int("buy_signals", stats.BuySignals),
		zap.Int("sell_signals", stats.SellSignals),
		zap.Int("buy_orders", stats.BuyOrders),
		zap.Int("sell_orders", stats.SellOrders),
		zap.Int("shares_bought", stats.SharesBought),
		zap.Int("shares_sold", stats.SharesSold),
		zap.Int("requested_buy_shares", stats.RequestedBuyShares),
		zap.Float64("amount_spent", stats.AmountSpent),
		zap.Float64("amount_earned", stats.AmountEarned),
		zap.Float64("realized_pnl", stats.RealizedPnL),
		zap.Int("skipped_no_data", stats.SkippedNoData),
		zap.Int("skipped_stale_data", stats.SkippedStaleData),
		zap.Int("skipped_invalid_buy_size", stats.SkippedInvalidBuySize),
		zap.Int("skipped_unaffordable_buy", stats.SkippedUnaffordable),
		zap.Int("strategy_errors", stats.StrategyErrors),
	)
}

func processExchange(s symbols.SymbolsStore, exchange string, currTs time.Time, stats *RunStats) {
	logging.L.Info("Processing symbols for exchange", zap.String("exchange", exchange))
	stats.ExchangesTotal++

	if isMarketClosed(currTs, exchange) {
		logging.L.Info("Market closed for exchange", zap.String("exchange", exchange))
		stats.ExchangesClosed++
		if !slices.Contains(stats.ClosedExchanges, exchange) {
			stats.ClosedExchanges = append(stats.ClosedExchanges, exchange)
		}
		return
	}

	if !slices.Contains(stats.ProcessedExchanges, exchange) {
		stats.ProcessedExchanges = append(stats.ProcessedExchanges, exchange)
	}

	strategies := config.GetEnabledStrategies()
	if len(strategies) == 0 {
		logging.L.Warn("No enabled strategies found, skipping exchange", zap.String("exchange", exchange))
		return
	}

	portfoliosByStrategy := make(map[string]ledger.Portfolio, len(strategies))
	for _, strategy := range strategies {
		portfolio, err := ensurePortfolio(strategy.PortfolioName, strategy.InitialWallet)
		if err != nil {
			logging.L.Error(
				"Failed to load or create strategy portfolio",
				zap.String("strategy", strategy.Name),
				zap.String("portfolio", strategy.PortfolioName),
				zap.Error(err),
			)
			stats.StrategyErrors++
			continue
		}

		portfoliosByStrategy[strategy.Name] = portfolio
		stats.PortfoliosLoaded++
	}

	if len(portfoliosByStrategy) == 0 {
		logging.L.Error("No strategy portfolio available, skipping exchange", zap.String("exchange", exchange))
		return
	}

	startTs := currTs.AddDate(0, 0, -config.DS_DATA_RANGE_D)

	for _, sym := range s.Symbols {
		logging.L.Info("Processing symbol", zap.String("symbol", sym.Code), zap.String("exchange", exchange))
		stats.SymbolsProcessed++

		candles, err := db.FetchMarketData(sym.Code, startTs, currTs)

		if err != nil {
			logging.L.Error("Failed to fetch market data", zap.String("symbol", sym.Code), zap.Error(err))
			stats.StrategyErrors++
			continue
		}

		if len(candles) == 0 {
			logging.L.Warn("No market data found, skipping symbol", zap.String("symbol", sym.Code))
			stats.SkippedNoData++
			continue
		}

		logging.L.Info("Fetched market data", zap.String("symbol", sym.Code), zap.Int("num_candles", len(candles)))

		lastCandle := candles[len(candles)-1]
		if currTs.Unix()-lastCandle.T.Unix() > config.MAX_DELTA_FROM_LAST_CANDLE_S {
			logging.L.Warn("Insufficient recent market data, skipping symbol", zap.String("symbol", sym.Code))
			stats.SkippedStaleData++
			continue
		}

		for _, strategy := range strategies {
			portfolio, ok := portfoliosByStrategy[strategy.Name]
			if !ok {
				continue
			}

			holdings, err := db.FetchHoldingsBySymbol(portfolio.Name, sym.Code)
			if err != nil {
				logging.L.Error("Failed to fetch holdings", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Error(err))
				stats.StrategyErrors++
				continue
			}
			stats.HoldingsChecked += len(holdings)

			for _, holding := range holdings {
				portData := map[string]any{
					"entry_point": holding.EntryPoint,
					"T":           time.Unix(holding.BuyTs, 0).UTC().Format(time.RFC3339),
				}

				sellResult, err := scripts.ExecuteScript(candles, "sell", sym.Code, strategy.ScriptPath, strategy.ConfigPath, portData)
				if err != nil {
					logging.L.Error("Sell strategy execution failed", zap.String("strategy", strategy.Name), zap.String("symbol", sym.Code), zap.Error(err))
					stats.StrategyErrors++
					continue
				}

				if !sellResult.Decision {
					continue
				}
				stats.SellSignals++

				sellPrice := lastCandle.C
				proceeds := sellPrice * float64(holding.NumShares)
				pnl := proceeds - holding.QuantInvestedINR
				newWallet := portfolio.Wallet + proceeds

				if err := db.UpdatePortfolioWallet(portfolio.Name, newWallet); err != nil {
					logging.L.Error("Failed to update wallet after sell", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.Error(err))
					stats.StrategyErrors++
					continue
				}

				if err := db.DeleteHolding(holding.Name, holding.Symbol, holding.BuyTs); err != nil {
					logging.L.Error("Failed to delete holding after sell", zap.String("strategy", strategy.Name), zap.String("portfolio", holding.Name), zap.String("symbol", holding.Symbol), zap.Error(err))
					stats.StrategyErrors++
					continue
				}

				if err := db.InsertTrade(portfolio.Name, sym.Code, "sell", currTs.Unix(), sellPrice, holding.NumShares, &pnl); err != nil {
					logging.L.Error("Failed to insert sell trade", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Error(err))
					stats.StrategyErrors++
				}

				portfolio.Wallet = newWallet
				portfoliosByStrategy[strategy.Name] = portfolio
				stats.SellOrders++
				stats.SharesSold += holding.NumShares
				stats.AmountEarned += proceeds
				stats.RealizedPnL += pnl
				logging.L.Info("Sold holding", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Int("shares", holding.NumShares), zap.Float64("price", sellPrice), zap.Float64("confidence", float64(sellResult.Confidence)), zap.Float64("wallet", portfolio.Wallet), zap.Float64("pnl", pnl))
			}

			logging.L.Info("Executing strategy", zap.String("strategy", strategy.Name), zap.String("symbol", sym.Code))
			buyPrice := lastCandle.C
			buyContext := map[string]any{
				"wallet":       portfolio.Wallet,
				"latest_price": buyPrice,
			}

			buyResult, err := scripts.ExecuteScript(candles, "buy", sym.Code, strategy.ScriptPath, strategy.ConfigPath, buyContext)
			if err != nil {
				logging.L.Error("Strategy execution failed", zap.String("strategy", strategy.Name), zap.String("symbol", sym.Code), zap.Error(err))
				stats.StrategyErrors++
				continue
			}

			if !buyResult.Decision {
				logging.L.Info("No buy signal", zap.String("strategy", strategy.Name), zap.String("symbol", sym.Code), zap.Float64("confidence", float64(buyResult.Confidence)))
				continue
			}
			stats.BuySignals++
			stats.RequestedBuyShares += buyResult.Shares

			executedShares := computeBuyShares(portfolio.Wallet, buyPrice, buyResult.Shares)
			if executedShares <= 0 {
				if buyResult.Shares <= 0 {
					stats.SkippedInvalidBuySize++
				} else {
					stats.SkippedUnaffordable++
				}
				logging.L.Warn("Invalid or unaffordable buy size", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Float64("wallet", portfolio.Wallet), zap.Float64("price", buyPrice), zap.Int("requested_shares", buyResult.Shares))
				continue
			}

			if executedShares < buyResult.Shares {
				logging.L.Info("Clamped buy size to affordable shares", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Int("requested_shares", buyResult.Shares), zap.Int("executed_shares", executedShares), zap.Float64("wallet", portfolio.Wallet), zap.Float64("price", buyPrice))
			}

			cost := buyPrice * float64(executedShares)

			holding := ledger.Holdings{
				Name:             portfolio.Name,
				Symbol:           sym.Code,
				BuyTs:            currTs.Unix(),
				EntryPoint:       buyPrice,
				NumShares:        executedShares,
				QuantInvestedINR: cost,
				InrUsdConvRatio:  1,
			}

			if err := db.InsertHolding(holding); err != nil {
				logging.L.Error("Failed to insert holding", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Error(err))
				stats.StrategyErrors++
				continue
			}

			newWallet := portfolio.Wallet - cost
			if err := db.UpdatePortfolioWallet(portfolio.Name, newWallet); err != nil {
				logging.L.Error("Failed to update wallet after buy", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.Error(err))
				stats.StrategyErrors++
				continue
			}

			if err := db.InsertTrade(portfolio.Name, sym.Code, "buy", currTs.Unix(), buyPrice, executedShares, nil); err != nil {
				logging.L.Error("Failed to insert buy trade", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Error(err))
				stats.StrategyErrors++
			}

			portfolio.Wallet = newWallet
			portfoliosByStrategy[strategy.Name] = portfolio
			stats.BuyOrders++
			stats.SharesBought += executedShares
			stats.AmountSpent += cost
			logging.L.Info("Bought symbol", zap.String("strategy", strategy.Name), zap.String("portfolio", portfolio.Name), zap.String("symbol", sym.Code), zap.Int("shares", executedShares), zap.Float64("price", buyPrice), zap.Float64("confidence", float64(buyResult.Confidence)), zap.Float64("wallet", portfolio.Wallet))

			logging.L.Info("Strategy executed", zap.String("strategy", strategy.Name), zap.String("symbol", sym.Code), zap.Bool("decision", buyResult.Decision), zap.Float64("confidence", float64(buyResult.Confidence)), zap.Int("requested_shares", buyResult.Shares), zap.Int("executed_shares", executedShares))
		}
	}
}

func main() {
	logging.Init()
	defer logging.L.Sync()

	startedAt := time.Now()
	currTs, err := parseCurrentTime()
	if err != nil {
		logging.L.Fatal("Failed to parse current time override", zap.Error(err))
	}

	if err := db.EnsureRunsTable(); err != nil {
		logging.L.Fatal("Failed to ensure runs table", zap.Error(err))
	}
	if err := db.EnsurePortfolioTable(); err != nil {
		logging.L.Fatal("Failed to ensure portfolio table", zap.Error(err))
	}
	if err := db.EnsurePortfolioHistoryTable(); err != nil {
		logging.L.Fatal("Failed to ensure portfolio history table", zap.Error(err))
	}

	stats := RunStats{}
	niftySymbols := symbols.RetrieveSymbolsNifty500()
	processExchange(niftySymbols, "NSE", currTs, &stats)

	sp500Symbols := symbols.RetrieveSymbolsSP500()
	processExchange(sp500Symbols, "NASDAQ", currTs, &stats)

	for _, strategy := range config.GetEnabledStrategies() {
		portfolio, err := db.FetchPortfolio(strategy.PortfolioName)
		if err != nil {
			logging.L.Error("Failed to load portfolio for valuation refresh", zap.String("portfolio", strategy.PortfolioName), zap.Error(err))
			continue
		}

		portfolio, historyRecord, err := refreshPortfolioValue(portfolio, strategy.InitialWallet, currTs)
		if err != nil {
			logging.L.Error("Failed to refresh portfolio value", zap.String("portfolio", strategy.PortfolioName), zap.Error(err))
			continue
		}

		if err := db.InsertPortfolioHistory(historyRecord); err != nil {
			logging.L.Error("Failed to insert portfolio history snapshot", zap.String("portfolio", strategy.PortfolioName), zap.Error(err))
			continue
		}

		logging.L.Info("Updated portfolio valuation", zap.String("portfolio", portfolio.Name), zap.Float64("wallet", portfolio.Wallet), zap.Float64("holdings_value", historyRecord.HoldingsValue), zap.Float64("portfolio_value", portfolio.PortfolioValue), zap.Float64("cumulative_pnl", historyRecord.CumulativePnL), zap.Float64("pnl_from_last_run", historyRecord.PnLFromLastRun), zap.Int("holdings_count", historyRecord.HoldingsCount), zap.Int("total_shares", historyRecord.TotalShares))
	}

	finishedAt := time.Now()
	logRunSummary(currTs, stats, startedAt, finishedAt)

	runRecord := db.RunRecord{
		RunTs:                  currTs.Unix(),
		StartedAt:              startedAt.Unix(),
		FinishedAt:             finishedAt.Unix(),
		Status:                 "completed",
		ProcessedExchanges:     strings.Join(stats.ProcessedExchanges, ","),
		ClosedExchanges:        strings.Join(stats.ClosedExchanges, ","),
		ExchangesTotal:         stats.ExchangesTotal,
		ExchangesClosed:        stats.ExchangesClosed,
		PortfoliosLoaded:       stats.PortfoliosLoaded,
		SymbolsProcessed:       stats.SymbolsProcessed,
		HoldingsChecked:        stats.HoldingsChecked,
		BuySignals:             stats.BuySignals,
		SellSignals:            stats.SellSignals,
		BuyOrders:              stats.BuyOrders,
		SellOrders:             stats.SellOrders,
		SharesBought:           stats.SharesBought,
		SharesSold:             stats.SharesSold,
		RequestedBuyShares:     stats.RequestedBuyShares,
		AmountSpent:            stats.AmountSpent,
		AmountEarned:           stats.AmountEarned,
		RealizedPnL:            stats.RealizedPnL,
		SkippedNoData:          stats.SkippedNoData,
		SkippedStaleData:       stats.SkippedStaleData,
		SkippedInvalidBuySize:  stats.SkippedInvalidBuySize,
		SkippedUnaffordableBuy: stats.SkippedUnaffordable,
		StrategyErrors:         stats.StrategyErrors,
	}
	if err := db.InsertRun(runRecord); err != nil {
		logging.L.Error("Failed to persist run summary", zap.Error(err))
	}
}
