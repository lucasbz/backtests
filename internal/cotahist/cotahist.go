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

// LoadCandles reads ticker's candles for the inclusive [from, to] range from
// DefaultCotahistDir, sorted by date.
func LoadCandles(ticker string, from, to time.Time) ([]domain.Candle, error) {
	return LoadCandlesFrom(DefaultCotahistDir, ticker, from, to)
}

// LoadCandlesFrom reads ticker's candles for the inclusive [from, to] range
// out of dir, sorted by date. It reads one file per year in the range
// (missing year files are skipped) and filters out any dates outside the
// range.
func LoadCandlesFrom(dir, ticker string, from, to time.Time) ([]domain.Candle, error) {
	var candles []domain.Candle
	for year := from.Year(); year <= to.Year(); year++ {
		path := filepath.Join(dir, ticker, fmt.Sprintf("%s_%d.json", ticker, year))
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		var yearCandles []domain.Candle
		if err := json.Unmarshal(data, &yearCandles); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		candles = append(candles, yearCandles...)
	}

	filtered := candles[:0]
	for _, c := range candles {
		date, err := time.Parse("2006-01-02", c.Date)
		if err != nil {
			return nil, fmt.Errorf("parsing candle date %q: %w", c.Date, err)
		}
		if date.Before(from) || date.After(to) {
			continue
		}
		filtered = append(filtered, c)
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Date < filtered[j].Date })
	return filtered, nil
}

// DateRange returns the earliest and latest candle dates available for
// ticker in DefaultCotahistDir, as "YYYY-MM-DD" strings.
func DateRange(ticker string) (earliest, latest string, err error) {
	return DateRangeFrom(DefaultCotahistDir, ticker)
}

// DateRangeFrom returns the earliest and latest candle dates available for
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

	firstYearCandles, err := readYearFile(dir, ticker, years[0])
	if err != nil {
		return "", "", err
	}
	lastYearCandles, err := readYearFile(dir, ticker, years[len(years)-1])
	if err != nil {
		return "", "", err
	}

	// Each year's file is already sorted by date (scripts/import_cotahist.go
	// sorts before writing).
	return firstYearCandles[0].Date, lastYearCandles[len(lastYearCandles)-1].Date, nil
}

// ListTickers returns the list of tickers with any imported data in
// DefaultCotahistDir. If year is non-zero, only tickers that have a
// <TICKER>_<year>.json file are included, and the result is sorted by that
// year's total trading volume, descending (most-traded first). If year is
// zero, the result is sorted alphabetically instead, since there's no
// single year's volume to rank tickers by.
func ListTickers(year int) ([]string, error) {
	return ListTickersFrom(DefaultCotahistDir, year)
}

// ListTickersFrom returns the list of tickers with any imported data under
// dir (one subdirectory per ticker). If year is non-zero, only tickers that
// have a <TICKER>_<year>.json file inside their directory are included, and
// each matching file is read to sum its candles' Volume so the result can
// be sorted by that year's total trading volume, descending (most-traded
// first; ties broken alphabetically by ticker). If year is zero, no file
// contents are read and the result is sorted alphabetically instead.
func ListTickersFrom(dir string, year int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	if year == 0 {
		tickers := make([]string, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			tickers = append(tickers, entry.Name())
		}
		sort.Strings(tickers)
		return tickers, nil
	}

	type tickerVolume struct {
		ticker string
		volume int64
	}
	volumes := make([]tickerVolume, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ticker := entry.Name()

		path := filepath.Join(dir, ticker, fmt.Sprintf("%s_%d.json", ticker, year))
		if _, err := os.Stat(path); err != nil {
			continue
		}

		candles, err := readYearFile(dir, ticker, year)
		if err != nil {
			return nil, err
		}
		var total int64
		for _, c := range candles {
			total += c.Volume.Amount()
		}
		volumes = append(volumes, tickerVolume{ticker: ticker, volume: total})
	}

	sort.Slice(volumes, func(i, j int) bool {
		if volumes[i].volume != volumes[j].volume {
			return volumes[i].volume > volumes[j].volume
		}
		return volumes[i].ticker < volumes[j].ticker
	})

	tickers := make([]string, len(volumes))
	for i, tv := range volumes {
		tickers[i] = tv.ticker
	}
	return tickers, nil
}

func readYearFile(dir, ticker string, year int) ([]domain.Candle, error) {
	path := filepath.Join(dir, ticker, fmt.Sprintf("%s_%d.json", ticker, year))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var candles []domain.Candle
	if err := json.Unmarshal(data, &candles); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(candles) == 0 {
		return nil, fmt.Errorf("%s has no candles", path)
	}
	return candles, nil
}
