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
