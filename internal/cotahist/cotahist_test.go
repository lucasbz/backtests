package cotahist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// testCandle builds a fully-populated domain.Candle for a given date; the
// price/volume amount is arbitrary since these tests only exercise
// date-range filtering and sorting.
func testCandle(date string, amount int64) domain.Candle {
	m := *money.New(amount, domain.Currency)
	return domain.Candle{Date: date, Open: m, High: m, Low: m, Avg: m, Close: m, Volume: m}
}

func writeCandleFile(t *testing.T, dir, ticker string, year int, candles []domain.Candle) {
	t.Helper()
	tickerDir := filepath.Join(dir, ticker)
	if err := os.MkdirAll(tickerDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := json.Marshal(candles)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(tickerDir, ticker+"_"+strconv.Itoa(year)+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestLoadCandlesFrom_FiltersAndSortsAcrossYears(t *testing.T) {
	dir := t.TempDir()

	writeCandleFile(t, dir, "ABCB4", 2010, []domain.Candle{
		testCandle("2010-06-15", 100),
		testCandle("2010-12-31", 110),
	})
	writeCandleFile(t, dir, "ABCB4", 2011, []domain.Candle{
		testCandle("2011-01-04", 120),
		testCandle("2011-06-01", 130),
	})

	from := time.Date(2010, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2011, 1, 31, 0, 0, 0, 0, time.UTC)

	got, err := LoadCandlesFrom(dir, "ABCB4", from, to)
	if err != nil {
		t.Fatalf("LoadCandlesFrom: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d candles, want 2: %+v", len(got), got)
	}
	if got[0].Date != "2010-12-31" || got[1].Date != "2011-01-04" {
		t.Errorf("dates = [%s, %s], want [2010-12-31, 2011-01-04]", got[0].Date, got[1].Date)
	}
}

func TestLoadCandlesFrom_MissingYearFileSkipped(t *testing.T) {
	dir := t.TempDir()

	writeCandleFile(t, dir, "ABCB4", 2010, []domain.Candle{
		testCandle("2010-06-15", 100),
	})

	from := time.Date(2009, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2011, 12, 31, 0, 0, 0, 0, time.UTC)

	got, err := LoadCandlesFrom(dir, "ABCB4", from, to)
	if err != nil {
		t.Fatalf("LoadCandlesFrom: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d candles, want 1: %+v", len(got), got)
	}
}

func TestDateRangeFrom_SpansEarliestToLatestYear(t *testing.T) {
	dir := t.TempDir()

	writeCandleFile(t, dir, "ABCB4", 2011, []domain.Candle{
		testCandle("2011-03-10", 100),
		testCandle("2011-09-20", 110),
	})
	writeCandleFile(t, dir, "ABCB4", 2010, []domain.Candle{
		testCandle("2010-01-04", 90),
		testCandle("2010-12-30", 95),
	})
	writeCandleFile(t, dir, "ABCB4", 2013, []domain.Candle{
		testCandle("2013-02-01", 120),
		testCandle("2013-11-15", 125),
	})

	earliest, latest, err := DateRangeFrom(dir, "ABCB4")
	if err != nil {
		t.Fatalf("DateRangeFrom: %v", err)
	}
	if earliest != "2010-01-04" {
		t.Errorf("earliest = %q, want 2010-01-04", earliest)
	}
	if latest != "2013-11-15" {
		t.Errorf("latest = %q, want 2013-11-15", latest)
	}
}

func TestDateRangeFrom_UnknownTicker(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := DateRangeFrom(dir, "NOPE4"); err == nil {
		t.Fatal("expected error for unknown ticker")
	}
}

func TestListAssetsFrom_NoFilterReturnsAllSorted(t *testing.T) {
	dir := t.TempDir()

	writeCandleFile(t, dir, "VALE3", 2010, []domain.Candle{testCandle("2010-06-15", 100)})
	writeCandleFile(t, dir, "ABCB4", 2011, []domain.Candle{testCandle("2011-01-04", 120)})
	writeCandleFile(t, dir, "PETR4", 2012, []domain.Candle{testCandle("2012-01-04", 130)})

	got, err := ListAssetsFrom(dir, 0)
	if err != nil {
		t.Fatalf("ListAssetsFrom: %v", err)
	}

	want := []string{"ABCB4", "PETR4", "VALE3"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func TestListAssetsFrom_YearFilter(t *testing.T) {
	dir := t.TempDir()

	// ABCB4 is excluded (no 2011 file); it also has a 2010 file that must
	// not be included in the sum for other tickers' 2010 totals.
	writeCandleFile(t, dir, "VALE3", 2010, []domain.Candle{testCandle("2010-06-15", 100)})
	writeCandleFile(t, dir, "ABCB4", 2011, []domain.Candle{testCandle("2011-01-04", 120)})
	writeCandleFile(t, dir, "PETR4", 2010, []domain.Candle{testCandle("2010-01-04", 130)})

	got, err := ListAssetsFrom(dir, 2010)
	if err != nil {
		t.Fatalf("ListAssetsFrom: %v", err)
	}

	want := []string{"PETR4", "VALE3"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func TestListAssetsFrom_YearFilterSortsByDescendingVolume(t *testing.T) {
	dir := t.TempDir()

	// Alphabetical order would be ABCB4, PETR4, VALE3; total 2010 volume
	// (sum of each ticker's candle amounts) is the opposite order, so this
	// only passes if the result is sorted by volume, not alphabetically.
	writeCandleFile(t, dir, "ABCB4", 2010, []domain.Candle{
		testCandle("2010-01-04", 10),
		testCandle("2010-06-15", 10),
	})
	writeCandleFile(t, dir, "PETR4", 2010, []domain.Candle{
		testCandle("2010-01-04", 500),
	})
	writeCandleFile(t, dir, "VALE3", 2010, []domain.Candle{
		testCandle("2010-01-04", 1000),
		testCandle("2010-06-15", 1000),
	})

	got, err := ListAssetsFrom(dir, 2010)
	if err != nil {
		t.Fatalf("ListAssetsFrom: %v", err)
	}

	want := []string{"VALE3", "PETR4", "ABCB4"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func TestListAssetsFrom_YearFilterTiesBrokenAlphabetically(t *testing.T) {
	dir := t.TempDir()

	writeCandleFile(t, dir, "VALE3", 2010, []domain.Candle{testCandle("2010-06-15", 100)})
	writeCandleFile(t, dir, "ABCB4", 2010, []domain.Candle{testCandle("2010-01-04", 100)})
	writeCandleFile(t, dir, "PETR4", 2010, []domain.Candle{testCandle("2010-01-04", 100)})

	got, err := ListAssetsFrom(dir, 2010)
	if err != nil {
		t.Fatalf("ListAssetsFrom: %v", err)
	}

	want := []string{"ABCB4", "PETR4", "VALE3"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func TestListAssetsFrom_YearFilterNoMatchesReturnsEmptyNotNil(t *testing.T) {
	dir := t.TempDir()

	writeCandleFile(t, dir, "VALE3", 2010, []domain.Candle{testCandle("2010-06-15", 100)})

	got, err := ListAssetsFrom(dir, 2099)
	if err != nil {
		t.Fatalf("ListAssetsFrom: %v", err)
	}
	if got == nil {
		t.Fatal("got nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestListAssetsFrom_MissingDirReturnsEmptyNotError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")

	got, err := ListAssetsFrom(dir, 0)
	if err != nil {
		t.Fatalf("ListAssetsFrom: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}
