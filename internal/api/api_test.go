package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/lucasbz/backtests/internal/cotahist"
)

func doRequest(t *testing.T, method, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	NewHandler().ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decoding response %q: %v", rec.Body.String(), err)
	}
}

// chdirToRepoRoot points the test's working directory at the repo root, so
// cotahist.DateRange/LoadCandles (which read from the fixed relative path
// resources/cotahist) can find the real imported data checked into the repo.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	t.Chdir("../..")
}

func TestHandleInfo_Valid(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/info?ticker=PETR4", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp infoResponse
	decodeJSON(t, rec, &resp)
	if resp.Ticker != "PETR4" {
		t.Errorf("Ticker = %q, want %q", resp.Ticker, "PETR4")
	}
	if resp.Earliest == "" || resp.Latest == "" {
		t.Errorf("Earliest/Latest should not be empty, got %+v", resp)
	}
}

func TestHandleInfo_MissingTicker(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/info", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleInfo_UnknownTicker(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/info?ticker=DOESNOTEXIST9", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleStrategies(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/strategies", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var names []string
	decodeJSON(t, rec, &names)
	found := false
	for _, name := range names {
		if name == "buy-and-hold" {
			found = true
		}
	}
	if !found {
		t.Errorf("strategies = %v, want it to contain %q", names, "buy-and-hold")
	}
}

func TestHandleBacktest_Valid(t *testing.T) {
	body := []byte(`{"ticker":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["strategyName"] != "Buy & Hold" {
		t.Errorf("strategyName = %v, want %q", resp["strategyName"], "Buy & Hold")
	}
	if _, ok := resp["operations"]; ok {
		t.Errorf("operations should be omitted when verbose is false, got %v", resp["operations"])
	}
	// Note: this test doesn't chdirToRepoRoot, so cotahist.LoadCandles finds
	// no data for PETR4 here (same as the "unknown ticker" case documented
	// in openapi.yaml) and the strategy produces zero operations.
	totalOperations, ok := resp["totalOperations"].(float64)
	if !ok {
		t.Fatalf("totalOperations missing or not a number, got %v", resp["totalOperations"])
	}
	if totalOperations != 0 {
		t.Errorf("totalOperations = %v, want 0", totalOperations)
	}
}

func TestHandleBacktest_Verbose(t *testing.T) {
	chdirToRepoRoot(t)
	body := []byte(`{"ticker":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00","verbose":true}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	ops, ok := resp["operations"].([]any)
	if !ok || len(ops) == 0 {
		t.Fatalf("expected a non-empty operations array, got %v", resp["operations"])
	}
	if totalOperations, ok := resp["totalOperations"].(float64); !ok || int(totalOperations) != len(ops) {
		t.Errorf("totalOperations = %v, want %d (len(operations))", resp["totalOperations"], len(ops))
	}

	op, ok := ops[0].(map[string]any)
	if !ok {
		t.Fatalf("operation is not an object: %v", ops[0])
	}
	buyOrder, ok := op["buyOrder"].(map[string]any)
	if !ok {
		t.Fatalf("buyOrder is not an object: %v", op["buyOrder"])
	}
	if _, ok := buyOrder["price"].(float64); !ok {
		t.Errorf("buyOrder.price = %v (%T), want a plain number", buyOrder["price"], buyOrder["price"])
	}
	if _, ok := buyOrder["orderType"]; ok {
		t.Errorf("buyOrder.orderType should be omitted, got %v", buyOrder["orderType"])
	}
}

func TestHandleBacktest_MissingFields(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/backtest", []byte(`{"ticker":"PETR4"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleBacktest_InvalidJSON(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/backtest", []byte(`not json`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleBacktest_InvalidBalance(t *testing.T) {
	body := []byte(`{"ticker":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"not-a-number"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleBacktest_ZeroBalance(t *testing.T) {
	body := []byte(`{"ticker":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"0"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleBacktest_UnknownStrategy(t *testing.T) {
	body := []byte(`{"ticker":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"does-not-exist","balance":"10000.00"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleTickers_NoFilter(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/tickers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var tickers []string
	decodeJSON(t, rec, &tickers)
	found := false
	for _, ticker := range tickers {
		if ticker == "PETR4" {
			found = true
		}
	}
	if !found {
		t.Errorf("tickers = %v, want it to contain %q", tickers, "PETR4")
	}
}

func TestHandleTickers_YearFilter(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/tickers?year=2015", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var tickers []string
	decodeJSON(t, rec, &tickers)
	found := false
	for _, ticker := range tickers {
		if ticker == "PETR4" {
			found = true
		}
	}
	if !found {
		t.Errorf("tickers = %v, want it to contain %q", tickers, "PETR4")
	}
}

// TestHandleTickers_YearFilterSortsByDescendingVolume checks the response
// against real imported data: with a year filter, tickers must come back in
// descending order of that year's total trading volume, not alphabetically.
func TestHandleTickers_YearFilterSortsByDescendingVolume(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/tickers?year=2015", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var tickers []string
	decodeJSON(t, rec, &tickers)
	if len(tickers) < 2 {
		t.Fatalf("expected at least 2 tickers for year 2015, got %v", tickers)
	}

	from := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2015, 12, 31, 0, 0, 0, 0, time.UTC)

	volumeOf := func(ticker string) int64 {
		t.Helper()
		candles, err := cotahist.LoadCandles(ticker, from, to)
		if err != nil {
			t.Fatalf("LoadCandles(%q): %v", ticker, err)
		}
		var total int64
		for _, c := range candles {
			total += c.Volume.Amount()
		}
		return total
	}

	prevVolume := int64(-1)
	for _, ticker := range tickers {
		volume := volumeOf(ticker)
		if prevVolume != -1 && volume > prevVolume {
			t.Errorf("tickers not sorted by descending volume: %q (volume %d) comes after a ticker with volume %d", ticker, volume, prevVolume)
		}
		prevVolume = volume
	}

	// Sanity check: the response isn't just alphabetical order either
	// (guards against a no-op sort that happens to look right above).
	alphabetical := make([]string, len(tickers))
	copy(alphabetical, tickers)
	sort.Strings(alphabetical)
	if reflect.DeepEqual(tickers, alphabetical) {
		t.Errorf("tickers = %v looks alphabetically sorted, want descending-volume order for year filter", tickers)
	}
}

func TestHandleTickers_YearFilterNoMatchesReturnsEmptyArray(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/tickers?year=1900", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "[]" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "[]")
	}
}

func TestHandleTickers_InvalidYear(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/tickers?year=not-a-year", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleBacktest_InvalidDate(t *testing.T) {
	body := []byte(`{"ticker":"PETR4","start":"not-a-date","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
