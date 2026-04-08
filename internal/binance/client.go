package binance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Kline struct {
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

func NewClient(baseURL string, timeoutSeconds int, proxyURL string) *Client {
	transport := &http.Transport{}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   time.Duration(timeoutSeconds) * time.Second,
			Transport: transport,
		},
	}
}

// WindowToIntervalAndLimit maps a window in minutes to a Binance kline interval and limit.
// It picks the largest standard interval that evenly divides the window,
// or falls back to 1m with enough candles to cover the window.
func WindowToIntervalAndLimit(windowMinutes int) (interval string, limit int) {
	// Standard Binance kline intervals in minutes (descending)
	type ivl struct {
		minutes int
		label   string
	}
	intervals := []ivl{
		{60, "1h"},
		{30, "30m"},
		{15, "15m"},
		{5, "5m"},
		{3, "3m"},
		{1, "1m"},
	}

	for _, iv := range intervals {
		if windowMinutes >= iv.minutes && windowMinutes%iv.minutes == 0 {
			// +1 to get one extra candle so we can compare open of first vs close of last
			return iv.label, windowMinutes/iv.minutes + 1
		}
	}
	// Fallback: use 1m candles
	return "1m", windowMinutes + 1
}

// FetchKlines retrieves kline/candlestick data from Binance.
func (c *Client) FetchKlines(symbol, interval string, limit int) ([]Kline, error) {
	url := fmt.Sprintf("%s/api/v3/klines?symbol=%s&interval=%s&limit=%d",
		c.baseURL, symbol, interval, limit)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request klines: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance API error %d: %s", resp.StatusCode, string(body))
	}

	var raw [][]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode klines: %w", err)
	}

	klines := make([]Kline, 0, len(raw))
	for _, r := range raw {
		if len(r) < 11 {
			continue
		}
		k, err := parseKline(r)
		if err != nil {
			continue
		}
		klines = append(klines, k)
	}
	return klines, nil
}

func parseKline(r []json.RawMessage) (Kline, error) {
	var (
		openTimeMs  int64
		closeTimeMs int64
		openStr     string
		highStr     string
		lowStr      string
		closeStr    string
		volumeStr   string
	)

	if err := json.Unmarshal(r[0], &openTimeMs); err != nil {
		return Kline{}, err
	}
	if err := json.Unmarshal(r[1], &openStr); err != nil {
		return Kline{}, err
	}
	if err := json.Unmarshal(r[2], &highStr); err != nil {
		return Kline{}, err
	}
	if err := json.Unmarshal(r[3], &lowStr); err != nil {
		return Kline{}, err
	}
	if err := json.Unmarshal(r[4], &closeStr); err != nil {
		return Kline{}, err
	}
	if err := json.Unmarshal(r[5], &volumeStr); err != nil {
		return Kline{}, err
	}
	if err := json.Unmarshal(r[6], &closeTimeMs); err != nil {
		return Kline{}, err
	}

	open, _ := strconv.ParseFloat(openStr, 64)
	high, _ := strconv.ParseFloat(highStr, 64)
	low, _ := strconv.ParseFloat(lowStr, 64)
	cls, _ := strconv.ParseFloat(closeStr, 64)
	vol, _ := strconv.ParseFloat(volumeStr, 64)

	return Kline{
		OpenTime:  time.UnixMilli(openTimeMs),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     cls,
		Volume:    vol,
		CloseTime: time.UnixMilli(closeTimeMs),
	}, nil
}
