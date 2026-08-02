IMPORT_YEAR ?= 2011

ASSET ?= PETR4
BALANCE ?= 10000.00
START ?= 2026-01-01
END ?= 2026-07-30

STRATEGY ?= buy-and-hold
COMPARE_STRATEGY ?= two-candle-breakout
YEAR ?= 0

ADDR ?= :8080

import:
	go run scripts/import_cotahist.go \
		-in resources/COTAHIST_A$(IMPORT_YEAR).TXT \
		-out resources/cotahist

backtest:
	go run ./cmd backtest -asset $(ASSET) -start $(START) -end $(END) -strategy $(STRATEGY) -balance $(BALANCE)

v-backtest:
	go run ./cmd backtest -v -asset $(ASSET) -start $(START) -end $(END) -strategy $(STRATEGY) -balance $(BALANCE)

compare:
	go run ./cmd compare -asset $(ASSET) -start $(START) -end $(END) -strategy $(COMPARE_STRATEGY) -balance $(BALANCE)

v-compare:
	go run ./cmd compare -v -asset $(ASSET) -start $(START) -end $(END) -strategy $(COMPARE_STRATEGY) -balance $(BALANCE)

scan:
	go run ./cmd scan -start $(START) -end $(END) -strategy $(COMPARE_STRATEGY) -balance $(BALANCE) -year $(YEAR)

v-scan:
	go run ./cmd scan -v -start $(START) -end $(END) -strategy $(COMPARE_STRATEGY) -balance $(BALANCE) -year $(YEAR)

info:
	go run ./cmd info -asset $(ASSET)

serve:
	go run ./cmd serve -addr $(ADDR)

test:
	go test ./...

react:
	cd frontend && npm run dev