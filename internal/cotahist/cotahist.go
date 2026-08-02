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

	"github.com/Rhymond/go-money"

	"github.com/lucasbz/backtests/internal/domain"
)

// DefaultCotahistDir is the root directory produced by
// scripts/import_cotahist.go: <dir>/<TICKER>/<TICKER>_<YEAR>.json.
const DefaultCotahistDir = "resources/cotahist"

// LoadCandles reads asset's candles for the inclusive [from, to] range from
// DefaultCotahistDir, sorted by date.
func LoadCandles(asset string, from, to time.Time) ([]domain.Candle, error) {
	return LoadCandlesFrom(DefaultCotahistDir, asset, from, to)
}

// LoadCandlesFrom reads asset's candles for the inclusive [from, to] range
// out of dir, sorted by date. It reads one file per year in the range
// (missing year files are skipped) and filters out any dates outside the
// range.
func LoadCandlesFrom(dir, asset string, from, to time.Time) ([]domain.Candle, error) {
	var candles []domain.Candle
	for year := from.Year(); year <= to.Year(); year++ {
		path := filepath.Join(dir, asset, fmt.Sprintf("%s_%d.json", asset, year))
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
// asset in DefaultCotahistDir, as "YYYY-MM-DD" strings.
func DateRange(asset string) (earliest, latest string, err error) {
	return DateRangeFrom(DefaultCotahistDir, asset)
}

// DateRangeFrom returns the earliest and latest candle dates available for
// asset under dir, as "YYYY-MM-DD" strings. It only reads the earliest and
// latest year files present, not every year in between.
func DateRangeFrom(dir, asset string) (earliest, latest string, err error) {
	assetDir := filepath.Join(dir, asset)
	entries, err := os.ReadDir(assetDir)
	if os.IsNotExist(err) {
		return "", "", fmt.Errorf("no data found for asset %q (not imported yet?)", asset)
	}
	if err != nil {
		return "", "", fmt.Errorf("reading %s: %w", assetDir, err)
	}

	prefix, suffix := asset+"_", ".json"
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
		return "", "", fmt.Errorf("no data found for asset %q (not imported yet?)", asset)
	}
	sort.Ints(years)

	firstYearCandles, err := readYearFile(dir, asset, years[0])
	if err != nil {
		return "", "", err
	}
	lastYearCandles, err := readYearFile(dir, asset, years[len(years)-1])
	if err != nil {
		return "", "", err
	}

	// Each year's file is already sorted by date (scripts/import_cotahist.go
	// sorts before writing).
	return firstYearCandles[0].Date, lastYearCandles[len(lastYearCandles)-1].Date, nil
}

// ListAssets returns the list of tickers with any imported data in
// DefaultCotahistDir. If year is non-zero, only tickers that have a
// <TICKER>_<year>.json file are included, and the result is sorted by that
// year's total trading volume, descending (most-traded first). If year is
// zero, the result is sorted alphabetically instead, since there's no
// single year's volume to rank tickers by.
func ListAssets(year int) ([]string, error) {
	return ListAssetsFrom(DefaultCotahistDir, year)
}

// ListAssetsFrom returns the list of tickers with any imported data under
// dir (one subdirectory per ticker). If year is non-zero, only tickers that
// have a <TICKER>_<year>.json file inside their directory are included, and
// each matching file is read to sum its candles' Volume so the result can
// be sorted by that year's total trading volume, descending (most-traded
// first; ties broken alphabetically by ticker). If year is zero, no file
// contents are read and the result is sorted alphabetically instead.
func ListAssetsFrom(dir string, year int) ([]string, error) {
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

	volumes, err := tickerVolumesFor(dir, entries, year)
	if err != nil {
		return nil, err
	}

	tickers := make([]string, len(volumes))
	for i, tv := range volumes {
		tickers[i] = tv.ticker
	}
	return tickers, nil
}

// tickerVolume pairs a ticker with its total traded volume for a single
// year, in money.Money's minor-unit (cents) integer amount.
type tickerVolume struct {
	ticker string
	volume int64
}

// tickerVolumesFor computes each directory entry's total year volume (for
// entries that are directories with a <TICKER>_<year>.json file inside),
// sorted by that volume descending (ties broken alphabetically by ticker).
// Shared by ListAssetsFrom and ListAssetsWithVolumeFrom so both read/sum
// each ticker's year file exactly once.
func tickerVolumesFor(dir string, entries []os.DirEntry, year int) ([]tickerVolume, error) {
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

	return volumes, nil
}

// AssetVolume pairs a ticker with its total traded volume for a single
// year, in major currency units (same convention as domain.Candle's JSON
// Volume field, e.g. domain.Candle.MarshalJSON's use of AsMajorUnits).
type AssetVolume struct {
	Ticker string
	Volume float64
}

// ListAssetsWithVolume returns tickers with imported data for year, paired
// with each ticker's total volume for that year, from DefaultCotahistDir.
// See ListAssetsWithVolumeFrom.
func ListAssetsWithVolume(year int) ([]AssetVolume, error) {
	return ListAssetsWithVolumeFrom(DefaultCotahistDir, year)
}

// ListAssetsWithVolumeFrom returns tickers with imported data under dir for
// year, in the same descending-volume order as ListAssetsFrom(dir, year)
// (ties broken alphabetically by ticker), paired with each ticker's total
// volume for that year in major currency units. year must be non-zero -
// volume has no meaning without a specific year to sum over, so year == 0
// is a caller error, not silently zero volumes.
func ListAssetsWithVolumeFrom(dir string, year int) ([]AssetVolume, error) {
	if year == 0 {
		return nil, fmt.Errorf("ListAssetsWithVolumeFrom: year must be non-zero")
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []AssetVolume{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	volumes, err := tickerVolumesFor(dir, entries, year)
	if err != nil {
		return nil, err
	}

	result := make([]AssetVolume, len(volumes))
	for i, tv := range volumes {
		result[i] = AssetVolume{
			Ticker: tv.ticker,
			Volume: money.New(tv.volume, domain.Currency).AsMajorUnits(),
		}
	}
	return result, nil
}

func readYearFile(dir, asset string, year int) ([]domain.Candle, error) {
	path := filepath.Join(dir, asset, fmt.Sprintf("%s_%d.json", asset, year))
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
