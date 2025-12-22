package db

import (
	"database/sql"

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
