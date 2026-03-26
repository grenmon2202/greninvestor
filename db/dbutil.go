package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/grenmon2202/greninvestor/charts"
	"github.com/grenmon2202/greninvestor/config"
	"github.com/grenmon2202/greninvestor/ledger"
	"github.com/grenmon2202/greninvestor/logging"
	"github.com/grenmon2202/greninvestor/symbols"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

type RunRecord struct {
	RunTs                  int64
	StartedAt              int64
	FinishedAt             int64
	Status                 string
	ProcessedExchanges     string
	ClosedExchanges        string
	ExchangesTotal         int
	ExchangesClosed        int
	PortfoliosLoaded       int
	SymbolsProcessed       int
	HoldingsChecked        int
	BuySignals             int
	SellSignals            int
	BuyOrders              int
	SellOrders             int
	SharesBought           int
	SharesSold             int
	RequestedBuyShares     int
	AmountSpent            float64
	AmountEarned           float64
	RealizedPnL            float64
	SkippedNoData          int
	SkippedStaleData       int
	SkippedInvalidBuySize  int
	SkippedUnaffordableBuy int
	StrategyErrors         int
}

type PortfolioHistoryRecord struct {
	RunTs           int64
	Portfolio       string
	Wallet          float64
	HoldingsValue   float64
	PortfolioValue  float64
	CumulativePnL   float64
	PnLFromLastRun  float64
	HoldingsCount   int
	TotalShares     int
}

