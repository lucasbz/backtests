// Package api exposes the same operations as the backtest/info CLI
// subcommands (cmd/main.go) as a JSON HTTP API. See openapi.yaml at the
// repo root for the documented contract.
package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/indicators"
	"github.com/lucasbz/backtests/internal/stoploss"
	"github.com/lucasbz/backtests/internal/strategies"
)

// assetPattern is the strict allowlist enforced on every request's
// asset/ticker string before it reaches internal/cotahist: uppercase
// letters and digits only, matching B3's own ticker format (e.g. "PETR4").
// internal/cotahist.LoadCandlesFrom/DateRangeFrom build filesystem paths by
// joining this string directly (filepath.Join(dir, asset, ...)) with no
// sanitization of their own, so without this check here, at the API
// boundary, a caller could pass an asset containing "/" or ".." segments
// and traverse outside resources/cotahist (see docs/plans/
// code-review-findings.md, finding #1).
var assetPattern = regexp.MustCompile(`^[A-Z0-9]+$`)

// validateAsset rejects any asset/ticker value that doesn't match
// assetPattern, before it's used to build a filesystem path.
func validateAsset(asset string) error {
	if !assetPattern.MatchString(asset) {
		return fmt.Errorf("asset must match %s (uppercase letters and digits only), got %q", assetPattern.String(), asset)
	}
	return nil
}

// NewHandler builds the HTTP handler for the API: GET /api/info,
// POST /api/backtest, GET /api/strategies, GET /api/stop-losses,
// GET /api/assets, GET /api/candles, GET /api/indicators/sma,
// GET /api/indicators/ema.
func NewHandler() http.Handler {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/api/info", handleInfo)
	engine.POST("/api/backtest", handleBacktest)
	engine.GET("/api/strategies", handleStrategies)
	engine.GET("/api/stop-losses", handleStopLosses)
	engine.GET("/api/assets", handleAssets)
	engine.GET("/api/candles", handleCandles)
	engine.GET("/api/indicators/sma", handleIndicator("sma"))
	engine.GET("/api/indicators/ema", handleIndicator("ema"))

	return engine
}

// errorResponse is the JSON shape every handler returns on failure:
// {"error": "message"}.
type errorResponse struct {
	Error string `json:"error"`
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, errorResponse{Error: message})
}

// infoResponse is the JSON shape for GET /api/info.
type infoResponse struct {
	Asset    string `json:"asset"`
	Earliest string `json:"earliest"`
	Latest   string `json:"latest"`
}

func handleInfo(c *gin.Context) {
	asset := c.Query("asset")
	if asset == "" {
		writeError(c, http.StatusBadRequest, "asset is required")
		return
	}
	if err := validateAsset(asset); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	earliest, latest, err := cotahist.DateRange(asset)
	if err != nil {
		// cotahist.DateRange's only failure mode is "no data found for this
		// asset (not imported yet?)", which is a 404, not a server error.
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, infoResponse{
		Asset:    asset,
		Earliest: earliest,
		Latest:   latest,
	})
}

func handleStrategies(c *gin.Context) {
	c.JSON(http.StatusOK, strategies.AvailableStrategyInfo())
}

// handleStopLosses lists the type names accepted by stopLoss.type on
// POST /api/backtest - a sibling to GET /api/strategies.
func handleStopLosses(c *gin.Context) {
	c.JSON(http.StatusOK, stoploss.AvailableStopLossNamesList())
}

// assetsResponse is the JSON shape for GET /api/assets: tickers split into
// "stocks" (common equities, per the loaded asset registry's Type) and
// "others" (everything else - units, ETFs, BDRs, and any ticker missing
// from the registry, e.g. because it was imported before assets.json
// existed and the importer hasn't been re-run since). Ordering within each
// group is preserved from cotahist.ListAssets (alphabetical, or by
// descending year volume when `year` is given).
type assetsResponse struct {
	Stocks []string `json:"stocks"`
	Others []string `json:"others"`
}

func handleAssets(c *gin.Context) {
	year := 0
	if raw := c.Query("year"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "year must be a valid integer")
			return
		}
		year = parsed
	}

	tickers, err := cotahist.ListAssets(year)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	registry, err := cotahist.LoadAssets()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp := assetsResponse{Stocks: []string{}, Others: []string{}}
	for _, ticker := range tickers {
		if asset, ok := registry[ticker]; ok && asset.Type == domain.Stock {
			resp.Stocks = append(resp.Stocks, ticker)
		} else {
			resp.Others = append(resp.Others, ticker)
		}
	}

	c.JSON(http.StatusOK, resp)
}

// stopLossRequest is the optional stopLoss object on backtestRequest: a
// type name (see GET /api/stop-losses for the accepted values) plus a
// value whose meaning depends on that type (see
// internal/stoploss.availableStopLosses).
type stopLossRequest struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

// backtestRequest is the JSON body for POST /api/backtest.
type backtestRequest struct {
	Asset          string             `json:"asset"`
	Start          string             `json:"start"`
	End            string             `json:"end"`
	Strategy       string             `json:"strategy"`
	Balance        string             `json:"balance"`
	Verbose        bool               `json:"verbose"`
	StopLoss       *stopLossRequest   `json:"stopLoss"`
	StrategyParams map[string]float64 `json:"strategyParams"`
}

