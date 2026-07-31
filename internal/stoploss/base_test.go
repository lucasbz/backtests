package stoploss

import (
	"testing"
)

func TestAvailableStopLossNamesList_Sorted(t *testing.T) {
	names := AvailableStopLossNamesList()
	if len(names) == 0 {
		t.Fatal("AvailableStopLossNamesList() is empty")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("names = %v, want sorted", names)
			break
		}
	}
}

func TestLoadStopLoss_Percent(t *testing.T) {
	sl, err := LoadStopLoss("percent", 5)
	if err != nil {
		t.Fatalf("LoadStopLoss: %v", err)
	}
	pct, ok := sl.(*PercentStopLoss)
	if !ok {
		t.Fatalf("LoadStopLoss(%q) = %T, want *PercentStopLoss", "percent", sl)
	}
	if pct.Percent != 5 {
		t.Errorf("Percent = %v, want 5", pct.Percent)
	}
}

func TestLoadStopLoss_FixedAmount(t *testing.T) {
	sl, err := LoadStopLoss("fixed-amount", 2.5)
	if err != nil {
		t.Fatalf("LoadStopLoss: %v", err)
	}
	fixed, ok := sl.(*FixedAmountStopLoss)
	if !ok {
		t.Fatalf("LoadStopLoss(%q) = %T, want *FixedAmountStopLoss", "fixed-amount", sl)
	}
	if fixed.Amount.Amount() != 250 {
		t.Errorf("Amount = %d, want 250 (R$2.50 in cents)", fixed.Amount.Amount())
	}
}

func TestLoadStopLoss_Unknown(t *testing.T) {
	if _, err := LoadStopLoss("does-not-exist", 5); err == nil {
		t.Error("LoadStopLoss(unknown) = nil error, want an error")
	}
}

func TestLoadStopLoss_NonPositiveValue(t *testing.T) {
	for _, name := range AvailableStopLossNamesList() {
		if name == "none" {
			// "none" is the odd one out: it has no meaningful trigger
			// value, so 0 (not a positive value) is exactly what it
			// requires. See TestLoadStopLoss_None.
			continue
		}
		if _, err := LoadStopLoss(name, 0); err == nil {
			t.Errorf("LoadStopLoss(%q, 0) = nil error, want an error", name)
		}
		if _, err := LoadStopLoss(name, -1); err == nil {
			t.Errorf("LoadStopLoss(%q, -1) = nil error, want an error", name)
		}
	}
}

func TestLoadStopLoss_None(t *testing.T) {
	sl, err := LoadStopLoss("none", 0)
	if err != nil {
		t.Fatalf("LoadStopLoss: %v", err)
	}
	if _, ok := sl.(*NoStopLoss); !ok {
		t.Fatalf("LoadStopLoss(%q) = %T, want *NoStopLoss", "none", sl)
	}
}

func TestLoadStopLoss_None_NonZeroValue(t *testing.T) {
	if _, err := LoadStopLoss("none", 5); err == nil {
		t.Error("LoadStopLoss(\"none\", 5) = nil error, want an error (none takes no value)")
	}
	if _, err := LoadStopLoss("none", -1); err == nil {
		t.Error("LoadStopLoss(\"none\", -1) = nil error, want an error (none takes no value)")
	}
}
