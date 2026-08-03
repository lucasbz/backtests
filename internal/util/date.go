package util

import (
	"fmt"
	"time"
)

// dateLayout is the plain YYYY-MM-DD format every start/end date input
// uses, whether it arrives as a CLI flag or a JSON request field.
const dateLayout = "2006-01-02"

// ParseDateRange parses startRaw/endRaw (YYYY-MM-DD) and validates end
// isn't before start - the shared timeframe-validation rule every backtest/
// candle/indicator entry point (CLI flags, the JSON API) applies to its
// start/end input, previously duplicated at each call site with its own
// copy of the two time.Parse calls and the range check.
func ParseDateRange(startRaw, endRaw string) (start, end time.Time, err error) {
	start, err = time.Parse(dateLayout, startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing start: %w", err)
	}
	end, err = time.Parse(dateLayout, endRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing end: %w", err)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end must not be before start")
	}
	return start, end, nil
}
