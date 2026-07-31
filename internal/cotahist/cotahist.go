package cotahist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

// DateRange returns the earliest and latest quote dates available for
// ticker in DefaultCotahistDir, as "YYYY-MM-DD" strings.
func DateRange(ticker string) (earliest, latest string, err error) {
	return DateRangeFrom(DefaultCotahistDir, ticker)
}

// DateRangeFrom returns the earliest and latest quote dates available for
// ticker under dir, as "YYYY-MM-DD" strings. It only reads the earliest and
// latest year files present, not every year in between.
func DateRangeFrom(dir, ticker string) (earliest, latest string, err error) {
	tickerDir := filepath.Join(dir, ticker)
	entries, err := os.ReadDir(tickerDir)
	if os.IsNotExist(err) {
		return "", "", fmt.Errorf("no data found for ticker %q (not imported yet?)", ticker)
	}
	if err != nil {
		return "", "", fmt.Errorf("reading %s: %w", tickerDir, err)
	}

	prefix, suffix := ticker+"_", ".json"
	var years []int
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		year, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix))
		if err != nil {
			continue
		}
		years = append(years, year)
	}
	if len(years) == 0 {
		return "", "", fmt.Errorf("no data found for ticker %q (not imported yet?)", ticker)
	}
	sort.Ints(years)

	firstYearQuotes, err := readYearFile(dir, ticker, years[0])
	if err != nil {
		return "", "", err
	}
	lastYearQuotes, err := readYearFile(dir, ticker, years[len(years)-1])
	if err != nil {
		return "", "", err
	}

	// Each year's file is already sorted by date (scripts/import_cotahist.go
	// sorts before writing).
	return firstYearQuotes[0].Date, lastYearQuotes[len(lastYearQuotes)-1].Date, nil
}

func readYearFile(dir, ticker string, year int) ([]domain.Quote, error) {
	path := filepath.Join(dir, ticker, fmt.Sprintf("%s_%d.json", ticker, year))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var quotes []domain.Quote
	if err := json.Unmarshal(data, &quotes); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(quotes) == 0 {
		return nil, fmt.Errorf("%s has no quotes", path)
	}
	return quotes, nil
}
