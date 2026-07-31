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

	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
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

// AssetInfo is one ticker's reference metadata as parsed off a single
// COTAHIST trade line - the per-run update that gets merged into
// resources/cotahist/assets.json (see mergeAssets).
type AssetInfo struct {
	CompanyName   string
	Specification string
	Type          domain.AssetType
	ISIN          string
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
func parseLine(line string) (ticker string, bar Bar, asset AssetInfo, ok bool, err error) {
	if len(line) < 2 || line[0:2] != "01" {
		return "", Bar{}, AssetInfo{}, false, nil
	}
	if len(line) < recordLen {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("malformed record: want %d bytes, got %d", recordLen, len(line))
	}

	bdi := line[10:12]
	if !includedBDI[bdi] {
		return "", Bar{}, AssetInfo{}, false, nil
	}

	ticker = strings.TrimSpace(line[12:24])
	if suffix := ticker[len(ticker)-1:]; excludedSuffixes[suffix] {
		return "", Bar{}, AssetInfo{}, false, nil
	}

	date, err := time.Parse("20060102", line[2:10])
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing date: %w", err)
	}

	open, err := parsePrice(line[56:69])
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing open: %w", err)
	}
	high, err := parsePrice(line[69:82])
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing high: %w", err)
	}
	low, err := parsePrice(line[82:95])
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing low: %w", err)
	}
	avg, err := parsePrice(line[95:108])
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing avg: %w", err)
	}
	closePrice, err := parsePrice(line[108:121])
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing close: %w", err)
	}
	trades, err := strconv.ParseInt(line[147:152], 10, 64)
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing trades: %w", err)
	}
	quantity, err := strconv.ParseInt(line[152:170], 10, 64)
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing quantity: %w", err)
	}
	volume, err := parsePrice(line[170:188])
	if err != nil {
		return "", Bar{}, AssetInfo{}, false, fmt.Errorf("parsing volume: %w", err)
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

	// NOMRES (issuer short name), ESPECI (specification + listing-segment
	// code) and CODISI (ISIN) - byte offsets verified against the B3
	// COTAHIST layout and cross-checked against
	// resources/COTAHIST_A2010.TXT/A2015/A2020/A2025 (see
	// specificationOf/deriveAssetType for how ESPECI is interpreted).
	nomres := strings.TrimSpace(line[27:39])
	especi := strings.TrimSpace(line[39:49])
	isin := strings.TrimSpace(line[230:242])

	asset = AssetInfo{
		CompanyName:   nomres,
		Specification: specificationOf(especi),
		Type:          deriveAssetType(bdi, especi),
		ISIN:          isin,
	}

	return ticker, bar, asset, true, nil
}

