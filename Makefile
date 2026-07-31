IMPORT_YEAR ?= 2011

TICKER ?= PETR4

START ?= 2015-01-02
END ?= 2015-01-02

STRATEGY ?= buy-and-hold
BALANCE ?= 10000.00

ADDR ?= :8080

import:
	go run scripts/import_cotahist.go \
		-in resources/COTAHIST_A$(IMPORT_YEAR).TXT \
		-out resources/cotahist

backtest:
	go run ./cmd backtest -ticker $(TICKER) -start $(START) -end $(END) -strategy $(STRATEGY) -balance $(BALANCE)

info:
	go run ./cmd info -ticker $(TICKER)

serve:
	go run ./cmd serve -addr $(ADDR)

test:
	go test ./...

frontend:
	cd frontend && npm run dev