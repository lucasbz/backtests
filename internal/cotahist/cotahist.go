package cotahist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lucasbz/backtests/internal/domain"
)

// DefaultCotahistDir is the root directory produced by
// scripts/import_cotahist.go: <dir>/<TICKER>/<TICKER>_<YEAR>.json.
const DefaultCotahistDir = "resources/cotahist"

// LoadQuotes reads ticker's quotes for the inclusive [from, to] range from
// DefaultCotahistDir, sorted by date.
func LoadQuotes(ticker string, from, to time.Time) ([]domain.Quote, error) {
	return LoadQuotesFrom(DefaultCotahistDir, ticker, from, to)
}

// LoadQuotesFrom reads ticker's quotes for the inclusive [from, to] range out
// of dir, sorted by date. It reads one file per year in the range (missing
// year files are skipped) and filters out any dates outside the range.
func LoadQuotesFrom(dir, ticker string, from, to time.Time) ([]domain.Quote, error) {
	var quotes []domain.Quote
	for year := from.Year(); year <= to.Year(); year++ {
		path := filepath.Join(dir, ticker, fmt.Sprintf("%s_%d.json", ticker, year))
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		var yearQuotes []domain.Quote
		if err := json.Unmarshal(data, &yearQuotes); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		quotes = append(quotes, yearQuotes...)
	}

	filtered := quotes[:0]
	for _, q := range quotes {
		date, err := time.Parse("2006-01-02", q.Date)
		if err != nil {
			return nil, fmt.Errorf("parsing quote date %q: %w", q.Date, err)
		}
		if date.Before(from) || date.After(to) {
			continue
		}
		filtered = append(filtered, q)
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Date < filtered[j].Date })
	return filtered, nil
}