// specificationOf extracts the actual specification code (e.g. "PN", "ON",
// "UNT", "DR3", "CI") from COTAHIST's 10-byte ESPECI field, which can carry
// a trailing listing-segment code after it (e.g. "PN      N2" for a
// Level-2-governance preferred share). The specification is always ESPECI's
// first whitespace-separated token - verified against real ESPECI values in
// resources/COTAHIST_A2010.TXT/A2015/A2020/A2025.
func specificationOf(especi string) string {
	fields := strings.Fields(especi)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// deriveAssetType classifies an instrument from its CODBDI + ESPECI. Rules
// worked out empirically against resources/COTAHIST_A2010.TXT, .../A2015,
// .../A2020 and .../A2025 (surveying every distinct ESPECI leading token
// under includedBDI across those four years):
//
//   - CODBDI 14 (index/ETF funds) -> etf, regardless of ESPECI. This bucket
//     also carries a handful of non-ETF instruments that ride along under
//     the same code (CEPACs "CPA", FIDCs "FIDC"/"FIDCEA"/"FIDCER", "TPR"),
//     which are classified as etf too rather than special-cased - a
//     deliberate simplification, not a claim that e.g. a CEPAC literally is
//     an ETF.
//   - Otherwise (CODBDI 02/06/08/58), classify by specificationOf(ESPECI):
//   - "ON"/"PN"/"PNA"/"PNB"/"PNC"/"PND"/"PNE"/"PNF" (including starred
//     variants like "PNA*") -> stock.
//   - "UNT" -> unit.
//   - "DR2"/"DR3"/"DRN" (any "DR"-prefixed code) -> bdr.
//   - Anything else (e.g. "IBO" for the one-off IBOV11 index-tracking
//     ticker filed under CODBDI 02 instead of 14, or a handful of
//     malformed/truncated ESPECI values seen in later years, like SMLL11's
//     in COTAHIST_A2025.TXT) -> other.
func deriveAssetType(bdi, especi string) domain.AssetType {
	if bdi == "14" {
		return domain.ETF
	}

	spec := specificationOf(especi)
	switch {
	case strings.HasPrefix(spec, "ON"), strings.HasPrefix(spec, "PN"):
		return domain.Stock
	case spec == "UNT":
		return domain.Unit
	case strings.HasPrefix(spec, "DR"):
		return domain.BDR
	default:
		return domain.Other
	}
}

// parsePrice parses a fixed-width digit string with 2 implied decimal places.
func parsePrice(s string) (float64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return float64(n) / 100, nil
}

// parseFile reads a full COTAHIST file, grouping its bars by ticker and
// collecting each ticker's latest-seen AssetInfo (COTAHIST's per-row
// company/specification/ISIN fields are effectively constant across all of
// a single ticker's rows in one file, so "latest wins" is just a simple way
// to pick one, not a meaningful precedence rule).
func parseFile(r io.Reader) (map[string][]Bar, map[string]AssetInfo, error) {
	grouped := make(map[string][]Bar)
	assets := make(map[string]AssetInfo)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		ticker, bar, asset, ok, err := parseLine(scanner.Text())
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			continue
		}
		grouped[ticker] = append(grouped[ticker], bar)
		assets[ticker] = asset
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return grouped, assets, nil
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

// loadExistingAssets reads path's asset registry as raw JSON objects rather
// than a fixed Go struct, so mergeAssets can preserve fields this importer
// doesn't know about (e.g. hand-added enrichment like "sector") across a
// re-run. A missing file is treated as an empty registry, not an error -
// the first import run bootstraps assets.json from scratch.
func loadExistingAssets(path string) (map[string]map[string]any, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var existing map[string]map[string]any
	if err := json.Unmarshal(data, &existing); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if existing == nil {
		existing = map[string]map[string]any{}
	}
	return existing, nil
}

// mergeAssets upserts each ticker in updates into existing, overwriting only
// the four fields this importer derives (companyName/specification/type/
// isin) and leaving any other existing keys on that ticker's entry
// untouched - see docs/plans/ticker-to-asset.md's "Bootstrap + enrich, not
// overwrite". existing is mutated in place and returned for convenience.
func mergeAssets(existing map[string]map[string]any, updates map[string]AssetInfo) map[string]map[string]any {
	if existing == nil {
		existing = map[string]map[string]any{}
	}
	for ticker, info := range updates {
		entry, ok := existing[ticker]
		if !ok || entry == nil {
			entry = map[string]any{}
		}
		entry["companyName"] = info.CompanyName
		entry["specification"] = info.Specification
		entry["type"] = string(info.Type)
		entry["isin"] = info.ISIN
		existing[ticker] = entry
	}
	return existing
}

// writeAssets marshals assets to path, indented and with top-level ticker
// keys sorted alphabetically (encoding/json already sorts Go map[string]...
// keys when marshaling, so this is automatic) for a stable, diffable file
// that's safe to hand-edit.
func writeAssets(path string, assets map[string]map[string]any) error {
	data, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling assets: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// updateAssetsFile loads outDir's existing asset registry, merges in
// updates (see mergeAssets) and writes the result back - the
// bootstrap-and-enrich behavior described in
// docs/plans/ticker-to-asset.md.
func updateAssetsFile(outDir string, updates map[string]AssetInfo) error {
	path := filepath.Join(outDir, cotahist.AssetsFileName)

	existing, err := loadExistingAssets(path)
	if err != nil {
		return err
	}

	merged := mergeAssets(existing, updates)

	return writeAssets(path, merged)
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

	grouped, assets, err := parseFile(f)
	if err != nil {
		panic(err)
	}

	if err := writeTickerFiles(*outDir, grouped); err != nil {
		panic(err)
	}

	if err := updateAssetsFile(*outDir, assets); err != nil {
		panic(err)
	}

	fmt.Printf("wrote %d ticker folders to %s\n", len(grouped), *outDir)
}