func EnsureRunsTable() error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.Error(err))
		return err
	}
	defer db.Close()

	query := `CREATE TABLE IF NOT EXISTS ` + config.TBL_RUNS + ` (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_ts INTEGER NOT NULL,
		started_at INTEGER NOT NULL,
		finished_at INTEGER NOT NULL,
		status TEXT NOT NULL,
		processed_exchanges TEXT NOT NULL,
		closed_exchanges TEXT NOT NULL,
		exchanges_total INTEGER NOT NULL,
		exchanges_closed INTEGER NOT NULL,
		portfolios_loaded INTEGER NOT NULL,
		symbols_processed INTEGER NOT NULL,
		holdings_checked INTEGER NOT NULL,
		buy_signals INTEGER NOT NULL,
		sell_signals INTEGER NOT NULL,
		buy_orders INTEGER NOT NULL,
		sell_orders INTEGER NOT NULL,
		shares_bought INTEGER NOT NULL,
		shares_sold INTEGER NOT NULL,
		requested_buy_shares INTEGER NOT NULL,
		amount_spent REAL NOT NULL,
		amount_earned REAL NOT NULL,
		realized_pnl REAL NOT NULL,
		skipped_no_data INTEGER NOT NULL,
		skipped_stale_data INTEGER NOT NULL,
		skipped_invalid_buy_size INTEGER NOT NULL,
		skipped_unaffordable_buy INTEGER NOT NULL,
		strategy_errors INTEGER NOT NULL
	)`
	if _, err := db.Exec(query); err != nil {
		logging.L.Error("Failed to ensure runs table", zap.Error(err))
		return err
	}

	alterStatements := []string{
		`ALTER TABLE ` + config.TBL_RUNS + ` ADD COLUMN processed_exchanges TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE ` + config.TBL_RUNS + ` ADD COLUMN closed_exchanges TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil && !isDuplicateColumnError(err) {
			logging.L.Error("Failed to backfill runs table column", zap.String("statement", stmt), zap.Error(err))
			return err
		}
	}

	indexQuery := `CREATE INDEX IF NOT EXISTS idx_runs_run_ts ON ` + config.TBL_RUNS + `(run_ts)`
	if _, err := db.Exec(indexQuery); err != nil {
		logging.L.Error("Failed to ensure runs table index", zap.Error(err))
		return err
	}

	return nil
}

func EnsurePortfolioTable() error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.Error(err))
		return err
	}
	defer db.Close()

	createQuery := `CREATE TABLE IF NOT EXISTS ` + config.TBL_PORTFOLIOS + ` (
		name TEXT NOT NULL,
		wallet NUMERIC NOT NULL,
		portfolio_value NUMERIC NOT NULL DEFAULT 0,
		PRIMARY KEY(name)
	)`
	if _, err := db.Exec(createQuery); err != nil {
		logging.L.Error("Failed to ensure portfolio table", zap.Error(err))
		return err
	}

	alterQuery := `ALTER TABLE ` + config.TBL_PORTFOLIOS + ` ADD COLUMN portfolio_value NUMERIC NOT NULL DEFAULT 0`
	if _, err := db.Exec(alterQuery); err != nil && !strings.Contains(err.Error(), "duplicate column name: portfolio_value") {
		logging.L.Error("Failed to backfill portfolio_value column", zap.Error(err))
		return err
	}

	return nil
}

func EnsurePortfolioHistoryTable() error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.Error(err))
		return err
	}
	defer db.Close()

	query := `CREATE TABLE IF NOT EXISTS ` + config.TBL_PORTFOLIO_HISTORY + ` (
		run_ts INTEGER NOT NULL,
		portfolio TEXT NOT NULL,
		wallet REAL NOT NULL,
		holdings_value REAL NOT NULL,
		portfolio_value REAL NOT NULL,
		cumulative_pnl REAL NOT NULL DEFAULT 0,
		pnl_from_last_run REAL NOT NULL DEFAULT 0,
		holdings_count INTEGER NOT NULL,
		total_shares INTEGER NOT NULL,
		PRIMARY KEY (run_ts, portfolio),
		FOREIGN KEY (portfolio) REFERENCES ` + config.TBL_PORTFOLIOS + `(name)
	)`
	if _, err := db.Exec(query); err != nil {
		logging.L.Error("Failed to ensure portfolio_history table", zap.Error(err))
		return err
	}

	alterStatements := []string{
		`ALTER TABLE ` + config.TBL_PORTFOLIO_HISTORY + ` ADD COLUMN cumulative_pnl REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE ` + config.TBL_PORTFOLIO_HISTORY + ` ADD COLUMN pnl_from_last_run REAL NOT NULL DEFAULT 0`,
	}
	for _, stmt := range alterStatements {
		if _, err := db.Exec(stmt); err != nil && !isDuplicateColumnError(err) {
			logging.L.Error("Failed to backfill portfolio_history column", zap.String("statement", stmt), zap.Error(err))
			return err
		}
	}

	indexQuery := `CREATE INDEX IF NOT EXISTS idx_portfolio_history_portfolio_run_ts ON ` + config.TBL_PORTFOLIO_HISTORY + `(portfolio, run_ts)`
	if _, err := db.Exec(indexQuery); err != nil {
		logging.L.Error("Failed to ensure portfolio_history index", zap.Error(err))
		return err
	}

	return nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate column name: processed_exchanges") ||
		strings.Contains(err.Error(), "duplicate column name: closed_exchanges") ||
		strings.Contains(err.Error(), "duplicate column name: cumulative_pnl") ||
		strings.Contains(err.Error(), "duplicate column name: pnl_from_last_run")
}

func FetchPreviousPortfolioValue(portfolio string, beforeRunTs int64) (float64, error) {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", portfolio), zap.Error(err))
		return 0, err
	}
	defer db.Close()

	query := `SELECT portfolio_value FROM ` + config.TBL_PORTFOLIO_HISTORY + ` WHERE portfolio = ? AND run_ts < ? ORDER BY run_ts DESC LIMIT 1`
	row := db.QueryRow(query, portfolio, beforeRunTs)

	var portfolioValue float64
	if err := row.Scan(&portfolioValue); err != nil {
		return 0, err
	}

	return portfolioValue, nil
}

func InsertStockIfNotExists(symbol symbols.Symbol) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("code", symbol.Code), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT OR IGNORE INTO ` + config.TBL_STOCKS + ` (symbol, name, exchange, currency) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(query, symbol.Code, symbol.Name, symbol.Exchange, symbol.Currency)
	if err != nil {
		logging.L.Error("Failed to insert stock", zap.String("code", symbol.Code), zap.Error(err))
		return err
	}

	logging.L.Info("Inserted stock if not exists", zap.String("code", symbol.Code))
	return nil
}

func InsertMarketData(symbol symbols.Symbol, candles []charts.Candle) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("code", symbol.Code), zap.Error(err))
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		logging.L.Error("Failed to begin transaction", zap.String("symbol", symbol.Code), zap.Error(err))
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO ` + config.TBL_MKT_DATA + ` (symbol, ts, o, h, l, c, v) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		logging.L.Error("Failed to prepare statement", zap.String("symbol", symbol.Code), zap.Error(err))
		return err
	}
	defer stmt.Close()

	for _, candle := range candles {
		_, err := stmt.Exec(symbol.Code, candle.T.Unix(), candle.O, candle.H, candle.L, candle.C, candle.V)
		if err != nil {
			logging.L.Error("Failed to execute statement", zap.String("symbol", symbol.Code), zap.Error(err))
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logging.L.Error("Failed to commit transaction", zap.String("symbol", symbol.Code), zap.Error(err))
		return err
	}

	logging.L.Info("Inserted market data", zap.String("symbol", symbol.Code), zap.Int("count", len(candles)))
	return nil
}

func FetchMarketData(symbolCode string, fromTs time.Time, toTs time.Time) ([]charts.Candle, error) {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("code", symbolCode), zap.Error(err))
		return nil, err
	}
	defer db.Close()

	query := `SELECT ts, o, h, l, c, v FROM ` + config.TBL_MKT_DATA + ` WHERE symbol = ? AND ts BETWEEN ? AND ? ORDER BY ts ASC`
	rows, err := db.Query(query, symbolCode, fromTs.Unix(), toTs.Unix())
	if err != nil {
		logging.L.Error("Failed to query market data", zap.String("symbol", symbolCode), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var candles []charts.Candle
	for rows.Next() {
		var ts int64
		var o, h, l, c, v float64

		if err := rows.Scan(&ts, &o, &h, &l, &c, &v); err != nil {
			logging.L.Error("Failed to scan row", zap.String("symbol", symbolCode), zap.Error(err))
			return nil, err
		}

		candle := charts.Candle{
			T: time.Unix(ts, 0),
			O: o,
			H: h,
			L: l,
			C: c,
			V: v,
		}
		candles = append(candles, candle)
	}

	if err := rows.Err(); err != nil {
		logging.L.Error("Row iteration error", zap.String("symbol", symbolCode), zap.Error(err))
		return nil, err
	}

	logging.L.Info("Fetched market data", zap.String("symbol", symbolCode), zap.Int("count", len(candles)))
	return candles, nil
}

func CreateNewPortfolio(name string, initialWallet float64) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT INTO ` + config.TBL_PORTFOLIOS + ` (name, wallet, portfolio_value) VALUES (?, ?, ?)`
	_, err = db.Exec(query, name, initialWallet, initialWallet)
	if err != nil {
		logging.L.Error("Failed to create portfolio", zap.String("portfolio", name), zap.Error(err))
		return err
	}

	logging.L.Info("Created new portfolio", zap.String("portfolio", name), zap.Float64("wallet", initialWallet))
	return nil
}

