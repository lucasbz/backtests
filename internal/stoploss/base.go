// Package stoploss provides pluggable domain.StopLoss implementations - a
// sibling concept to internal/strategies, following the same registry
// pattern (see internal/strategies/base.go), but kept in its own package
// since a stop-loss is a distinct concept from a strategy: it only ever
// evaluates an exit once a position is already open (see
// domain.StopLoss), never an entry.
package stoploss

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lucasbz/backtests/internal/domain"
)

// availableStopLosses maps a stop-loss type name (the API/CLI-facing slug,
// e.g. "percent") to a constructor for the concrete domain.StopLoss it
// should run, given the type's configured trigger value. Mirrors
// availableStrategies in internal/strategies/base.go.
//
// value's meaning depends on the type:
//   - "percent": a percentage below the entry price, e.g. 5 for 5%.
//   - "fixed-amount": a currency amount below the entry price, in BRL
//     major units, e.g. 2.5 for R$2.50.
//   - "none": unused - there's no trigger to configure, so value must be 0.
//     This gives callers an explicit "no stop-loss" type to select, as an
//     alternative to omitting the stop-loss configuration entirely.
var availableStopLosses = map[string]func(value float64) (domain.StopLoss, error){
	"percent": func(value float64) (domain.StopLoss, error) {
		// value must be strictly between 0 and 100: at or above 100,
		// PercentStopLoss.triggerPrice produces a zero/negative trigger
		// price, and since a candle's Low can never be negative, Check's
		// guard is always true - the stop-loss would silently never fire,
		// becoming a permanent no-op instead of a rejected invalid config.
		if value <= 0 || value >= 100 {
			return nil, fmt.Errorf("percent stop-loss value must be greater than zero and less than 100, got %v", value)
		}
		return &PercentStopLoss{Percent: value}, nil
	},
	"fixed-amount": func(value float64) (domain.StopLoss, error) {
		if value <= 0 {
			return nil, fmt.Errorf("fixed-amount stop-loss value must be greater than zero, got %v", value)
		}
		return &FixedAmountStopLoss{Amount: domain.MoneyFromFloat(value)}, nil
	},
	"none": func(value float64) (domain.StopLoss, error) {
		if value != 0 {
			return nil, fmt.Errorf("none stop-loss value must be 0 (unused), got %v", value)
		}
		return &NoStopLoss{}, nil
	},
}

// AvailableStopLossNamesList returns every registered stop-loss type name,
// sorted. Unlike AvailableStopLossNames (a comma-joined string meant for
// CLI help text), this is meant for callers that want the raw list, e.g. to
// serialize as a JSON array.
func AvailableStopLossNamesList() []string {
	names := make([]string, 0, len(availableStopLosses))
	for name := range availableStopLosses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func AvailableStopLossNames() string {
	return strings.Join(AvailableStopLossNamesList(), ", ")
}

// LoadStopLoss builds a fresh instance of the named stop-loss type, seeded
// with value as its trigger configuration (see availableStopLosses for what
// value means per type). Returns an error both for an unknown type name and
// for a type-specific invalid value (e.g. zero or negative).
func LoadStopLoss(name string, value float64) (domain.StopLoss, error) {
	newStopLoss, ok := availableStopLosses[name]
	if !ok {
		return nil, fmt.Errorf("unknown stop-loss type %q, available: %s", name, AvailableStopLossNames())
	}
	return newStopLoss(value)
}
