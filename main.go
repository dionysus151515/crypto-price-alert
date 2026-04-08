package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"crypto-price-alert/internal/config"
	"crypto-price-alert/internal/monitor"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	// Setup logging
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("config loaded",
		"symbols", len(cfg.Symbols),
		"poll_interval", cfg.Monitor.PollIntervalSeconds,
	)
	for _, s := range cfg.Symbols {
		slog.Info("monitoring",
			"symbol", s.Symbol,
			"window", s.WindowMinutes,
			"threshold", s.ThresholdPct,
		)
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Run monitor
	m := monitor.New(cfg)
	if err := m.Run(ctx); err != nil {
		slog.Error("monitor error", "error", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
}
