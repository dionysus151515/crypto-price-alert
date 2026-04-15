package alert

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"crypto-price-alert/internal/price"
)

type FeishuClient struct {
	webhookURL string
	secret     string
	httpClient *http.Client
}

func NewFeishuClient(webhookURL, secret string) *FeishuClient {
	return &FeishuClient{
		webhookURL: webhookURL,
		secret:     secret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendAlertWithThreshold sends an alert card with threshold info to Feishu.
func (f *FeishuClient) SendAlertWithThreshold(change price.PriceChange, thresholdPct float64) error {
	card := f.buildCard(change, thresholdPct)

	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}

	if f.secret != "" {
		ts := time.Now().Unix()
		sign, err := genSign(f.secret, ts)
		if err != nil {
			return fmt.Errorf("generate sign: %w", err)
		}
		payload["timestamp"] = fmt.Sprintf("%d", ts)
		payload["sign"] = sign
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := f.httpClient.Post(f.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send feishu webhook: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.Code != 0 {
		return fmt.Errorf("feishu error code %d: %s", result.Code, result.Msg)
	}

	slog.Info("feishu alert sent", "symbol", change.Symbol, "change", fmt.Sprintf("%.2f%%", change.ChangePct))
	return nil
}

func (f *FeishuClient) buildCard(c price.PriceChange, thresholdPct float64) map[string]interface{} {
	template := "turquoise"
	if c.ChangePct < 0 {
		template = "red"
	}

	title := fmt.Sprintf("Price Alert: %s [%s] %s%.2f%% (%dmin)",
		c.Symbol, c.TitleLabel, c.Direction(), c.ChangePct, c.WindowMin)

	buttonText := "查看 Binance"
	if c.Market == "usdm_perp" {
		buttonText = "查看 Binance Futures"
	}

	content := fmt.Sprintf(
		"**交易对** %s\n**市场** %s\n**涨跌幅** %s%.2f%%（%d 分钟）\n**当前价格** $%s\n**窗口起始价格** $%s\n**时间** %s\n**阈值** ≥ %.1f%%",
		c.Symbol,
		c.MarketLabel,
		c.Direction(), c.ChangePct, c.WindowMin,
		formatPrice(c.CurrentPrice),
		formatPrice(c.OldPrice),
		c.Timestamp.Format("2006-01-02 15:04:05 MST"),
		thresholdPct,
	)

	return map[string]interface{}{
		"header": map[string]interface{}{
			"template": template,
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": content,
				},
			},
			map[string]interface{}{"tag": "hr"},
			map[string]interface{}{
				"tag": "action",
				"actions": []interface{}{
					map[string]interface{}{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": buttonText,
						},
						"type": "primary",
						"url":  buildTradingURL(c),
					},
				},
			},
		},
	}
}

func buildTradingPairURL(symbol string) string {
	// BTCUSDT -> BTC_USDT
	suffixes := []string{"USDT", "BUSD", "BTC", "ETH", "BNB"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(symbol, suffix) {
			base := strings.TrimSuffix(symbol, suffix)
			return base + "_" + suffix
		}
	}
	return symbol
}

func buildTradingURL(c price.PriceChange) string {
	if c.Market == "usdm_perp" {
		return fmt.Sprintf("https://www.binance.com/en/futures/%s", c.Symbol)
	}

	tradingPair := buildTradingPairURL(c.Symbol)
	return fmt.Sprintf("https://www.binance.com/en/trade/%s", tradingPair)
}

func formatPrice(p float64) string {
	if p >= 1000 {
		return fmt.Sprintf("%.2f", p)
	}
	if p >= 1 {
		return fmt.Sprintf("%.4f", p)
	}
	return fmt.Sprintf("%.6f", p)
}

func genSign(secret string, timestamp int64) (string, error) {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte{})
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
