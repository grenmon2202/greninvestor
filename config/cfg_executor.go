package config

import "time"

var NSE_END_HOUR = 15
var NSE_END_MINUTE = 30

var NASDAQ_END_HOUR = 2
var NASDAQ_END_MINUTE = 30

var DS_DATA_RANGE_D = 15
var MAX_DELTA_FROM_LAST_CANDLE_S = int64(1 * 60 * 60) // 1 hour

var MINUTE_PADDING_NSE = 15
var MINUTE_PADDING_NASDAQ = 30

var PYTHON_EXECUTABLE_PATH = "/opt/anaconda3/envs/dawg/bin/python"

var TREND_FOLLOWING_SCRIPT_PATH = "scripts/trend_following.py"
var TREND_FOLLOWING_CONFIG_PATH = "config/trend_following.yaml"

func GetMinutePadding(exchange string) int {
	switch exchange {
	case "NSE":
		return MINUTE_PADDING_NSE
	case "NASDAQ":
		return MINUTE_PADDING_NASDAQ
	default:
		panic("unsupported exchange: " + exchange)
	}
}

func GetClosestHour(exchange string) int64 {
	now := time.Now()
	targetMinute := GetMinutePadding(exchange)

	result := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), targetMinute, 0, 0, now.Location())

	if now.Minute() < targetMinute {
		result = result.Add(-time.Hour)
	}

	return result.Unix()
}