func FetchPortfolio(name string) (ledger.Portfolio, error) {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return ledger.Portfolio{}, err
	}
	defer db.Close()

	query := `SELECT name, wallet, portfolio_value FROM ` + config.TBL_PORTFOLIOS + ` WHERE name = ?`
	row := db.QueryRow(query, name)

	var portfolio ledger.Portfolio
	err = row.Scan(&portfolio.Name, &portfolio.Wallet, &portfolio.PortfolioValue)
	if err != nil {
		if err != sql.ErrNoRows {
			logging.L.Error("Failed to fetch portfolio", zap.String("portfolio", name), zap.Error(err))
		}
		return ledger.Portfolio{}, err
	}

	return portfolio, nil
}

func UpdatePortfolioWallet(name string, newBalance float64) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `UPDATE ` + config.TBL_PORTFOLIOS + ` SET wallet = ? WHERE name = ?`
	res, err := db.Exec(query, newBalance, name)
	if err != nil {
		logging.L.Error("Failed to update portfolio wallet", zap.String("portfolio", name), zap.Error(err))
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func UpdatePortfolioValue(name string, newValue float64) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `UPDATE ` + config.TBL_PORTFOLIOS + ` SET portfolio_value = ? WHERE name = ?`
	res, err := db.Exec(query, newValue, name)
	if err != nil {
		logging.L.Error("Failed to update portfolio value", zap.String("portfolio", name), zap.Error(err))
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func FetchLatestMarketPrice(symbolCode string, asOf time.Time) (float64, error) {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("code", symbolCode), zap.Error(err))
		return 0, err
	}
	defer db.Close()

	query := `SELECT c FROM ` + config.TBL_MKT_DATA + ` WHERE symbol = ? AND ts <= ? ORDER BY ts DESC LIMIT 1`
	row := db.QueryRow(query, symbolCode, asOf.Unix())

	var price float64
	if err := row.Scan(&price); err != nil {
		return 0, err
	}

	return price, nil
}

func InsertHolding(h ledger.Holdings) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", h.Name), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT INTO ` + config.TBL_HOLDINGS + ` (name, symbol, buy_ts, entry_point, num_shares, quant_invested_inr, inr_usd_conv_ratio) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, h.Name, h.Symbol, h.BuyTs, h.EntryPoint, h.NumShares, h.QuantInvestedINR, h.InrUsdConvRatio)
	if err != nil {
		logging.L.Error("Failed to insert holding", zap.String("portfolio", h.Name), zap.String("symbol", h.Symbol), zap.Error(err))
		return err
	}

	return nil
}

