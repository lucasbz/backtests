package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// configFlagUsage is shared by the -config flag on runBacktest, runCompare
// and runScan's flag sets.
const configFlagUsage = "path to a JSON config file; when given, every other flag except -v is ignored in favor of the file's asset/start/end/balance/strategy/strategyParams (-v still applies if passed explicitly, otherwise falls back to the file's \"verbose\")"

// config mirrors the -config JSON file's shape. Passing -config is
// all-or-nothing: every other flag except -v is ignored in favor of the
// file, rather than the file merely providing defaults a flag can
// override - see verboseWasSet's doc comment for why -v alone still
// applies independently.
//
// StrategyParams has no corresponding CLI flag (there's no per-key flag
// for e.g. sma-crossover's shortPeriod/longPeriod - it wouldn't scale
// across strategies' differently-shaped param sets), so it's only settable
// via this config file.
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

// verboseWasSet reports whether -v was passed explicitly on the command
// line (via fs.Visit, which unlike fs.VisitAll only walks flags that were
// actually set). -v is the one flag -config doesn't make irrelevant: since
// it has no bearing on which backtest runs, only how much this process
// prints about it, a config file's "verbose" is just its own default,
// still overridable by passing -v explicitly alongside -config.
func verboseWasSet(fs *flag.FlagSet) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "v" {
			set = true
		}
	})
	return set
}
