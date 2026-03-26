package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var NSE_END_HOUR = 15
var NSE_END_MINUTE = 30
var NSE_START_HOUR = 9
var NSE_START_MINUTE = 15

var NASDAQ_START_HOUR = 9
var NASDAQ_START_MINUTE = 30
var NASDAQ_END_HOUR = 16
var NASDAQ_END_MINUTE = 0

var NSE_TIMEZONE = "Asia/Kolkata"
var NASDAQ_TIMEZONE = "America/New_York"

var DS_DATA_RANGE_D = 15
var MAX_DELTA_FROM_LAST_CANDLE_S = int64(1 * 60 * 60) // 1 hour

var MINUTE_PADDING_NSE = 15
var MINUTE_PADDING_NASDAQ = 30

var PYTHON_EXECUTABLE_PATH = getEnvOrDefault("PYTHON_PATH", "python")

var STRATEGY_RUNTIMES_PATH = "config/strategy_runtimes.json"
var TREND_FOLLOWING_SCRIPT_PATH = "scripts/trend_following.py"
var TREND_FOLLOWING_CONFIG_PATH = "config/trend_following.yaml"

type StrategyRuntime struct {
	Name          string  `json:"name"`
	ScriptPath    string  `json:"script_path"`
	ConfigPath    string  `json:"config_path"`
	PortfolioName string  `json:"portfolio_name"`
	InitialWallet float64 `json:"initial_wallet"`
	Enabled       bool    `json:"enabled"`
}

func LoadStrategyRuntimes() ([]StrategyRuntime, error) {
	candidatePaths := []string{
		STRATEGY_RUNTIMES_PATH,
		filepath.Join("..", STRATEGY_RUNTIMES_PATH),
	}

	var data []byte
	var err error
	for _, path := range candidatePaths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read strategy runtimes config: %w", err)
	}

	var runtimes []StrategyRuntime
	if err := json.Unmarshal(data, &runtimes); err != nil {
		return nil, err
	}

	return runtimes, nil
}

func GetEnabledStrategies() []StrategyRuntime {
	runtimes, err := LoadStrategyRuntimes()
	if err != nil {
		panic(err)
	}

	enabled := make([]StrategyRuntime, 0)
	for _, strategy := range runtimes {
		if strategy.Enabled {
			enabled = append(enabled, strategy)
		}
	}
	return enabled
}

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

func GetExchangeTimezone(exchange string) string {
	switch exchange {
	case "NSE":
		return NSE_TIMEZONE
	case "NASDAQ":
		return NASDAQ_TIMEZONE
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
