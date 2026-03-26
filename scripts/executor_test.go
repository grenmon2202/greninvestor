package scripts

import "testing"

func TestParseStrategyDecisionBuy(t *testing.T) {
	result, err := parseStrategyDecision("buy", `{"decision": true, "confidence": 1.25, "shares": 4}`)
	if err != nil {
		t.Fatalf("parseStrategyDecision returned error: %v", err)
	}

	if !result.Decision {
		t.Fatalf("expected decision=true")
	}

	if result.Confidence != float32(1.25) {
		t.Fatalf("expected confidence=1.25, got %v", result.Confidence)
	}

	if result.Shares != 4 {
		t.Fatalf("expected shares=4, got %d", result.Shares)
	}
}

func TestParseStrategyDecisionSell(t *testing.T) {
	result, err := parseStrategyDecision("sell", `{"decision": true, "confidence": -1, "shares": 0}`)
	if err != nil {
		t.Fatalf("parseStrategyDecision returned error: %v", err)
	}

	if !result.Decision {
		t.Fatalf("expected decision=true")
	}

	if result.Shares != 0 {
		t.Fatalf("expected shares=0 for sell, got %d", result.Shares)
	}
}

func TestParseStrategyDecisionMalformedJSON(t *testing.T) {
	if _, err := parseStrategyDecision("buy", `not-json`); err == nil {
		t.Fatalf("expected malformed JSON to fail")
	}
}

func TestParseStrategyDecisionMissingSharesOnBuy(t *testing.T) {
	if _, err := parseStrategyDecision("buy", `{"decision": true, "confidence": 0.5}`); err == nil {
		t.Fatalf("expected missing shares to fail for buy decision")
	}
}

func TestParseStrategyDecisionNoBuyZeroesShares(t *testing.T) {
	result, err := parseStrategyDecision("buy", `{"decision": false, "confidence": 0.2, "shares": 7}`)
	if err != nil {
		t.Fatalf("parseStrategyDecision returned error: %v", err)
	}

	if result.Shares != 0 {
		t.Fatalf("expected shares=0 for no-buy, got %d", result.Shares)
	}
}
