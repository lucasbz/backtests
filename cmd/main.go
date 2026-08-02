package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/lucasbz/backtests/internal/api"
	"github.com/lucasbz/backtests/internal/cli"
)

func main() {

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: backtests <backtest|compare|scan|info|serve> [flags]")
	}

	if args[0] == "serve" {
		return serve(args[1:])
	} else {
		return cli.RunCLI(args)
	}
}

// runServe starts an HTTP server exposing the backtest/info commands as a
// JSON API (see internal/api and openapi.yaml). It runs until the server exits
// with an error (e.g. the address is already in use); it never exits on
// its own otherwise.
func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "address to listen on (e.g. :8080)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("listening on %s\n", *addr)
	return http.ListenAndServe(*addr, api.NewHandler())
}
