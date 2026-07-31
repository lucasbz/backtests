package domain

import (
	"encoding/json"
	"testing"

	"github.com/Rhymond/go-money"
)

func TestCandle_UnmarshalJSON(t *testing.T) {
	// Real sample bar from resources/cotahist/SMFT3/SMFT3_2026.json.
	input := `{"date":"2026-07-27","open":19.9,"high":19.99,"low":19.65,"avg":19.8,"close":19.77,"quantity":2438000,"volume":48281780,"trades":9250}`

	var c Candle
	if err := json.Unmarshal([]byte(input), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if c.Date != "2026-07-27" {
		t.Errorf("Date = %q, want 2026-07-27", c.Date)
	}
	if c.Quantity != 2438000 {
		t.Errorf("Quantity = %d, want 2438000", c.Quantity)
	}
	if c.Trades != 9250 {
		t.Errorf("Trades = %d, want 9250", c.Trades)
	}
	assertMoneyAmount(t, "Open", c.Open, 1990)
	assertMoneyAmount(t, "High", c.High, 1999)
	assertMoneyAmount(t, "Low", c.Low, 1965)
	assertMoneyAmount(t, "Avg", c.Avg, 1980)
	assertMoneyAmount(t, "Close", c.Close, 1977)
	assertMoneyAmount(t, "Volume", c.Volume, 4828178000)
}

func assertMoneyAmount(t *testing.T, field string, got money.Money, want int64) {
	t.Helper()
	if got.Amount() != want {
		t.Errorf("%s.Amount() = %d, want %d", field, got.Amount(), want)
	}
	if got.Currency().Code != Currency {
		t.Errorf("%s.Currency = %s, want %s", field, got.Currency().Code, Currency)
	}
}

func TestCandle_MarshalJSON_RoundTrips(t *testing.T) {
	original := `{"date":"2026-07-27","open":19.90,"high":19.99,"low":19.65,"avg":19.80,"close":19.77,"quantity":2438000,"volume":48281780.00,"trades":9250}`

	var c Candle
	if err := json.Unmarshal([]byte(original), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundTripped Candle
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal round-tripped: %v", err)
	}

	if c.Date != roundTripped.Date || c.Quantity != roundTripped.Quantity || c.Trades != roundTripped.Trades {
		t.Errorf("round-trip mismatch: got %+v, want %+v", roundTripped, c)
	}
	for field, pair := range map[string][2]money.Money{
		"Open":   {c.Open, roundTripped.Open},
		"High":   {c.High, roundTripped.High},
		"Low":    {c.Low, roundTripped.Low},
		"Avg":    {c.Avg, roundTripped.Avg},
		"Close":  {c.Close, roundTripped.Close},
		"Volume": {c.Volume, roundTripped.Volume},
	} {
		if pair[0].Amount() != pair[1].Amount() {
			t.Errorf("%s round-trip mismatch: got %d, want %d", field, pair[1].Amount(), pair[0].Amount())
		}
	}
}

func TestParseMoney(t *testing.T) {
	got, err := ParseMoney("10000.00")
	if err != nil {
		t.Fatalf("ParseMoney: %v", err)
	}
	assertMoneyAmount(t, "ParseMoney(10000.00)", got, 1000000)
}

func TestParseMoney_Invalid(t *testing.T) {
	if _, err := ParseMoney("not-a-number"); err == nil {
		t.Fatal("expected error for invalid input")
	}
}
