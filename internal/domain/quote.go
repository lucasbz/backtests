package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/Rhymond/go-money"
)

// Currency is the currency every Quote/Order/Result money value is
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

// ParseMoney parses a plain decimal string (e.g. "10000.00", from a CLI
// flag) into a money.Money in Currency.
func ParseMoney(s string) (money.Money, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return money.Money{}, fmt.Errorf("parsing money %q: %w", s, err)
	}
	return moneyFromFloat(f), nil
}

type Quote struct {
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

// quoteJSON mirrors the plain-decimal shape scripts/import_cotahist.go
// writes: money fields are bare JSON numbers, not go-money's default
// {"amount":...,"currency":...} object form.
type quoteJSON struct {
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

func (q Quote) MarshalJSON() ([]byte, error) {
	return json.Marshal(quoteJSON{
		Date:     q.Date,
		Open:     q.Open.AsMajorUnits(),
		High:     q.High.AsMajorUnits(),
		Low:      q.Low.AsMajorUnits(),
		Avg:      q.Avg.AsMajorUnits(),
		Close:    q.Close.AsMajorUnits(),
		Quantity: q.Quantity,
		Volume:   q.Volume.AsMajorUnits(),
		Trades:   q.Trades,
	})
}

func (q *Quote) UnmarshalJSON(data []byte) error {
	var raw quoteJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing quote: %w", err)
	}

	q.Date = raw.Date
	q.Open = moneyFromFloat(raw.Open)
	q.High = moneyFromFloat(raw.High)
	q.Low = moneyFromFloat(raw.Low)
	q.Avg = moneyFromFloat(raw.Avg)
	q.Close = moneyFromFloat(raw.Close)
	q.Quantity = raw.Quantity
	q.Volume = moneyFromFloat(raw.Volume)
	q.Trades = raw.Trades
	return nil
}
