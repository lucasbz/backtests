package strategies

import (
	"sort"
	"strings"
	"testing"
)

// TestAvailableStrategyNamesList checks the registry's raw name list: sorted,
// non-empty, and containing every strategy this package registers (see
// availableStrategies).
func TestAvailableStrategyNamesList(t *testing.T) {
	names := AvailableStrategyNamesList()

	want := []string{"buy-and-hold", "ema-crossover", "rsi-threshold", "sma-crossover", "two-candle-breakout"}
	if len(names) != len(want) {
		t.Fatalf("AvailableStrategyNamesList() = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("AvailableStrategyNamesList()[%d] = %q, want %q", i, names[i], name)
		}
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("AvailableStrategyNamesList() = %v, want sorted", names)
	}
}

// TestAvailableStrategyNamesList_FreshSlice checks each call returns its own
// slice, so a caller mutating the result can't corrupt subsequent calls'
// output (there's no shared backing array to leak through).
func TestAvailableStrategyNamesList_FreshSlice(t *testing.T) {
	first := AvailableStrategyNamesList()
	if len(first) == 0 {
		t.Fatal("AvailableStrategyNamesList() is empty")
	}
	first[0] = "mutated"

	second := AvailableStrategyNamesList()
	if second[0] == "mutated" {
		t.Error("AvailableStrategyNamesList() shares state across calls, want a fresh slice each time")
	}
}

// TestAvailableStrategyNames checks the comma-joined form (meant for CLI
// help text) is consistent with AvailableStrategyNamesList.
func TestAvailableStrategyNames(t *testing.T) {
	got := AvailableStrategyNames()
	want := strings.Join(AvailableStrategyNamesList(), ", ")
	if got != want {
		t.Errorf("AvailableStrategyNames() = %q, want %q", got, want)
	}
	if !strings.Contains(got, "buy-and-hold") {
		t.Errorf("AvailableStrategyNames() = %q, want it to contain %q", got, "buy-and-hold")
	}
}

// TestLoadStrategy_KnownNames checks every registered name loads a
// non-nil domain.Strategy instance with the expected display Name(), given
// valid params for whichever ones need them.
func TestLoadStrategy_KnownNames(t *testing.T) {
	cases := []struct {
		slug            string
		params          map[string]float64
		wantDisplayName string
	}{
		{"buy-and-hold", nil, "Buy & Hold"},
		{"two-candle-breakout", nil, "Two-Candle Breakout"},
		{"sma-crossover", map[string]float64{"shortPeriod": 2, "longPeriod": 5}, "SMA Crossover"},
		{"ema-crossover", map[string]float64{"shortPeriod": 2, "longPeriod": 5}, "EMA Crossover"},
		{"rsi-threshold", map[string]float64{"period": 14, "oversold": 30, "overbought": 70}, "RSI Threshold"},
	}

	for _, tc := range cases {
		s, err := LoadStrategy(tc.slug, tc.params)
		if err != nil {
			t.Fatalf("LoadStrategy(%q): %v", tc.slug, err)
		}
		if s == nil {
			t.Fatalf("LoadStrategy(%q) = nil, want a non-nil domain.Strategy", tc.slug)
		}
		if got := s.Name(); got != tc.wantDisplayName {
			t.Errorf("LoadStrategy(%q).Name() = %q, want %q", tc.slug, got, tc.wantDisplayName)
		}
	}
}

// TestLoadStrategy_ReturnsFreshInstance checks that two separate
// LoadStrategy calls return distinct instances, not a shared singleton
// (see availableStrategies' doc comment on why that matters: each backtest
// run needs its own strategy state). Uses "two-candle-breakout" rather than
// "buy-and-hold" because BuyAndHold is a zero-size struct{} - Go's runtime
// can (and does) hand out the same address for repeated zero-size
// allocations, making pointer-identity an unreliable/misleading check for
// it specifically; TwoCandleBreakout carries a real (if initially nil)
// window field, so its instances are guaranteed distinct addresses.
func TestLoadStrategy_ReturnsFreshInstance(t *testing.T) {
	a, err := LoadStrategy("two-candle-breakout", nil)
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	b, err := LoadStrategy("two-candle-breakout", nil)
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	if a == b {
		t.Error("LoadStrategy returned the same instance twice, want a fresh instance per call")
	}
}