func FetchHoldings(name string) ([]ledger.Holdings, error) {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return nil, err
	}
	defer db.Close()

	query := `SELECT name, symbol, buy_ts, entry_point, num_shares, quant_invested_inr, inr_usd_conv_ratio FROM ` + config.TBL_HOLDINGS + ` WHERE name = ? ORDER BY buy_ts ASC`
	rows, err := db.Query(query, name)
	if err != nil {
		logging.L.Error("Failed to fetch holdings", zap.String("portfolio", name), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	holdings := make([]ledger.Holdings, 0)
	for rows.Next() {
		var h ledger.Holdings
		if err := rows.Scan(&h.Name, &h.Symbol, &h.BuyTs, &h.EntryPoint, &h.NumShares, &h.QuantInvestedINR, &h.InrUsdConvRatio); err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return holdings, nil
}

func FetchHoldingsBySymbol(name string, symbol string) ([]ledger.Holdings, error) {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return nil, err
	}
	defer db.Close()

	query := `SELECT name, symbol, buy_ts, entry_point, num_shares, quant_invested_inr, inr_usd_conv_ratio FROM ` + config.TBL_HOLDINGS + ` WHERE name = ? AND symbol = ? ORDER BY buy_ts ASC`
	rows, err := db.Query(query, name, symbol)
	if err != nil {
		logging.L.Error("Failed to fetch holdings by symbol", zap.String("portfolio", name), zap.String("symbol", symbol), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	holdings := make([]ledger.Holdings, 0)
	for rows.Next() {
		var h ledger.Holdings
		if err := rows.Scan(&h.Name, &h.Symbol, &h.BuyTs, &h.EntryPoint, &h.NumShares, &h.QuantInvestedINR, &h.InrUsdConvRatio); err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return holdings, nil
}

func DeleteHolding(name string, symbol string, buyTs int64) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `DELETE FROM ` + config.TBL_HOLDINGS + ` WHERE name = ? AND symbol = ? AND buy_ts = ?`
	res, err := db.Exec(query, name, symbol, buyTs)
	if err != nil {
		logging.L.Error("Failed to delete holding", zap.String("portfolio", name), zap.String("symbol", symbol), zap.Error(err))
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func InsertTrade(portfolio string, symbol string, side string, ts int64, price float64, shares int, pnl *float64) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", portfolio), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT INTO ` + config.TBL_TRADES + ` (portfolio, symbol, side, ts, price, shares, pnl) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, portfolio, symbol, side, ts, price, shares, pnl)
	if err != nil {
		logging.L.Error("Failed to insert trade", zap.String("portfolio", portfolio), zap.String("symbol", symbol), zap.String("side", side), zap.Error(err))
		return err
	}

	return nil
}

func InsertRun(record RunRecord) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT INTO ` + config.TBL_RUNS + ` (
		run_ts, started_at, finished_at, status, processed_exchanges, closed_exchanges, exchanges_total, exchanges_closed,
		portfolios_loaded, symbols_processed, holdings_checked, buy_signals, sell_signals,
		buy_orders, sell_orders, shares_bought, shares_sold, requested_buy_shares,
		amount_spent, amount_earned, realized_pnl, skipped_no_data, skipped_stale_data,
		skipped_invalid_buy_size, skipped_unaffordable_buy, strategy_errors
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(
		query,
		record.RunTs,
		record.StartedAt,
		record.FinishedAt,
		record.Status,
		record.ProcessedExchanges,
		record.ClosedExchanges,
		record.ExchangesTotal,
		record.ExchangesClosed,
		record.PortfoliosLoaded,
		record.SymbolsProcessed,
		record.HoldingsChecked,
		record.BuySignals,
		record.SellSignals,
		record.BuyOrders,
		record.SellOrders,
		record.SharesBought,
		record.SharesSold,
		record.RequestedBuyShares,
		record.AmountSpent,
		record.AmountEarned,
		record.RealizedPnL,
		record.SkippedNoData,
		record.SkippedStaleData,
		record.SkippedInvalidBuySize,
		record.SkippedUnaffordableBuy,
		record.StrategyErrors,
	)
	if err != nil {
		logging.L.Error("Failed to insert run", zap.Error(err))
		return err
	}

	return nil
}

func InsertPortfolioHistory(record PortfolioHistoryRecord) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT OR REPLACE INTO ` + config.TBL_PORTFOLIO_HISTORY + ` (
		run_ts, portfolio, wallet, holdings_value, portfolio_value, cumulative_pnl, pnl_from_last_run, holdings_count, total_shares
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(
		query,
		record.RunTs,
		record.Portfolio,
		record.Wallet,
		record.HoldingsValue,
		record.PortfolioValue,
		record.CumulativePnL,
		record.PnLFromLastRun,
		record.HoldingsCount,
		record.TotalShares,
	)
	if err != nil {
		logging.L.Error("Failed to insert portfolio history", zap.String("portfolio", record.Portfolio), zap.Error(err))
		return err
	}

	return nil
}
