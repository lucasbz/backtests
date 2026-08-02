package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// configFlagUsage is shared by the -config flag on both runBacktest and
// runCompare's flag sets.
const configFlagUsage = "path to a JSON config file providing defaults for -asset, -start, -end, -balance, -strategy, -v and strategyParams (flags passed explicitly always override the config file; strategyParams has no flag equivalent - it's config-file-only)"

// config mirrors the -config JSON file's shape: optional defaults for the
// backtest/compare flags. Any field a user doesn't want to default from the
// file can simply be omitted; the flags themselves remain the primary
// interface and always take precedence when passed explicitly (see the
// merge logic in runBacktest/runCompare).
//
// StrategyParams has no corresponding CLI flag (there's no per-key flag
// for e.g. sma-crossover's shortPeriod/longPeriod - it wouldn't scale
// across strategies' differently-shaped param sets), so it's only settable
// via this config file, unconditionally (there's no flag value it could
// ever lose an override to).
type config struct {
	Asset          string             `json:"asset"`
	Start          string             `json:"start"`
	End            string             `json:"end"`
	Balance        string             `json:"balance"`
	Strategy       string             `json:"strategy"`
	Verbose        bool               `json:"verbose"`
	StrategyParams map[string]float64 `json:"strategyParams"`
}

// loadConfig reads and parses the JSON config file at path. Errors
// distinguish an unreadable file (e.g. missing, permission denied) from
// malformed JSON in it, so a user hitting either can tell which one it is.
// Unknown fields (e.g. a typo like "assset") are rejected rather than
// silently ignored, since a silently-dropped field would otherwise just
// leave that flag at its empty default with no clue why.
func loadConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return &cfg, nil
}

// explicitFlags returns the set of flag names actually passed on the
// command line for fs, via fs.Visit (which, unlike fs.VisitAll, only walks
// flags that were set). It's used to determine whether a flag's parsed
// value should win over the config file's, or vice versa.
func explicitFlags(fs *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

// applyConfigDefaults fills in asset/start/end/balance/strategy/verbose from
// cfg wherever the corresponding flag wasn't explicitly passed (per
// visited, as returned by explicitFlags). Flags the user did pass are left
// untouched, so they always win over the config file.
func applyConfigDefaults(cfg *config, visited map[string]bool, asset, start, end, balance, strategy *string, verbose *bool) {
	if !visited["asset"] && cfg.Asset != "" {
		*asset = cfg.Asset
	}
	if !visited["start"] && cfg.Start != "" {
		*start = cfg.Start
	}
	if !visited["end"] && cfg.End != "" {
		*end = cfg.End
	}
	if !visited["balance"] && cfg.Balance != "" {
		*balance = cfg.Balance
	}
	if !visited["strategy"] && cfg.Strategy != "" {
		*strategy = cfg.Strategy
	}
	if !visited["v"] {
		*verbose = cfg.Verbose
	}
}
