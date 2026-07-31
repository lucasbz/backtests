IMPORT_YEAR ?= 2011

TICKER ?= PETR4

START ?= 2015-01-02
END ?= 2015-01-02

STRATEGY ?= buy-and-hold
BALANCE ?= 10000.00

import:
	go run scripts/import_cotahist.go \
		-in resources/COTAHIST_A$(IMPORT_YEAR).TXT \
		-out resources/cotahist

backtest:
	go run ./cmd -ticker $(TICKER) -start $(START) -end $(END) -strategy $(STRATEGY) -balance $(BALANCE)

test:
	go test ./...