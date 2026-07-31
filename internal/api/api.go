// Package api exposes the same operations as the backtest/info CLI
// subcommands (cmd/main.go) as a JSON HTTP API. See openapi.yaml at the
// repo root for the documented contract.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lucasbz/backtests/internal/backtest"
	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
)

// NewHandler builds the HTTP handler for the API: GET /api/info,
// POST /api/backtest, GET /api/strategies, GET /api/tickers.
func NewHandler() http.Handler {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/api/info", handleInfo)
	engine.POST("/api/backtest", handleBacktest)
	engine.GET("/api/strategies", handleStrategies)
	engine.GET("/api/tickers", handleTickers)

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
	Ticker   string `json:"ticker"`
	Earliest string `json:"earliest"`
	Latest   string `json:"latest"`
}

func handleInfo(c *gin.Context) {
	ticker := c.Query("ticker")
	if ticker == "" {
		writeError(c, http.StatusBadRequest, "ticker is required")
		return
	}

	earliest, latest, err := cotahist.DateRange(ticker)
	if err != nil {
		// cotahist.DateRange's only failure mode is "no data found for this
		// ticker (not imported yet?)", which is a 404, not a server error.
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, infoResponse{
		Ticker:   ticker,
		Earliest: earliest,
		Latest:   latest,
	})
}

func handleStrategies(c *gin.Context) {
	c.JSON(http.StatusOK, strategies.AvailableStrategyNamesList())
}

// tickersResponse is the JSON shape for GET /api/tickers: tickers split
// into "stocks" (common equities) and "others" (units, ETFs, FIIs, BDRs),
// per cotahist.IsStock. Ordering within each group is preserved from
// cotahist.ListTickers (alphabetical, or by descending year volume when
// `year` is given).
type tickersResponse struct {
	Stocks []string `json:"stocks"`
	Others []string `json:"others"`
}

func handleTickers(c *gin.Context) {
	year := 0
	if raw := c.Query("year"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, "year must be a valid integer")
			return
		}
		year = parsed
	}

	tickers, err := cotahist.ListTickers(year)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp := tickersResponse{Stocks: []string{}, Others: []string{}}
	for _, ticker := range tickers {
		if cotahist.IsStock(ticker) {
			resp.Stocks = append(resp.Stocks, ticker)
		} else {
			resp.Others = append(resp.Others, ticker)
		}
	}

	c.JSON(http.StatusOK, resp)
}

// backtestRequest is the JSON body for POST /api/backtest.
type backtestRequest struct {
	Ticker   string `json:"ticker"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Strategy string `json:"strategy"`
	Balance  string `json:"balance"`
	Verbose  bool   `json:"verbose"`
}

func handleBacktest(c *gin.Context) {
	var req backtestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if req.Ticker == "" || req.Start == "" || req.End == "" || req.Strategy == "" || req.Balance == "" {
		writeError(c, http.StatusBadRequest, "ticker, start, end, strategy and balance are all required")
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

	newStrategy, err := strategies.LoadStrategy(req.Strategy, startingBalance)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
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
		Ticker:   req.Ticker,
		Start:    startDate,
		End:      endDate,
		Balance:  startingBalance,
		Strategy: newStrategy,
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
