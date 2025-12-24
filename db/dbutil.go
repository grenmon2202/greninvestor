package db

import (
	"database/sql"
	"time"

	"github.com/grenmon2202/greninvestor/charts"
	"github.com/grenmon2202/greninvestor/config"
	"github.com/grenmon2202/greninvestor/logging"
	"github.com/grenmon2202/greninvestor/symbols"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

func InsertStockIfNotExists(symbol symbols.Symbol) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("code", symbol.Code), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT OR IGNORE INTO ` + config.TBL_STOCKS + ` (name, symbol, exchange, currency) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(query, symbol.Name, symbol.Code, symbol.Exchange, symbol.Currency)
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

func CreateNewPortfolio(name string, initialWallet float32) error {
	logging.Init()
	defer logging.L.Sync()

	db, err := sql.Open("sqlite", config.DB_PATH)
	if err != nil {
		logging.L.Error("Failed to open database", zap.String("portfolio", name), zap.Error(err))
		return err
	}
	defer db.Close()

	query := `INSERT INTO ` + config.TBL_PORTFOLIOS + ` (name, wallet) VALUES (?, ?)`
	_, err = db.Exec(query, name, initialWallet)
	if err != nil {
		logging.L.Error("Failed to create portfolio", zap.String("portfolio", name), zap.Error(err))
		return err
	}

	logging.L.Info("Created new portfolio", zap.String("portfolio", name), zap.Float32("wallet", initialWallet))
	return nil
}
