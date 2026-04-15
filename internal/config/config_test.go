package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsAndIdentity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data := `symbols:
  - symbol: BTCUSDT
    window_minutes: 15
    threshold_pct: 2.0
  - symbol: BTCUSDT
    market: usdm_perp
    window_minutes: 5
    threshold_pct: 1.5
feishu:
  webhook_url: "https://example.com/hook"
`

	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got := cfg.Symbols[0].Market; got != "spot" {
		t.Fatalf("spot market default = %q, want spot", got)
	}
	if got := cfg.Symbols[0].PriceSource; got != "last" {
		t.Fatalf("spot price source default = %q, want last", got)
	}
	if got := cfg.Symbols[1].PriceSource; got != "mark" {
		t.Fatalf("usdm_perp price source default = %q, want mark", got)
	}
	if got := cfg.Binance.USDMBaseURL; got != "https://fapi.binance.com" {
		t.Fatalf("usdm base url default = %q, want https://fapi.binance.com", got)
	}

	spotID := cfg.Symbols[0].Identity()
	perpID := cfg.Symbols[1].Identity()
	if spotID == perpID {
		t.Fatalf("identities should differ, both = %q", spotID)
	}

	if got := cfg.Symbols[0].MarketLabel(); got != "现货" {
		t.Fatalf("spot market label = %q, want 现货", got)
	}
	if got := cfg.Symbols[1].MarketTitleLabel(); got != "永续-Mark" {
		t.Fatalf("usdm title label = %q, want 永续-Mark", got)
	}
}

func TestLoadRejectsUnsupportedMarketCombination(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data := `symbols:
  - symbol: BTCUSDT
    market: usdm_perp
    price_source: last
    window_minutes: 5
    threshold_pct: 1.5
feishu:
  webhook_url: "https://example.com/hook"
`

	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "price_source=mark") {
		t.Fatalf("error = %q, want mention of price_source=mark", err)
	}
}
