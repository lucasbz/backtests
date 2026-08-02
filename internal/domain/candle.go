package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/Rhymond/go-money"
)

// Currency is the currency every Candle/Order/BacktestResult money value is
// denominated in. B3, the exchange scripts/import_cotahist.go pulls from,
// only ever quotes in Brazilian reais.
const Currency = money.BRL

// moneyFromFloat converts a decimal value (e.g. 19.9) to money.Money.
// money.NewFromFloat truncates (int64(amount*100)) instead of rounding, so
// float64 imprecision silently corrupts values like 19.9 -> 1989 instead of
// 1990; rounding first and going through money.New avoids that.
func moneyFromFloat(f float64) money.Money {
	return *money.New(int64(math.Round(f*100)), Currency)
}

// MoneyFromFloat is the exported form of moneyFromFloat, for callers
// outside this package that need to convert a plain decimal value (e.g. a
// JSON number from an API request) to money.Money - e.g. a fixed-amount
// stop-loss's currency value. Prefer ParseMoney when the source is already
// a string (it goes through the same rounding).
func MoneyFromFloat(f float64) money.Money {
	return moneyFromFloat(f)
}

// ParseMoney parses a plain decimal string (e.g. "10000.00", from a CLI
// flag) into a money.Money in Currency.
//
// strconv.ParseFloat happily accepts "Inf"/"+Inf"/"-Inf"/"NaN" (and their
// case-insensitive variants) as valid float64 values with no error; feeding
// one of those into moneyFromFloat's math.Round(f*100) then int64(...)
// conversion silently saturates to math.MaxInt64 (or NaN converts to 0)
// instead of erroring, which would let a caller like handleBacktest's
// "balance must be positive" check pass trivially on a bogus,
// astronomically large "balance". Reject non-finite input explicitly
// instead.
func ParseMoney(s string) (money.Money, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return money.Money{}, fmt.Errorf("parsing money %q: %w", s, err)
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return money.Money{}, fmt.Errorf("parsing money %q: value must be a finite number", s)
	}
	return moneyFromFloat(f), nil
}

type Candle struct {
	Date     string
	Open     money.Money
	High     money.Money
	Low      money.Money
	Avg      money.Money
	Close    money.Money
	Quantity int64
	Volume   money.Money
	Trades   int64
}

// candleJSON mirrors the plain-decimal shape scripts/import_cotahist.go
// writes: money fields are bare JSON numbers, not go-money's default
// {"amount":...,"currency":...} object form.
type candleJSON struct {
	Date     string  `json:"date"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Avg      float64 `json:"avg"`
	Close    float64 `json:"close"`
	Quantity int64   `json:"quantity"`
	Volume   float64 `json:"volume"`
	Trades   int64   `json:"trades"`
}

func (c Candle) MarshalJSON() ([]byte, error) {
	return json.Marshal(candleJSON{
		Date:     c.Date,
		Open:     c.Open.AsMajorUnits(),
		High:     c.High.AsMajorUnits(),
		Low:      c.Low.AsMajorUnits(),
		Avg:      c.Avg.AsMajorUnits(),
		Close:    c.Close.AsMajorUnits(),
		Quantity: c.Quantity,
		Volume:   c.Volume.AsMajorUnits(),
		Trades:   c.Trades,
	})
}

func (c *Candle) UnmarshalJSON(data []byte) error {
	var raw candleJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing candle: %w", err)
	}

	c.Date = raw.Date
	c.Open = moneyFromFloat(raw.Open)
	c.High = moneyFromFloat(raw.High)
	c.Low = moneyFromFloat(raw.Low)
	c.Avg = moneyFromFloat(raw.Avg)
	c.Close = moneyFromFloat(raw.Close)
	c.Quantity = raw.Quantity
	c.Volume = moneyFromFloat(raw.Volume)
	c.Trades = raw.Trades
	return nil
}
