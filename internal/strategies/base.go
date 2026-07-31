package strategies

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// availableStrategies maps a -strategy flag value to a constructor for the
// concrete domain.Strategy it should run, given the run's starting balance.
// It's a map of constructors (not shared instances) so each run gets its own
// strategy state instead of leaking traversal state across runs.
var availableStrategies = map[string]func(balance money.Money) domain.Strategy{
	"buy-and-hold": func(balance money.Money) domain.Strategy {
		return &BuyAndHold{Balance: balance}
	},
	"two-candle-breakout": func(balance money.Money) domain.Strategy {
		return &TwoCandleBreakout{Balance: balance}
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

// LoadStrategy builds a fresh instance of the named strategy, seeded with
// balance as its starting cash for position sizing.
func LoadStrategy(strategyName string, balance money.Money) (domain.Strategy, error) {
	newStrategy, ok := availableStrategies[strategyName]
	if !ok {
		return nil, fmt.Errorf("unknown strategy %q, available: %s", strategyName, AvailableStrategyNames())
	}
	return newStrategy(balance), nil
}
