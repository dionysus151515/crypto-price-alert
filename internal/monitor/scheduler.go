package monitor

import (
	"context"
	"log/slog"
	"time"

	"crypto-price-alert/internal/alert"
	"crypto-price-alert/internal/binance"
	"crypto-price-alert/internal/config"
	"crypto-price-alert/internal/price"
)

type Monitor struct {
	cfg     *config.Config
	client  *binance.Client
	tracker *price.Tracker
	dedup   *alert.Deduplicator
	feishu  *alert.FeishuClient
}

func New(cfg *config.Config) *Monitor {
	return &Monitor{
		cfg:     cfg,
		client:  binance.NewClient(cfg.Binance.BaseURL, cfg.Binance.USDMBaseURL, cfg.Binance.TimeoutSeconds, cfg.Proxy.HTTP),
		tracker: price.NewTracker(),
		dedup:   alert.NewDeduplicator(),
		feishu:  alert.NewFeishuClient(cfg.Feishu.WebhookURL, cfg.Feishu.Secret),
	}
}

func (m *Monitor) Run(ctx context.Context) error {
	interval := time.Duration(m.cfg.Monitor.PollIntervalSeconds) * time.Second
	cooldown := time.Duration(m.cfg.Alert.CooldownMinutes) * time.Minute

	slog.Info("monitor started",
		"symbols", len(m.cfg.Symbols),
		"poll_interval", interval,
		"cooldown", cooldown,
	)

	// Run immediately on start, then on ticker
	m.pollAll(cooldown)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("monitor shutting down")
			return nil
		case <-ticker.C:
			m.pollAll(cooldown)
		}
	}
}

func (m *Monitor) pollAll(cooldown time.Duration) {
	for _, sym := range m.cfg.Symbols {
		m.pollSymbol(sym, cooldown)
	}
}

func (m *Monitor) pollSymbol(sym config.SymbolConfig, cooldown time.Duration) {
	interval, limit := binance.WindowToIntervalAndLimit(sym.WindowMinutes)
	identity := sym.Identity()

	klines, err := m.fetchSeries(sym, interval, limit)
	if err != nil {
		slog.Error("fetch klines failed", "symbol", sym.Symbol, "error", err)
		return
	}

	if len(klines) == 0 {
		slog.Warn("no klines returned", "symbol", sym.Symbol)
		return
	}

	// Record the latest close price in the tracker
	latest := klines[len(klines)-1]
	now := time.Now()
	m.tracker.Record(identity, latest.Close, now, sym.WindowMinutes)

	// Compute change from klines: first candle open vs last candle close
	if len(klines) >= 2 {
		first := klines[0]
		last := klines[len(klines)-1]

		if first.Open > 0 {
			changePct := (last.Close - first.Open) / first.Open * 100

			slog.Debug("price check",
				"symbol", sym.Symbol,
				"window", sym.WindowMinutes,
				"open", first.Open,
				"close", last.Close,
				"change_pct", changePct,
			)

			change := price.PriceChange{
				Symbol:       sym.Symbol,
				Market:       sym.Market,
				PriceSource:  sym.PriceSource,
				MarketLabel:  sym.MarketLabel(),
				TitleLabel:   sym.MarketTitleLabel(),
				CurrentPrice: last.Close,
				OldPrice:     first.Open,
				ChangePct:    changePct,
				WindowMin:    sym.WindowMinutes,
				Timestamp:    now,
			}

			if abs(changePct) >= sym.ThresholdPct {
				if m.dedup.ShouldAlert(identity, cooldown) {
					slog.Info("threshold triggered", "change", change.String())
					if err := m.feishu.SendAlertWithThreshold(change, sym.ThresholdPct); err != nil {
						slog.Error("send alert failed", "symbol", sym.Symbol, "error", err)
					} else {
						m.dedup.MarkAlerted(identity)
					}
				} else {
					slog.Debug("alert suppressed by cooldown", "symbol", sym.Symbol, "identity", identity)
				}
			}
		}
	}
}

func (m *Monitor) fetchSeries(sym config.SymbolConfig, interval string, limit int) ([]binance.Kline, error) {
	switch sym.Market {
	case "usdm_perp":
		return m.client.FetchUSDMMarkPriceKlines(sym.Symbol, interval, limit)
	default:
		return m.client.FetchSpotKlines(sym.Symbol, interval, limit)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
