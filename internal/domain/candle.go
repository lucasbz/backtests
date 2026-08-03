package domain

import (
	"encoding/json"
	"fmt"

	"github.com/Rhymond/go-money"

	"github.com/lucasbz/backtests/internal/util"
)

// Currency is the currency every Candle/Order/BacktestResult money value is
// denominated in. B3, the exchange scripts/import_cotahist.go pulls from,
// only ever quotes in Brazilian reais.
const Currency = money.BRL

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
	c.Open = util.MoneyFromFloat(raw.Open, Currency)
	c.High = util.MoneyFromFloat(raw.High, Currency)
	c.Low = util.MoneyFromFloat(raw.Low, Currency)
	c.Avg = util.MoneyFromFloat(raw.Avg, Currency)
	c.Close = util.MoneyFromFloat(raw.Close, Currency)
	c.Quantity = raw.Quantity
	c.Volume = util.MoneyFromFloat(raw.Volume, Currency)
	c.Trades = raw.Trades
	return nil
}
