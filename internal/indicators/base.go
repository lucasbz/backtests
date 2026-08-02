package indicators

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// availableIndicators maps an indicator type name (used by
// internal/strategies constructors, never directly exposed over the API)
// to a constructor for the concrete Indicator it should run, given its
// configured params. Mirrors internal/stoploss.availableStopLosses'
// shape, generalized from a single value float64 to a named-params map,
// since an indicator can need more than one number (an RSI's period plus
// two thresholds live on the strategy that owns it, but even an indicator
// alone already needs more than a bare value for e.g. a future
// multi-param indicator - see docs/plans/indicators.md).
//
// Every indicator currently registered here takes a single "period" param:
// a positive whole number of candles.
var availableIndicators = map[string]func(params map[string]float64) (Indicator, error){
	"sma": func(params map[string]float64) (Indicator, error) {
		period, err := periodParam(params)
		if err != nil {
			return nil, fmt.Errorf("sma: %w", err)
		}
		return &SMA{Period: period}, nil
	},
	"ema": func(params map[string]float64) (Indicator, error) {
		period, err := periodParam(params)
		if err != nil {
			return nil, fmt.Errorf("ema: %w", err)
		}
		return &EMA{Period: period}, nil
	},
	"rsi": func(params map[string]float64) (Indicator, error) {
		period, err := periodParam(params)
		if err != nil {
			return nil, fmt.Errorf("rsi: %w", err)
		}
		return &RSI{Period: period}, nil
	},
}

// periodParam extracts and validates the "period" param shared by every
// indicator registered above: it must be present and a positive whole
// number.
func periodParam(params map[string]float64) (int, error) {
	value, ok := params["period"]
	if !ok {
		return 0, fmt.Errorf(`missing required "period" param`)
	}
	if value <= 0 || value != math.Trunc(value) {
		return 0, fmt.Errorf(`"period" param must be a positive whole number, got %v`, value)
	}
	return int(value), nil
}

// AvailableIndicatorNamesList returns every registered indicator type
// name, sorted.
func AvailableIndicatorNamesList() []string {
	names := make([]string, 0, len(availableIndicators))
	for name := range availableIndicators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func AvailableIndicatorNames() string {
	return strings.Join(AvailableIndicatorNamesList(), ", ")
}

// LoadIndicator builds a fresh instance of the named indicator type,
// configured from params (see availableIndicators for what each type
// expects). Returns an error both for an unknown type name and for
// type-specific invalid params (e.g. a missing or non-positive period).
func LoadIndicator(name string, params map[string]float64) (Indicator, error) {
	newIndicator, ok := availableIndicators[name]
	if !ok {
		return nil, fmt.Errorf("unknown indicator %q, available: %s", name, AvailableIndicatorNames())
	}
	return newIndicator(params)
}
