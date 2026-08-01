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

	want := []string{"buy-and-hold", "two-candle-breakout"}
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
// non-nil domain.Strategy instance with the expected display Name().
func TestLoadStrategy_KnownNames(t *testing.T) {
	cases := map[string]string{
		"buy-and-hold":        "Buy & Hold",
		"two-candle-breakout": "Two-Candle Breakout",
	}

	for slug, wantDisplayName := range cases {
		s, err := LoadStrategy(slug)
		if err != nil {
			t.Fatalf("LoadStrategy(%q): %v", slug, err)
		}
		if s == nil {
			t.Fatalf("LoadStrategy(%q) = nil, want a non-nil domain.Strategy", slug)
		}
		if got := s.Name(); got != wantDisplayName {
			t.Errorf("LoadStrategy(%q).Name() = %q, want %q", slug, got, wantDisplayName)
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
	a, err := LoadStrategy("two-candle-breakout")
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	b, err := LoadStrategy("two-candle-breakout")
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	if a == b {
		t.Error("LoadStrategy returned the same instance twice, want a fresh instance per call")
	}
}

func TestLoadStrategy_Unknown(t *testing.T) {
	s, err := LoadStrategy("does-not-exist")
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