func TestLoadStrategy_Unknown(t *testing.T) {
	s, err := LoadStrategy("does-not-exist", nil)
	if err == nil {
		t.Fatal("LoadStrategy(unknown) = nil error, want an error")
	}
	if s != nil {
		t.Errorf("LoadStrategy(unknown) = %+v, want nil", s)
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("error = %q, want it to mention the unknown name", err.Error())
	}
}

// TestAvailableStrategyInfo checks the descriptor list: same names/order as
// AvailableStrategyNamesList, zero-param strategies get an empty (non-nil)
// Params slice, and multi-param strategies expose their declared keys in
// the order strategyParams lists them.
func TestAvailableStrategyInfo(t *testing.T) {
	infos := AvailableStrategyInfo()
	names := AvailableStrategyNamesList()

	if len(infos) != len(names) {
		t.Fatalf("AvailableStrategyInfo() has %d entries, want %d (one per AvailableStrategyNamesList entry)", len(infos), len(names))
	}
	for i, info := range infos {
		if info.Name != names[i] {
			t.Errorf("AvailableStrategyInfo()[%d].Name = %q, want %q (same order as AvailableStrategyNamesList)", i, info.Name, names[i])
		}
	}

	byName := map[string]StrategyInfo{}
	for _, info := range infos {
		byName[info.Name] = info
	}

	for _, zeroParam := range []string{"buy-and-hold", "two-candle-breakout"} {
		info, ok := byName[zeroParam]
		if !ok {
			t.Fatalf("AvailableStrategyInfo() missing %q", zeroParam)
		}
		if info.Params == nil {
			t.Errorf("%s.Params = nil, want an empty (non-nil) slice", zeroParam)
		}
		if len(info.Params) != 0 {
			t.Errorf("%s.Params = %+v, want empty", zeroParam, info.Params)
		}
	}

	smaCrossover, ok := byName["sma-crossover"]
	if !ok {
		t.Fatal("AvailableStrategyInfo() missing \"sma-crossover\"")
	}
	wantKeys := []string{"shortPeriod", "longPeriod"}
	if len(smaCrossover.Params) != len(wantKeys) {
		t.Fatalf("sma-crossover.Params = %+v, want %d entries", smaCrossover.Params, len(wantKeys))
	}
	for i, key := range wantKeys {
		if smaCrossover.Params[i].Key != key {
			t.Errorf("sma-crossover.Params[%d].Key = %q, want %q", i, smaCrossover.Params[i].Key, key)
		}
	}

	rsiThreshold, ok := byName["rsi-threshold"]
	if !ok {
		t.Fatal("AvailableStrategyInfo() missing \"rsi-threshold\"")
	}
	foundBoundedMax := false
	for _, p := range rsiThreshold.Params {
		if p.Key == "oversold" || p.Key == "overbought" {
			if p.Max == nil {
				t.Errorf("rsi-threshold.%s.Max = nil, want a bounded max (100)", p.Key)
				continue
			}
			if *p.Max != 100 {
				t.Errorf("rsi-threshold.%s.Max = %v, want 100", p.Key, *p.Max)
			}
			foundBoundedMax = true
		}
		if p.Key == "period" && p.Max != nil {
			t.Errorf("rsi-threshold.period.Max = %v, want nil (unbounded)", *p.Max)
		}
	}
	if !foundBoundedMax {
		t.Error("rsi-threshold.Params has no oversold/overbought entry with a bounded Max")
	}
}

// TestAvailableStrategyInfo_FreshSlice checks each call's Params slices are
// independent, so a caller mutating the result can't corrupt subsequent
// calls' output.
func TestAvailableStrategyInfo_FreshSlice(t *testing.T) {
	first := AvailableStrategyInfo()
	for i, info := range first {
		if len(info.Params) > 0 {
			first[i].Params[0].Key = "mutated"
			break
		}
	}

	second := AvailableStrategyInfo()
	for _, info := range second {
		for _, p := range info.Params {
			if p.Key == "mutated" {
				t.Error("AvailableStrategyInfo() shares Params backing state across calls, want fresh slices each time")
			}
		}
	}
}

func TestBuyAndHold_Name(t *testing.T) {
	s := &BuyAndHold{}
	if got := s.Name(); got != "Buy & Hold" {
		t.Errorf("Name() = %q, want %q", got, "Buy & Hold")
	}
}

func TestTwoCandleBreakout_Name(t *testing.T) {
	s := &TwoCandleBreakout{}
	if got := s.Name(); got != "Two-Candle Breakout" {
		t.Errorf("Name() = %q, want %q", got, "Two-Candle Breakout")
	}
}
