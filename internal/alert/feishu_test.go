package alert

import (
	"strings"
	"testing"
	"time"

	"crypto-price-alert/internal/price"
)

func TestBuildTradingURLByMarket(t *testing.T) {
	t.Parallel()

	spotURL := buildTradingURL(price.PriceChange{Symbol: "BTCUSDT", Market: "spot"})
	if spotURL != "https://www.binance.com/en/trade/BTC_USDT" {
		t.Fatalf("spot URL = %q, want spot trade URL", spotURL)
	}

	perpURL := buildTradingURL(price.PriceChange{Symbol: "BTCUSDT", Market: "usdm_perp"})
	if perpURL != "https://www.binance.com/en/futures/BTCUSDT" {
		t.Fatalf("perp URL = %q, want futures URL", perpURL)
	}
}

func TestBuildCardIncludesMarketLabel(t *testing.T) {
	t.Parallel()

	client := NewFeishuClient("https://example.com/hook", "")
	change := price.PriceChange{
		Symbol:       "BTCUSDT",
		Market:       "usdm_perp",
		PriceSource:  "mark",
		MarketLabel:  "永续 (USDⓈ-M Mark Price)",
		TitleLabel:   "永续-Mark",
		CurrentPrice: 65000,
		OldPrice:     64000,
		ChangePct:    1.56,
		WindowMin:    5,
		Timestamp:    time.Unix(1710000000, 0).UTC(),
	}

	card := client.buildCard(change, 1.5)
	header := card["header"].(map[string]interface{})
	title := header["title"].(map[string]interface{})["content"].(string)
	if !strings.Contains(title, "[永续-Mark]") {
		t.Fatalf("title = %q, want market label", title)
	}

	elements := card["elements"].([]interface{})
	content := elements[0].(map[string]interface{})["text"].(map[string]interface{})["content"].(string)
	if !strings.Contains(content, "**市场** 永续 (USDⓈ-M Mark Price)") {
		t.Fatalf("content = %q, want market field", content)
	}

	actions := elements[2].(map[string]interface{})["actions"].([]interface{})
	button := actions[0].(map[string]interface{})
	if got := button["url"].(string); got != "https://www.binance.com/en/futures/BTCUSDT" {
		t.Fatalf("button URL = %q, want futures URL", got)
	}
	if got := button["text"].(map[string]interface{})["content"].(string); got != "查看 Binance Futures" {
		t.Fatalf("button text = %q, want futures button", got)
	}
}
