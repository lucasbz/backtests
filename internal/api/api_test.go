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
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
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
	rec := doRequest(t, http.MethodGet, "/api/info?asset=PETR4", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp infoResponse
	decodeJSON(t, rec, &resp)
	if resp.Asset != "PETR4" {
		t.Errorf("Asset = %q, want %q", resp.Asset, "PETR4")
	}
	if resp.Earliest == "" || resp.Latest == "" {
		t.Errorf("Earliest/Latest should not be empty, got %+v", resp)
	}
}

func TestHandleInfo_MissingAsset(t *testing.T) {
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

// TestHandleInfo_PathTraversalAssetRejected proves a path-traversal-shaped
// asset value is rejected with a 400 by validateAsset, before it ever
// reaches cotahist.DateRange to be joined into a filesystem path (see
// docs/plans/code-review-findings.md, finding #1).
func TestHandleInfo_PathTraversalAssetRejected(t *testing.T) {
	chdirToRepoRoot(t)
	for _, asset := range []string{
		"../../etc/passwd",
		"..%2F..%2Fetc",
		"PETR4/../../etc",
		"foo/bar",
		"petr4", // lowercase also rejected - allowlist is strict uppercase+digits
	} {
		rec := doRequest(t, http.MethodGet, "/api/info?asset="+asset, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("asset=%q: status = %d, want %d, body = %s", asset, rec.Code, http.StatusBadRequest, rec.Body.String())
			continue
		}

		var resp errorResponse
		decodeJSON(t, rec, &resp)
		if resp.Error == "" {
			t.Errorf("asset=%q: expected a non-empty error message", asset)
		}
	}
}

func TestHandleInfo_UnknownAsset(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/info?asset=DOESNOTEXIST9", nil)
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

	var infos []strategies.StrategyInfo
	decodeJSON(t, rec, &infos)

	byName := map[string]strategies.StrategyInfo{}
	for _, info := range infos {
		byName[info.Name] = info
	}

	buyAndHold, ok := byName["buy-and-hold"]
	if !ok {
		t.Fatalf("strategies = %+v, want it to contain %q", infos, "buy-and-hold")
	}
	if buyAndHold.Params == nil || len(buyAndHold.Params) != 0 {
		t.Errorf("buy-and-hold.Params = %+v, want an empty (non-nil) slice", buyAndHold.Params)
	}

	smaCrossover, ok := byName["sma-crossover"]
	if !ok {
		t.Fatalf("strategies = %+v, want it to contain %q", infos, "sma-crossover")
	}
	if len(smaCrossover.Params) != 2 {
		t.Fatalf("sma-crossover.Params = %+v, want 2 entries", smaCrossover.Params)
	}
	if smaCrossover.Params[0].Key != "shortPeriod" || smaCrossover.Params[1].Key != "longPeriod" {
		t.Errorf("sma-crossover.Params = %+v, want keys shortPeriod, longPeriod in order", smaCrossover.Params)
	}
}

func TestHandleStopLosses(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/stop-losses", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var names []string
	decodeJSON(t, rec, &names)
	found := false
	for _, name := range names {
		if name == "percent" {
			found = true
		}
	}
	if !found {
		t.Errorf("stop-losses = %v, want it to contain %q", names, "percent")
	}
}

func TestHandleBacktest_Valid(t *testing.T) {
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00"}`)
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
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00","verbose":true}`)
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
	if _, ok := op["days"].(float64); !ok {
		t.Errorf("operation.days = %v (%T), want a plain number", op["days"], op["days"])
	}
}

func TestHandleBacktest_MissingFields(t *testing.T) {
	rec := doRequest(t, http.MethodPost, "/api/backtest", []byte(`{"asset":"PETR4"}`))
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
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"not-a-number"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleBacktest_ZeroBalance(t *testing.T) {
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"0"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestHandleBacktest_PathTraversalAssetRejected mirrors
// TestHandleInfo_PathTraversalAssetRejected for POST /api/backtest: a
// path-traversal-shaped asset must be rejected with a 400 before it reaches
// cotahist.LoadCandles.
func TestHandleBacktest_PathTraversalAssetRejected(t *testing.T) {
	for _, asset := range []string{"../../etc/passwd", "PETR4/../../etc", "foo/bar"} {
		body := []byte(`{"asset":"` + asset + `","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00"}`)
		rec := doRequest(t, http.MethodPost, "/api/backtest", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("asset=%q: status = %d, want %d, body = %s", asset, rec.Code, http.StatusBadRequest, rec.Body.String())
			continue
		}

		var resp errorResponse
		decodeJSON(t, rec, &resp)
		if resp.Error == "" {
			t.Errorf("asset=%q: expected a non-empty error message", asset)
		}
	}
}

func TestHandleBacktest_UnknownStrategy(t *testing.T) {
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"does-not-exist","balance":"10000.00"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestHandleBacktest_StrategyParams asserts strategyParams is threaded
// through to strategies.LoadStrategy: sma-crossover requires
// shortPeriod/longPeriod (see internal/strategies/crossover.go), so
// omitting strategyParams entirely must fail, while supplying it must
// succeed.
func TestHandleBacktest_StrategyParams(t *testing.T) {
	chdirToRepoRoot(t)
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"sma-crossover","balance":"10000.00","strategyParams":{"shortPeriod":5,"longPeriod":20},"verbose":true}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["strategyName"] != "SMA Crossover" {
		t.Errorf("strategyName = %v, want %q", resp["strategyName"], "SMA Crossover")
	}
}

func TestHandleBacktest_StrategyParams_MissingRequiredParam(t *testing.T) {
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"sma-crossover","balance":"10000.00"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleBacktest_WithStopLoss(t *testing.T) {
	chdirToRepoRoot(t)
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"two-candle-breakout","balance":"10000.00","stopLoss":{"type":"percent","value":5},"verbose":true}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if _, ok := resp["totalOperations"].(float64); !ok {
		t.Fatalf("totalOperations missing or not a number, got %v", resp["totalOperations"])
	}
}

func TestHandleBacktest_UnknownStopLossType(t *testing.T) {
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00","stopLoss":{"type":"does-not-exist","value":5}}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleBacktest_MissingStopLossType(t *testing.T) {
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00","stopLoss":{"value":5}}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleBacktest_NonPositiveStopLossValue(t *testing.T) {
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00","stopLoss":{"type":"percent","value":0}}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleBacktest_NoStopLossOmitsIt(t *testing.T) {
	chdirToRepoRoot(t)
	body := []byte(`{"asset":"PETR4","start":"2015-01-02","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// allAssets flattens an assetsResponse's stocks/others back into a single
// slice, preserving stocks-then-others order, for tests that don't care
// about the grouping itself.
func allAssets(resp assetsResponse) []string {
	tickers := make([]string, 0, len(resp.Stocks)+len(resp.Others))
	tickers = append(tickers, resp.Stocks...)
	tickers = append(tickers, resp.Others...)
	return tickers
}

func TestHandleAssets_NoFilter(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/assets", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp assetsResponse
	decodeJSON(t, rec, &resp)
	found := false
	for _, ticker := range allAssets(resp) {
		if ticker == "PETR4" {
			found = true
		}
	}
	if !found {
		t.Errorf("tickers = %+v, want it to contain %q", resp, "PETR4")
	}
}

// TestHandleAssets_NoFilterGroupsStocksAndOthers checks the grouping is
// self-consistent with the loaded asset registry (see cotahist.LoadAssets):
// every ticker in "stocks" is registered with Type stock, and every ticker
// in "others" is either registered with some other Type or missing from
// the registry entirely (e.g. imported before assets.json existed).
func TestHandleAssets_NoFilterGroupsStocksAndOthers(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/assets", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp assetsResponse
	decodeJSON(t, rec, &resp)

	registry, err := cotahist.LoadAssetsFrom(cotahist.DefaultCotahistDir)
	if err != nil {
		t.Fatalf("LoadAssetsFrom: %v", err)
	}

	for _, ticker := range resp.Stocks {
		if registry[ticker].Type != domain.Stock {
			t.Errorf("stocks contains %q, whose registry Type is %q, not %q", ticker, registry[ticker].Type, domain.Stock)
		}
	}
	for _, ticker := range resp.Others {
		if registry[ticker].Type == domain.Stock {
			t.Errorf("others contains %q, whose registry Type is %q", ticker, domain.Stock)
		}
	}
	if !sort.StringsAreSorted(resp.Stocks) {
		t.Errorf("stocks = %v, want alphabetically sorted", resp.Stocks)
	}
	if !sort.StringsAreSorted(resp.Others) {
		t.Errorf("others = %v, want alphabetically sorted", resp.Others)
	}
}

func TestHandleAssets_YearFilter(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/assets?year=2015", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp assetsResponse
	decodeJSON(t, rec, &resp)
	found := false
	for _, ticker := range allAssets(resp) {
		if ticker == "PETR4" {
			found = true
		}
	}
	if !found {
		t.Errorf("tickers = %+v, want it to contain %q", resp, "PETR4")
	}
}

// TestHandleAssets_YearFilterSortsByDescendingVolume checks the response
// against real imported data: with a year filter, tickers must come back in
// descending order of that year's total trading volume within each group,
// not alphabetically.
func TestHandleAssets_YearFilterSortsByDescendingVolume(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/assets?year=2015", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp assetsResponse
	decodeJSON(t, rec, &resp)
	tickers := resp.Stocks
	if len(tickers) < 2 {
		t.Fatalf("expected at least 2 stock tickers for year 2015, got %v", tickers)
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

func TestHandleAssets_YearFilterNoMatchesReturnsEmptyArrays(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/assets?year=1900", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp assetsResponse
	decodeJSON(t, rec, &resp)
	if resp.Stocks == nil || len(resp.Stocks) != 0 {
		t.Errorf("stocks = %v, want empty non-nil slice", resp.Stocks)
	}
	if resp.Others == nil || len(resp.Others) != 0 {
		t.Errorf("others = %v, want empty non-nil slice", resp.Others)
	}
}

func TestHandleAssets_InvalidYear(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/assets?year=not-a-year", nil)
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
	body := []byte(`{"asset":"PETR4","start":"not-a-date","end":"2015-12-30","strategy":"buy-and-hold","balance":"10000.00"}`)
	rec := doRequest(t, http.MethodPost, "/api/backtest", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleCandles_Valid(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/candles?asset=PETR4&start=2015-01-02&end=2015-01-30", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var candles []domain.Candle
	decodeJSON(t, rec, &candles)
	if len(candles) == 0 {
		t.Fatal("expected at least one candle")
	}
	for _, c := range candles {
		if c.Date < "2015-01-02" || c.Date > "2015-01-30" {
			t.Errorf("candle date %q outside requested range", c.Date)
		}
	}
}

func TestHandleCandles_MissingParams(t *testing.T) {
	for _, target := range []string{
		"/api/candles",
		"/api/candles?asset=PETR4",
		"/api/candles?asset=PETR4&start=2015-01-02",
		"/api/candles?start=2015-01-02&end=2015-01-30",
	} {
		rec := doRequest(t, http.MethodGet, target, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", target, rec.Code, http.StatusBadRequest)
		}
	}
}

// TestHandleCandles_PathTraversalAssetRejected mirrors
// TestHandleInfo_PathTraversalAssetRejected - see that test's comment.
func TestHandleCandles_PathTraversalAssetRejected(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/candles?asset=..%2F..%2Fetc&start=2015-01-02&end=2015-01-30", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleCandles_InvalidDate(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/candles?asset=PETR4&start=not-a-date&end=2015-01-30", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleCandles_EndBeforeStart(t *testing.T) {
	rec := doRequest(t, http.MethodGet, "/api/candles?asset=PETR4&start=2015-01-30&end=2015-01-02", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	decodeJSON(t, rec, &resp)
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleCandles_NoDataReturnsEmptyArray(t *testing.T) {
	chdirToRepoRoot(t)
	rec := doRequest(t, http.MethodGet, "/api/candles?asset=ZZZZ9&start=2015-01-02&end=2015-01-30", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var candles []domain.Candle
	decodeJSON(t, rec, &candles)
	if len(candles) != 0 {
		t.Errorf("candles = %v, want empty", candles)
	}
}
