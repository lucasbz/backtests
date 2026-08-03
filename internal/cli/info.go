package cli

import (
	"flag"
	"fmt"

	"github.com/lucasbz/backtests/internal/cotahist"
)

func RunInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	asset := fs.String("asset", "", "asset (ticker) to look up (e.g. PETR4)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *asset == "" {
		fs.Usage()
		return fmt.Errorf("-asset is required")
	}

	earliest, latest, err := cotahist.DateRange(*asset)
	if err != nil {
		return err
	}

	fmt.Printf("%s: data available from %s to %s\n", *asset, earliest, latest)
	return nil
}
