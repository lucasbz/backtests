package strategies

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lucasbz/backtests/internal/domain"
)

// availableStrategies maps a -strategy flag value to a constructor for the
// concrete domain.Strategy it should run. It's a map of constructors (not
// shared instances) so each run gets its own strategy state instead of
// leaking state across runs. Strategies no longer need a starting balance
// at construction time - Backtest owns the running balance and quantity
// sizing (see domain.Strategy.Decide) - so, unlike before this package's
// interface moved to Decide, these constructors take no arguments.
var availableStrategies = map[string]func() domain.Strategy{
	"buy-and-hold": func() domain.Strategy {
		return &BuyAndHold{}
	},
	"two-candle-breakout": func() domain.Strategy {
		return &TwoCandleBreakout{}
	},
}

// AvailableStrategyNamesList returns every registered strategy name, sorted.
// Unlike AvailableStrategyNames (a comma-joined string meant for CLI help
// text), this is meant for callers that want the raw list, e.g. to serialize
// as a JSON array.
func AvailableStrategyNamesList() []string {
	names := make([]string, 0, len(availableStrategies))
	for name := range availableStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func AvailableStrategyNames() string {
	return strings.Join(AvailableStrategyNamesList(), ", ")
}

// LoadStrategy builds a fresh instance of the named strategy.
func LoadStrategy(strategyName string) (domain.Strategy, error) {
	newStrategy, ok := availableStrategies[strategyName]
	if !ok {
		return nil, fmt.Errorf("unknown strategy %q, available: %s", strategyName, AvailableStrategyNames())
	}
	return newStrategy(), nil
}
