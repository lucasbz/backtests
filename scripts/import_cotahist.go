package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Bar is one daily price bar for a single ticker.
type Bar struct {
	Date     string  `json:"date"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Avg      float64 `json:"avg"`
	Close    float64 `json:"close"`
	Quantity int64   `json:"quantity"`
	Volume   float64 `json:"volume"`
	Trades   int64   `json:"trades"`
}

// includedBDI lists the CODBDI (instrument category) codes treated as
// stocks/BDRs/ETFs in scope for this import. Verified against
// resources/COTAHIST_A2010.TXT:
//   - 02: spot lot, common/preferred stock, units and BDRs (e.g. WSON11=DR3)
//   - 06: concordata (companies under old bankruptcy-protection regime)
//   - 08: judicial recovery stocks
//   - 14: index/ETF funds (e.g. BOVA11)
//   - 58: other real stocks (e.g. IGBR3), catch-all administrative regime
//
// Excluded: 10 (rights/receipts), 12 (FIIs), 22 (bonus), 30 (subscription
// forward), 32/33 (index options), 38 (unlisted auction), 42/46/49/50/51/52
// (auctions/subscription misc), 62 (forward market, TPMERC=030), 74/75
// (index-related), 78/82 (stock options — confirmed via populated
// PREEXE+DATVEN fields), 96 (fractional lot, TPMERC=020). 62 and 96 are also
// covered by excludedSuffixes below, since every row under those codes has a
// CODNEG ending in F/S/T.
var includedBDI = map[string]bool{
	"02": true,
	"06": true,
	"08": true,
	"14": true,
	"58": true,
}

// excludedSuffixes drops fractional-lot (F) and forward/term-market (S, T)
// variants of a ticker, keeping only the spot-market series per underlying so
// each ticker file has one bar per date. Verified against
// resources/COTAHIST_A2010.TXT: no spot-market ticker ends in F, S, or T.
var excludedSuffixes = map[string]bool{
	"F": true,
	"S": true,
	"T": true,
}

const recordLen = 245

// parseLine parses a single line of a COTAHIST file.
//
// ok is false (with err nil) for header/trailer lines and for trade lines
// whose instrument category isn't in includedBDI. err is non-nil only for a
// structurally malformed 01 (trade) line.
func parseLine(line string) (ticker string, bar Bar, ok bool, err error) {
	if len(line) < 2 || line[0:2] != "01" {
		return "", Bar{}, false, nil
	}
	if len(line) < recordLen {
		return "", Bar{}, false, fmt.Errorf("malformed record: want %d bytes, got %d", recordLen, len(line))
	}

	bdi := line[10:12]
	if !includedBDI[bdi] {
		return "", Bar{}, false, nil
	}

	ticker = strings.TrimSpace(line[12:24])
	if suffix := ticker[len(ticker)-1:]; excludedSuffixes[suffix] {
		return "", Bar{}, false, nil
	}

	date, err := time.Parse("20060102", line[2:10])
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing date: %w", err)
	}

	open, err := parsePrice(line[56:69])
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing open: %w", err)
	}
	high, err := parsePrice(line[69:82])
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing high: %w", err)
	}
	low, err := parsePrice(line[82:95])
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing low: %w", err)
	}
	avg, err := parsePrice(line[95:108])
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing avg: %w", err)
	}
	closePrice, err := parsePrice(line[108:121])
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing close: %w", err)
	}
	trades, err := strconv.ParseInt(line[147:152], 10, 64)
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing trades: %w", err)
	}
	quantity, err := strconv.ParseInt(line[152:170], 10, 64)
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing quantity: %w", err)
	}
	volume, err := parsePrice(line[170:188])
	if err != nil {
		return "", Bar{}, false, fmt.Errorf("parsing volume: %w", err)
	}

	bar = Bar{
		Date:     date.Format("2006-01-02"),
		Open:     open,
		High:     high,
		Low:      low,
		Avg:      avg,
		Close:    closePrice,
		Quantity: quantity,
		Volume:   volume,
		Trades:   trades,
	}
	return ticker, bar, true, nil
}

// parsePrice parses a fixed-width digit string with 2 implied decimal places.
func parsePrice(s string) (float64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return float64(n) / 100, nil
}

// parseFile reads a full COTAHIST file and groups its bars by ticker.
func parseFile(r io.Reader) (map[string][]Bar, error) {
	grouped := make(map[string][]Bar)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		ticker, bar, ok, err := parseLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		grouped[ticker] = append(grouped[ticker], bar)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return grouped, nil
}

// groupByYear splits a ticker's bars into buckets keyed by the 4-digit year
// of each bar's date, each sorted by date.
func groupByYear(bars []Bar) map[string][]Bar {
	byYear := make(map[string][]Bar)
	for _, b := range bars {
		year := b.Date[:4]
		byYear[year] = append(byYear[year], b)
	}
	for _, yearBars := range byYear {
		sort.Slice(yearBars, func(i, j int) bool { return yearBars[i].Date < yearBars[j].Date })
	}
	return byYear
}

// writeTickerFiles writes one JSON file per ticker per year, at
// outDir/TICKER/TICKER_YEAR.json. Re-running for a given ticker/year
// overwrites that year's file wholesale; other years are untouched.
func writeTickerFiles(outDir string, grouped map[string][]Bar) error {
	for ticker, bars := range grouped {
		tickerDir := filepath.Join(outDir, ticker)
		if err := os.MkdirAll(tickerDir, 0755); err != nil {
			return err
		}

		for year, yearBars := range groupByYear(bars) {
			data, err := json.MarshalIndent(yearBars, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling %s %s: %w", ticker, year, err)
			}
			path := filepath.Join(tickerDir, fmt.Sprintf("%s_%s.json", ticker, year))
			if err := os.WriteFile(path, data, 0644); err != nil {
				return fmt.Errorf("writing %s: %w", path, err)
			}
		}
	}
	return nil
}

func main() {
	inFile := flag.String("in", "", "path to input COTAHIST file")
	outDir := flag.String("out", "resources/cotahist", "output directory for per-ticker JSON files")
	flag.Parse()

	if *inFile == "" {
		fmt.Fprintln(os.Stderr, "usage: import_cotahist -in <COTAHIST file> [-out <output dir>]")
		os.Exit(1)
	}

	f, err := os.Open(*inFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	grouped, err := parseFile(f)
	if err != nil {
		panic(err)
	}

	if err := writeTickerFiles(*outDir, grouped); err != nil {
		panic(err)
	}

	fmt.Printf("wrote %d ticker folders to %s\n", len(grouped), *outDir)
}
