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

// testQuote builds a fully-populated domain.Quote for a given date; the
// price/volume amount is arbitrary since these tests only exercise
// date-range filtering and sorting.
func testQuote(date string, amount int64) domain.Quote {
	m := *money.New(amount, domain.Currency)
	return domain.Quote{Date: date, Open: m, High: m, Low: m, Avg: m, Close: m, Volume: m}
}

func writeQuoteFile(t *testing.T, dir, ticker string, year int, quotes []domain.Quote) {
	t.Helper()
	tickerDir := filepath.Join(dir, ticker)
	if err := os.MkdirAll(tickerDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := json.Marshal(quotes)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(tickerDir, ticker+"_"+strconv.Itoa(year)+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestLoadQuotesFrom_FiltersAndSortsAcrossYears(t *testing.T) {
	dir := t.TempDir()

	writeQuoteFile(t, dir, "ABCB4", 2010, []domain.Quote{
		testQuote("2010-06-15", 100),
		testQuote("2010-12-31", 110),
	})
	writeQuoteFile(t, dir, "ABCB4", 2011, []domain.Quote{
		testQuote("2011-01-04", 120),
		testQuote("2011-06-01", 130),
	})

	from := time.Date(2010, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2011, 1, 31, 0, 0, 0, 0, time.UTC)

	got, err := LoadQuotesFrom(dir, "ABCB4", from, to)
	if err != nil {
		t.Fatalf("LoadQuotesFrom: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d quotes, want 2: %+v", len(got), got)
	}
	if got[0].Date != "2010-12-31" || got[1].Date != "2011-01-04" {
		t.Errorf("dates = [%s, %s], want [2010-12-31, 2011-01-04]", got[0].Date, got[1].Date)
	}
}

func TestLoadQuotesFrom_MissingYearFileSkipped(t *testing.T) {
	dir := t.TempDir()

	writeQuoteFile(t, dir, "ABCB4", 2010, []domain.Quote{
		testQuote("2010-06-15", 100),
	})

	from := time.Date(2009, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2011, 12, 31, 0, 0, 0, 0, time.UTC)

	got, err := LoadQuotesFrom(dir, "ABCB4", from, to)
	if err != nil {
		t.Fatalf("LoadQuotesFrom: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d quotes, want 1: %+v", len(got), got)
	}
}

func TestDateRangeFrom_SpansEarliestToLatestYear(t *testing.T) {
	dir := t.TempDir()

	writeQuoteFile(t, dir, "ABCB4", 2011, []domain.Quote{
		testQuote("2011-03-10", 100),
		testQuote("2011-09-20", 110),
	})
	writeQuoteFile(t, dir, "ABCB4", 2010, []domain.Quote{
		testQuote("2010-01-04", 90),
		testQuote("2010-12-30", 95),
	})
	writeQuoteFile(t, dir, "ABCB4", 2013, []domain.Quote{
		testQuote("2013-02-01", 120),
		testQuote("2013-11-15", 125),
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
