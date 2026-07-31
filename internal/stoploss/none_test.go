package stoploss

import (
	"testing"

	"github.com/lucasbz/backtests/internal/domain"
)

func TestNoStopLoss_Name(t *testing.T) {
	s := &NoStopLoss{}
	if s.Name() != "None" {
		t.Errorf("Name() = %q, want %q", s.Name(), "None")
	}
}

func TestNoStopLoss_NeverTriggers(t *testing.T) {
	s := &NoStopLoss{}

	cases := []struct {
		label    string
		position domain.Position
		candle   domain.Candle
	}{
		{
			label:    "low far below entry",
			position: stopLossTestPosition(1000, 10),
			candle:   stopLossTestCandle("2010-01-05", 1, 980),
		},
		{
			label:    "low exactly at entry",
			position: stopLossTestPosition(1000, 10),
			candle:   stopLossTestCandle("2010-01-05", 1000, 1050),
		},
		{
			label:    "low above entry",
			position: stopLossTestPosition(1000, 10),
			candle:   stopLossTestCandle("2010-01-05", 1100, 1200),
		},
		{
			label:    "zero-price position",
			position: stopLossTestPosition(0, 5),
			candle:   stopLossTestCandle("2010-01-05", 0, 100),
		},
	}

	for _, c := range cases {
		if order := s.Check(c.candle, c.position); order != nil {
			t.Errorf("%s: Check() = %+v, want nil (never triggers)", c.label, order)
		}
	}
}
