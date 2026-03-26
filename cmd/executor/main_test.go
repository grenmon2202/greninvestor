package main

import "testing"

func TestComputeBuySharesRequestedFitsWallet(t *testing.T) {
	shares := computeBuyShares(1000, 100, 3)
	if shares != 3 {
		t.Fatalf("expected 3 shares, got %d", shares)
	}
}

func TestComputeBuySharesClampsToAffordable(t *testing.T) {
	shares := computeBuyShares(250, 100, 5)
	if shares != 2 {
		t.Fatalf("expected clamp to 2 shares, got %d", shares)
	}
}

func TestComputeBuySharesRejectsInvalidRequestedSize(t *testing.T) {
	shares := computeBuyShares(1000, 100, 0)
	if shares != 0 {
		t.Fatalf("expected 0 shares for invalid request, got %d", shares)
	}
}

func TestComputeBuySharesRejectsNegativeRequestedSize(t *testing.T) {
	shares := computeBuyShares(1000, 100, -2)
	if shares != 0 {
		t.Fatalf("expected 0 shares for negative request, got %d", shares)
	}
}

func TestComputeBuySharesRejectsUnaffordableBuy(t *testing.T) {
	shares := computeBuyShares(50, 100, 2)
	if shares != 0 {
		t.Fatalf("expected 0 shares when wallet cannot afford one share, got %d", shares)
	}
}
