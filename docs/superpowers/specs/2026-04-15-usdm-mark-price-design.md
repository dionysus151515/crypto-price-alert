# USDⓈ-M Perpetual Mark Price Monitoring Design

## Summary

This project currently monitors Binance spot symbols by polling spot kline data and sending Feishu alerts when the percentage change over a configured window exceeds a threshold.

The next change adds support for Binance USDⓈ-M perpetual contracts using mark price while keeping existing spot monitoring intact. The same deployment should be able to monitor spot and perpetual instruments side by side, and alert messages must clearly indicate whether the alert came from spot or perpetual mark price.

## Current State

- The monitor loop is centralized in `main.go` and `internal/monitor/scheduler.go`.
- Symbol configuration currently includes only `symbol`, `window_minutes`, and `threshold_pct`.
- Binance market data access is implemented as a single spot-only client in `internal/binance/client.go` using spot `klines`.
- Feishu alerts do not include market type and always link to a spot trading page.
- Deduplication and price tracking keys use only the symbol, which prevents safe coexistence of spot and perpetual monitors for the same symbol.

## Goals

- Support both Binance spot and Binance USDⓈ-M perpetual monitoring in one config.
- For perpetual contracts, monitor mark price instead of last traded price.
- Reuse the existing polling-based alert flow.
- Add enough structure so more market types or price sources can be added later without rewriting the scheduler.
- Keep old configs working by defaulting to the current spot behavior.

## Non-Goals

- No WebSocket implementation in this change.
- No absolute price alerts.
- No COIN-M support in this change.
- No additional derivatives metrics such as funding rate or open interest.

## Requirements

### Functional

1. A monitoring item can represent either:
   - Spot market using last price derived from spot klines.
   - USDⓈ-M perpetual market using mark price derived from mark price klines.
2. Alert calculation remains percentage change over a configured window in minutes.
3. Feishu messages must show whether the alert is from spot or perpetual mark price.
4. Spot and perpetual monitors for the same symbol must not share cooldown or state.
5. Existing configs that do not specify market metadata must continue to behave as spot monitoring.

### Validation

- `market=spot` only supports `price_source=last` in this phase.
- `market=usdm_perp` only supports `price_source=mark` in this phase.
- Invalid combinations should fail at startup with a clear config error.

## Proposed Configuration

Each symbol entry is extended as follows:

```yaml
symbols:
  - symbol: BTCUSDT
    market: spot
    price_source: last
    window_minutes: 15
    threshold_pct: 2.0

  - symbol: BTCUSDT
    market: usdm_perp
    price_source: mark
    window_minutes: 5
    threshold_pct: 1.5
```

### Backward Compatibility

When fields are omitted, defaults are applied:

- `market: spot`
- `price_source: last`

This preserves the behavior of existing config files.

## API Strategy

Use Binance REST endpoints, selected by market type and price source.

### Spot

- Endpoint family: spot market data
- Source for alert computation: spot klines
- Semantics: use the first candle open and last candle close across the requested window, preserving current behavior

### USDⓈ-M Perpetual Mark Price

- Endpoint family: USDⓈ-M futures market data
- Source for alert computation: mark price klines
- Semantics: use the first mark-price kline open and the last mark-price kline close across the requested window

## Architecture

### 1. Config Model

Extend `SymbolConfig` with:

- `Market string`
- `PriceSource string`

Add defaulting and validation logic in `internal/config/config.go`.

### 2. Quote Identity

Introduce a stable identity for a monitoring item, derived from:

- market
- symbol
- price source

Example key format:

```text
spot:BTCUSDT:last
usdm_perp:BTCUSDT:mark
```

This key will replace symbol-only keys in deduplication and price tracking.

### 3. Market Data Abstraction

Add a small abstraction to decouple the scheduler from Binance endpoint selection.

Suggested shape:

```go
type SeriesProvider interface {
    FetchSeries(item config.SymbolConfig, interval string, limit int) ([]binance.Kline, error)
}
```

Implementation options:

- Keep one Binance client with multiple fetch methods.
- Or wrap the Binance client in a resolver that chooses the correct fetch method.

The preferred implementation is a single Binance client with explicit methods:

- `FetchSpotKlines(...)`
- `FetchUSDMMarkPriceKlines(...)`

Then the scheduler uses a resolver function to select the correct method based on config.

This keeps the project small while still separating endpoint-specific logic.

### 4. Scheduler Flow

The polling loop remains unchanged at a high level:

1. Iterate over configured monitoring items.
2. Resolve interval and limit from the time window.
3. Fetch the correct series for that item.
4. Compute percentage change.
5. Apply cooldown using the monitoring-item identity.
6. Send Feishu alert.

The existing `Tracker` can either:

- continue to store the latest point using the new identity key, or
- be simplified later if it remains unused for alert decisions.

This change should keep the tracker but migrate its keys to the new identity. Broader tracker cleanup is out of scope.

### 5. Alert Presentation

Extend the alert payload to include market context.

Examples:

- `市场：现货`
- `市场：永续 (USDⓈ-M Mark Price)`

Title examples:

- `Price Alert: BTCUSDT [现货] +2.10% (15min)`
- `Price Alert: BTCUSDT [永续-Mark] -1.80% (5min)`

Link behavior:

- Spot alerts link to the spot trading page.
- USDⓈ-M perpetual alerts link to the futures trading page.

## Data Flow

```text
config.yaml
  -> config load/default/validate
  -> scheduler loop
  -> endpoint selection by market + price_source
  -> Binance REST request
  -> normalized kline-like series
  -> percentage change calculation
  -> cooldown check using monitor identity
  -> Feishu card with market label and correct trading link
```

## Error Handling

- Invalid market or price source combinations fail during startup validation.
- Individual request failures are logged and do not stop monitoring of other items.
- Empty series responses log a warning and skip alert evaluation for that cycle.
- Feishu send failures do not mark the item as alerted.

## Testing Plan

### Config

- Defaults for omitted `market` and `price_source`.
- Validation rejects unsupported combinations.

### Endpoint Selection

- Spot items resolve to spot kline fetch.
- USDⓈ-M mark items resolve to mark price kline fetch.

### Identity and Cooldown

- `spot:BTCUSDT:last` and `usdm_perp:BTCUSDT:mark` do not suppress each other.

### Alert Rendering

- Market label is included in the Feishu card.
- Trading link changes by market type.

## Rollout Notes

- Existing users can upgrade without changing config.
- New perpetual monitoring can be enabled incrementally by adding entries.
- A future enhancement can replace or supplement polling with WebSocket without changing the config model.

## Implementation Notes

The smallest clean implementation is:

1. Extend config fields and validation.
2. Add explicit Binance client methods for spot klines and USDⓈ-M mark price klines.
3. Add a monitoring-item identity helper.
4. Update scheduler to resolve the correct fetcher and use identity keys.
5. Update Feishu card content and link generation.
6. Add focused unit tests for config defaults, validation, and identity separation.

## Open Decision Log

- Chosen market for perpetual support in phase one: Binance USDⓈ-M perpetual.
- Chosen price source for perpetual support in phase one: mark price.
- Chosen alert rule: unchanged percentage move over a rolling time window.
- Chosen transport: REST polling, not WebSocket.
