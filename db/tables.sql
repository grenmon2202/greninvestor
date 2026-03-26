-- Stock master table
CREATE TABLE IF NOT EXISTS stock (
  symbol   TEXT PRIMARY KEY,                 -- e.g., 'AAPL', 'RELIANCE.NS'
  name     TEXT,                             -- optional friendly name
  exchange TEXT NOT NULL,                    -- e.g., 'NASDAQ', 'NSE'
  currency TEXT NOT NULL                     -- e.g., 'USD', 'INR'
);

-- Hourly OHLCV candles
-- ts is epoch seconds (INTEGER). Decide once: candle start OR candle close and stick with it.
CREATE TABLE IF NOT EXISTS market_data_1h (
  symbol TEXT NOT NULL,
  ts     INTEGER NOT NULL,                   -- epoch seconds
  o      REAL NOT NULL,
  h      REAL NOT NULL,
  l      REAL NOT NULL,
  c      REAL NOT NULL,
  v      INTEGER NOT NULL,

  PRIMARY KEY (symbol, ts),
  FOREIGN KEY (symbol) REFERENCES stock(symbol)
);

-- Helpful index for time-range queries per symbol
CREATE INDEX IF NOT EXISTS idx_market_data_1h_symbol_ts
  ON market_data_1h(symbol, ts);

CREATE TABLE IF NOT EXISTS portfolio (
	"name"	TEXT NOT NULL,
	"wallet"	NUMERIC NOT NULL,
	"portfolio_value" NUMERIC NOT NULL DEFAULT 0,
	PRIMARY KEY("name")
);

CREATE TABLE IF NOT EXISTS portfolio_history (
  run_ts INTEGER NOT NULL,
  portfolio TEXT NOT NULL,
  wallet REAL NOT NULL,
  holdings_value REAL NOT NULL,
  portfolio_value REAL NOT NULL,
  cumulative_pnl REAL NOT NULL,
  pnl_from_last_run REAL NOT NULL,
  holdings_count INTEGER NOT NULL,
  total_shares INTEGER NOT NULL,

  PRIMARY KEY (run_ts, portfolio),
  FOREIGN KEY (portfolio) REFERENCES portfolio(name)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_history_portfolio_run_ts
  ON portfolio_history(portfolio, run_ts);

CREATE TABLE IF NOT EXISTS holdings (
  name TEXT NOT NULL,
  symbol          TEXT NOT NULL,
  buy_ts         INTEGER NOT NULL,
  entry_point REAL NOT NULL,
  num_shares INTEGER NOT NULL,
  quant_invested_inr REAL NOT NULL,
  inr_usd_conv_ratio REAL,
  
  PRIMARY KEY (name, symbol, buy_ts),

  FOREIGN KEY (name) REFERENCES portfolio(name),
  FOREIGN KEY (symbol) REFERENCES stock(symbol)
);

CREATE INDEX IF NOT EXISTS idx_holdings_portfolio
  ON holdings(name);

CREATE INDEX IF NOT EXISTS idx_holdings_stock
  ON holdings(symbol);

CREATE TABLE IF NOT EXISTS trades (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  portfolio TEXT NOT NULL,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  ts INTEGER NOT NULL,
  price REAL NOT NULL,
  shares INTEGER NOT NULL,
  pnl REAL,

  FOREIGN KEY (portfolio) REFERENCES portfolio(name),
  FOREIGN KEY (symbol) REFERENCES stock(symbol)
);

CREATE INDEX IF NOT EXISTS idx_trades_portfolio_ts
  ON trades(portfolio, ts);

CREATE INDEX IF NOT EXISTS idx_trades_symbol_ts
  ON trades(symbol, ts);

CREATE TABLE IF NOT EXISTS runs (
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
);

CREATE INDEX IF NOT EXISTS idx_runs_run_ts
  ON runs(run_ts);

