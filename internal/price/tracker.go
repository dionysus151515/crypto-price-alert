package price

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type PricePoint struct {
	Price     float64
	Timestamp time.Time
}

type PriceChange struct {
	Symbol       string
	Market       string
	PriceSource  string
	MarketLabel  string
	TitleLabel   string
	CurrentPrice float64
	OldPrice     float64
	ChangePct    float64
	WindowMin    int
	Timestamp    time.Time
}

func (pc PriceChange) Direction() string {
	if pc.ChangePct >= 0 {
		return "+"
	}
	return ""
}

func (pc PriceChange) String() string {
	return fmt.Sprintf("%s [%s] %s%.2f%% (%dmin) $%.4f -> $%.4f",
		pc.Symbol, pc.TitleLabel, pc.Direction(), pc.ChangePct, pc.WindowMin, pc.OldPrice, pc.CurrentPrice)
}

type Tracker struct {
	mu      sync.RWMutex
	windows map[string][]PricePoint // symbol -> sliding window
}

func NewTracker() *Tracker {
	return &Tracker{
		windows: make(map[string][]PricePoint),
	}
}

// Record adds a price point and evicts entries older than the window.
func (t *Tracker) Record(symbol string, price float64, ts time.Time, windowMinutes int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := ts.Add(-time.Duration(windowMinutes) * time.Minute)
	points := t.windows[symbol]

	// Evict old entries
	start := 0
	for start < len(points) && points[start].Timestamp.Before(cutoff) {
		start++
	}
	// Keep one point at or just before cutoff for accurate comparison
	if start > 0 {
		start--
	}

	points = append(points[start:], PricePoint{Price: price, Timestamp: ts})
	t.windows[symbol] = points
}

// CheckChange computes the price change from the oldest point in the window to the latest.
// Returns the change and whether it exceeds the threshold.
func (t *Tracker) CheckChange(symbol string, thresholdPct float64, windowMinutes int) (PriceChange, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	points := t.windows[symbol]
	if len(points) < 2 {
		return PriceChange{}, false
	}

	oldest := points[0]
	latest := points[len(points)-1]

	if oldest.Price == 0 {
		return PriceChange{}, false
	}

	changePct := (latest.Price - oldest.Price) / oldest.Price * 100

	change := PriceChange{
		Symbol:       symbol,
		CurrentPrice: latest.Price,
		OldPrice:     oldest.Price,
		ChangePct:    changePct,
		WindowMin:    windowMinutes,
		Timestamp:    latest.Timestamp,
	}

	return change, math.Abs(changePct) >= thresholdPct
}