func handleBacktest(c *gin.Context) {
	var req backtestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if req.Asset == "" || req.Start == "" || req.End == "" || req.Strategy == "" || req.Balance == "" {
		writeError(c, http.StatusBadRequest, "asset, start, end, strategy and balance are all required")
		return
	}
	if err := validateAsset(req.Asset); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	startingBalance, err := domain.ParseMoney(req.Balance)
	if err != nil {
		writeError(c, http.StatusBadRequest, "parsing balance: "+err.Error())
		return
	}
	if !startingBalance.IsPositive() {
		writeError(c, http.StatusBadRequest, "balance must be greater than zero")
		return
	}

	newStrategy, err := strategies.LoadStrategy(req.Strategy, req.StrategyParams)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	// stopLoss is optional: omitted (nil) means no stop-loss, exactly the
	// backtest's behavior before stop-losses existed.
	var newStopLoss domain.StopLoss
	if req.StopLoss != nil {
		if req.StopLoss.Type == "" {
			writeError(c, http.StatusBadRequest, "stopLoss.type is required when stopLoss is present")
			return
		}
		newStopLoss, err = stoploss.LoadStopLoss(req.StopLoss.Type, req.StopLoss.Value)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	startDate, err := time.Parse("2006-01-02", req.Start)
	if err != nil {
		writeError(c, http.StatusBadRequest, "parsing start: "+err.Error())
		return
	}
	endDate, err := time.Parse("2006-01-02", req.End)
	if err != nil {
		writeError(c, http.StatusBadRequest, "parsing end: "+err.Error())
		return
	}

	bt := &backtest.Backtest{
		Asset:    req.Asset,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: newStrategy,
		StopLoss: newStopLoss,
	}

	result, err := bt.Run()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if !req.Verbose {
		result.Operations = nil
	}

	c.JSON(http.StatusOK, result)
}

// handleCandles serves an asset's raw daily candles for the requested
// [start, end] range, for charting (e.g. a candlestick view) - unlike
// handleBacktest, it runs no strategy and returns no derived result, just
// the underlying price data. domain.Candle already marshals to a
// charting-friendly plain-decimal JSON shape (see candleJSON), so the
// response is the []domain.Candle slice as-is.
func handleCandles(c *gin.Context) {
	asset := c.Query("asset")
	start := c.Query("start")
	end := c.Query("end")
	if asset == "" || start == "" || end == "" {
		writeError(c, http.StatusBadRequest, "asset, start and end are all required")
		return
	}
	if err := validateAsset(asset); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	startDate, err := time.Parse("2006-01-02", start)
	if err != nil {
		writeError(c, http.StatusBadRequest, "parsing start: "+err.Error())
		return
	}
	endDate, err := time.Parse("2006-01-02", end)
	if err != nil {
		writeError(c, http.StatusBadRequest, "parsing end: "+err.Error())
		return
	}
	if endDate.Before(startDate) {
		writeError(c, http.StatusBadRequest, "end must not be before start")
		return
	}

	candles, err := cotahist.LoadCandles(asset, startDate, endDate)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if candles == nil {
		candles = []domain.Candle{}
	}

	c.JSON(http.StatusOK, candles)
}

// indicatorPoint is the JSON shape of one element in the array returned by
// GET /api/indicators/sma and GET /api/indicators/ema.
type indicatorPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// handleIndicator builds a handler for GET /api/indicators/{indicatorType}
// (indicatorType is "sma" or "ema" - see NewHandler's route registrations),
// serving a bare JSON array of {date, value} points computed by running
// indicators.LoadIndicator(indicatorType, ...) forward over the asset's
// candles in the requested [start, end] range - separate from, and much
// simpler than, the backtest/strategy machinery in handleBacktest: no
// decisions, no positions, just the indicator's raw values for charting.
//
// asset/start/end validation mirrors handleCandles exactly. The warm-up
// prefix (candles seen before the indicator has enough history to produce
// a value, e.g. the first Period-1 candles of an SMA) is simply omitted
// from the response, not sent as null/0 - see indicators.Indicator.Value's
// ready bool.
func handleIndicator(indicatorType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		asset := c.Query("asset")
		start := c.Query("start")
		end := c.Query("end")
		periodRaw := c.Query("period")
		if asset == "" || start == "" || end == "" || periodRaw == "" {
			writeError(c, http.StatusBadRequest, "asset, start, end and period are all required")
			return
		}
		if err := validateAsset(asset); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		startDate, err := time.Parse("2006-01-02", start)
		if err != nil {
			writeError(c, http.StatusBadRequest, "parsing start: "+err.Error())
			return
		}
		endDate, err := time.Parse("2006-01-02", end)
		if err != nil {
			writeError(c, http.StatusBadRequest, "parsing end: "+err.Error())
			return
		}
		if endDate.Before(startDate) {
			writeError(c, http.StatusBadRequest, "end must not be before start")
			return
		}

		period, err := strconv.Atoi(periodRaw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "period must be a valid integer")
			return
		}
		if period <= 0 {
			writeError(c, http.StatusBadRequest, "period must be a positive integer")
			return
		}

		candles, err := cotahist.LoadCandles(asset, startDate, endDate)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			return
		}

		// Can't actually fail here given the validation above (period is a
		// positive integer, indicatorType is always "sma" or "ema" - both
		// registered in internal/indicators), but LoadIndicator can fail in
		// general (unknown type, invalid params), so handle it for
		// robustness/symmetry with its contract rather than assuming.
		ind, err := indicators.LoadIndicator(indicatorType, map[string]float64{"period": float64(period)})
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		// candles is already sorted chronologically by LoadCandles; walk it
		// in order, updating the indicator before reading Value() (same
		// "update-then-read" ordering strategies.crossoverStrategy.Decide
		// uses), collecting a point for every candle where it's ready.
		points := []indicatorPoint{}
		for _, candle := range candles {
			ind.Update(candle)
			if value, ready := ind.Value(); ready {
				points = append(points, indicatorPoint{Date: candle.Date, Value: value})
			}
		}

		c.JSON(http.StatusOK, points)
	}
}
