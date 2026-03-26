package scripts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/grenmon2202/greninvestor/charts"
	"github.com/grenmon2202/greninvestor/config"
)

type StrategyDecision struct {
	Decision   bool    `json:"decision"`
	Confidence float32 `json:"confidence"`
	Shares     int     `json:"shares"`
}

func ExecuteTrendFollowingStrategy(data []charts.Candle, mode string, symbol string) (StrategyDecision, error) {
	return ExecuteScript(data, mode, symbol, config.TREND_FOLLOWING_SCRIPT_PATH, config.TREND_FOLLOWING_CONFIG_PATH, nil)
}

func ExecuteTrendFollowingStrategyWithPortfolio(data []charts.Candle, mode string, symbol string, portfolioData any) (StrategyDecision, error) {
	return ExecuteScript(data, mode, symbol, config.TREND_FOLLOWING_SCRIPT_PATH, config.TREND_FOLLOWING_CONFIG_PATH, portfolioData)
}

func parseStrategyDecision(mode string, raw string) (StrategyDecision, error) {
	var decision StrategyDecision
	if err := json.Unmarshal([]byte(raw), &decision); err != nil {
		return StrategyDecision{}, fmt.Errorf("failed to decode strategy output: %w", err)
	}

	if mode == "buy" && decision.Decision && decision.Shares <= 0 {
		return StrategyDecision{}, fmt.Errorf("buy decision must include positive shares, got %d", decision.Shares)
	}

	if !decision.Decision {
		decision.Shares = 0
	}

	return decision, nil
}

func ExecuteScript(data []charts.Candle, mode string, symbol string, script string, strategy_config string, portfolioData any) (StrategyDecision, error) {
	if mode != "buy" && mode != "sell" {
		return StrategyDecision{}, fmt.Errorf("invalid mode: %s", mode)
	}

	data_stringify, err := json.Marshal(data)
	if err != nil {
		return StrategyDecision{}, fmt.Errorf("failed to marshal candle data: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		config.PYTHON_EXECUTABLE_PATH,
		script,
		"-c", strategy_config,
		"-m", mode,
		"-i", string(data_stringify),
		"-s", symbol,
	)

	if portfolioData != nil {
		portfolioStringify, err := json.Marshal(portfolioData)
		if err != nil {
			return StrategyDecision{}, fmt.Errorf("failed to marshal portfolio data: %v", err)
		}

		cmd.Args = append(cmd.Args, "-p", string(portfolioStringify))
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return StrategyDecision{}, fmt.Errorf("command timed out")
	}

	if err != nil {
		return StrategyDecision{}, fmt.Errorf("command execution failed: %v, stderr: %s", err, stderr.String())
	}

	decision, err := parseStrategyDecision(mode, stdout.String())
	if err != nil {
		return StrategyDecision{}, fmt.Errorf("unexpected output format: %w", err)
	}

	return decision, nil
}
