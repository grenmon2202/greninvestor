package main

import (
	"testing"
	"time"
)

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", value, err)
	}

	return parsed
}

func TestIsMarketClosedNSEDuringSession(t *testing.T) {
	currTs := mustParseTime(t, "2026-03-26T10:00:00+05:30")

	if isMarketClosed(currTs, "NSE") {
		t.Fatalf("expected NSE to be open at %s", currTs)
	}
}

func TestIsMarketClosedNSEAfterSession(t *testing.T) {
	currTs := mustParseTime(t, "2026-03-26T20:00:00+05:30")

	if !isMarketClosed(currTs, "NSE") {
		t.Fatalf("expected NSE to be closed at %s", currTs)
	}
}

func TestIsMarketClosedNASDAQDuringSessionFromIST(t *testing.T) {
	currTs := mustParseTime(t, "2026-03-26T00:30:00+05:30")

	if isMarketClosed(currTs, "NASDAQ") {
		t.Fatalf("expected NASDAQ to be open at %s", currTs)
	}
}

func TestIsMarketClosedNASDAQAfterSessionFromIST(t *testing.T) {
	currTs := mustParseTime(t, "2026-03-26T02:00:00+05:30")

	if !isMarketClosed(currTs, "NASDAQ") {
		t.Fatalf("expected NASDAQ to be closed at %s", currTs)
	}
}

func TestIsMarketClosedWeekend(t *testing.T) {
	currTs := mustParseTime(t, "2026-03-28T10:00:00+05:30")

	if !isMarketClosed(currTs, "NSE") {
		t.Fatalf("expected NSE to be closed on weekend at %s", currTs)
	}
}
