package scripts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/grenmon2202/greninvestor/charts"
	"github.com/grenmon2202/greninvestor/config"
)

func ExecuteTrendFollowingStrategy(data []charts.Candle, mode string, symbol string) (bool, float32, error) {
	return ExecuteScript(data, mode, symbol, config.TREND_FOLLOWING_SCRIPT_PATH, config.TREND_FOLLOWING_CONFIG_PATH)
}

func ExecuteScript(data []charts.Candle, mode string, symbol string, script string, strategy_config string) (bool, float32, error) {
	if mode != "buy" && mode != "sell" {
		return false, 0, fmt.Errorf("invalid mode: %s", mode)
	}

	data_stringify, err := json.Marshal(data)
	if err != nil {
		return false, 0, fmt.Errorf("failed to marshal candle data: %v", err)
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

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return false, 0, fmt.Errorf("command timed out")
	}

	if err != nil {
		return false, 0, fmt.Errorf("command execution failed: %v, stderr: %s", err, stderr.String())
	}

	var decision bool
	var confidence float32

	parts := strings.Fields(strings.TrimSpace(stdout.String()))
	if len(parts) != 2 {
		return false, 0, fmt.Errorf("unexpected output format: %s", stdout.String())
	}

	decision = parts[0] == "True"
	score, err := strconv.ParseFloat(parts[1], 32)
	if err != nil {
		return false, 0, fmt.Errorf("failed to parse confidence score: %v", err)
	}
	confidence = float32(score)

	return decision, confidence, nil
}
