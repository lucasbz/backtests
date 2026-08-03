package cli

import "fmt"

func RunCLI(args []string) error {
	switch args[0] {
	case "backtest":
		return RunBacktest(args[1:])
	case "compare":
		return RunCompare(args[1:])
	case "scan":
		return RunScan(args[1:])
	case "info":
		return RunInfo(args[1:])
	default:
		return fmt.Errorf("unknown command %q, want backtest, compare, scan, info or serve", args[0])
	}
}
