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

type config struct {
	Asset          string             `json:"asset"`
	Start          string             `json:"start"`
	End            string             `json:"end"`
	Balance        string             `json:"balance"`
	Strategy       string             `json:"strategy"`
	Verbose        bool               `json:"verbose"`
	StrategyParams map[string]float64 `json:"strategyParams"`
}

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

type resolvedParams struct {
	Asset          string
	Start          string
	End            string
	Balance        string
	Strategy       string
	StrategyParams map[string]float64
	Verbose        bool
}

func resolveParams(fs *flag.FlagSet, configPath, asset, start, end, balance, strategyName string, verbose bool) (resolvedParams, error) {
	params := resolvedParams{
		Asset:    asset,
		Start:    start,
		End:      end,
		Balance:  balance,
		Strategy: strategyName,
		Verbose:  verbose,
	}
	if configPath == "" {
		return params, nil
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return resolvedParams{}, err
	}
	params.Asset, params.Start, params.End, params.Balance, params.Strategy = cfg.Asset, cfg.Start, cfg.End, cfg.Balance, cfg.Strategy
	params.StrategyParams = cfg.StrategyParams
	if !verboseWasSet(fs) {
		params.Verbose = cfg.Verbose
	}
	return params, nil
}
