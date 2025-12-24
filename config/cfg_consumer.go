package config

import "net/url"

var yahooBaseURL string = "https://query1.finance.yahoo.com/v8/finance/chart/"

func GetYahooURL(symbol string) string {
	return yahooBaseURL + url.PathEscape(symbol)
}

var CANDLE_RANGE = "1d"
var CANDLE_INTERVAL = "1h"

var CANDLE_SCRAPER_SLEEP_S = 0.2
