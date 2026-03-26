package config

import "testing"

func TestLoadStrategyRuntimes(t *testing.T) {
	runtimes, err := LoadStrategyRuntimes()
	if err != nil {
		t.Fatalf("LoadStrategyRuntimes returned error: %v", err)
	}

	if len(runtimes) == 0 {
		t.Fatalf("expected at least one strategy runtime")
	}

	first := runtimes[0]
	if first.Name == "" || first.ScriptPath == "" || first.ConfigPath == "" || first.PortfolioName == "" {
		t.Fatalf("expected first strategy runtime to be fully populated: %+v", first)
	}
}
