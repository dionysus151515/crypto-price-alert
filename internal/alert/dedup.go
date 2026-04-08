package alert

import (
	"sync"
	"time"
)

type Deduplicator struct {
	mu       sync.Mutex
	lastSent map[string]time.Time // symbol -> last alert time
}

func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		lastSent: make(map[string]time.Time),
	}
}

// ShouldAlert returns true if enough time has passed since the last alert for this symbol.
func (d *Deduplicator) ShouldAlert(symbol string, cooldown time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	last, ok := d.lastSent[symbol]
	if !ok {
		return true
	}
	return time.Since(last) >= cooldown
}

// MarkAlerted records the current time as the last alert time for the symbol.
func (d *Deduplicator) MarkAlerted(symbol string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastSent[symbol] = time.Now()
}
