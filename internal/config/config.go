package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SymbolConfig struct {
	Symbol        string  `yaml:"symbol"`
	WindowMinutes int     `yaml:"window_minutes"`
	ThresholdPct  float64 `yaml:"threshold_pct"`
}

type MonitorConfig struct {
	PollIntervalSeconds int `yaml:"poll_interval_seconds"`
}

type AlertConfig struct {
	CooldownMinutes int `yaml:"cooldown_minutes"`
}

type FeishuConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Secret     string `yaml:"secret"`
}

type ProxyConfig struct {
	HTTP string `yaml:"http"`
}

type BinanceConfig struct {
	BaseURL        string `yaml:"base_url"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type Config struct {
	Symbols []SymbolConfig `yaml:"symbols"`
	Monitor MonitorConfig  `yaml:"monitor"`
	Alert   AlertConfig    `yaml:"alert"`
	Feishu  FeishuConfig   `yaml:"feishu"`
	Binance BinanceConfig  `yaml:"binance"`
	Proxy   ProxyConfig    `yaml:"proxy"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.setDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.Monitor.PollIntervalSeconds <= 0 {
		c.Monitor.PollIntervalSeconds = 30
	}
	if c.Alert.CooldownMinutes <= 0 {
		c.Alert.CooldownMinutes = 10
	}
	if c.Binance.BaseURL == "" {
		c.Binance.BaseURL = "https://api.binance.com"
	}
	if c.Binance.TimeoutSeconds <= 0 {
		c.Binance.TimeoutSeconds = 10
	}
	for i := range c.Symbols {
		if c.Symbols[i].WindowMinutes <= 0 {
			c.Symbols[i].WindowMinutes = 5
		}
		if c.Symbols[i].ThresholdPct <= 0 {
			c.Symbols[i].ThresholdPct = 3.0
		}
	}
}

func (c *Config) validate() error {
	if len(c.Symbols) == 0 {
		return fmt.Errorf("symbols list is empty")
	}
	for _, s := range c.Symbols {
		if s.Symbol == "" {
			return fmt.Errorf("symbol name is empty")
		}
	}
	if c.Feishu.WebhookURL == "" {
		return fmt.Errorf("feishu webhook_url is required")
	}
	return nil
}
