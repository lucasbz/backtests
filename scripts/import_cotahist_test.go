package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
)

// Fixture lines copied verbatim from resources/COTAHIST_A2010.TXT.
const (
	fixtureHeader  = "00COTAHIST.2010BOVESPA 20101230                                                                                                                                                                                                                      "
	fixtureTrailer = "99COTAHIST.2010BOVESPA 2010123000000287482                                                                                                                                                                                                           "
	// ABCB4, spot lot (BDI 02).
	fixtureABCB4Spot = "012010010402ABCB4       010ABC BRASIL  PN  EJ  N2   R$  000000000120000000000012420000000001200000000000123200000000012350000000001234000000000123500421000000000000205500000000000253269800000000000000009999123100000010000000000000BRABCBACNPR4109"
	// ABCB4F, fractional lot (BDI 96) of the same underlying - excluded (F suffix).
	fixtureABCB4Fractional = "012010010496ABCB4F      020ABC BRASIL  PN  EJ  N2   R$  000000000121000000000012510000000001210000000000122800000000012400000000001238000000000124000017000000000000000423000000000000520805000000000000009999123100000010000000000000BRABCBACNPR4109"
	// ABCB4T, forward market of the same underlying - excluded (T suffix).
	fixtureABCB4Forward = "012010010562ABCB4T      030ABC BRASIL  PN  EJ  N2060R$  000000000126600000000012670000000001266000000000126600000000012670000000000000000000000000000002000000000000005000000000000006332500000000000000009999123100000010000000000000BRABCBACNPR4109"
	// ALMI11B, FII (BDI 12) - excluded from scope.
	fixtureFII = "012010010412ALMI11B     010FII TORRE ALCI  ER  MB   R$  000000017500000000001750000000000175000000000017500000000001750000000000000000000000000000000004000000000000000046000000000008050000000000000000009999123100000010000000000000BRALMICTF003161"
	// BBASA31, stock option (BDI 78) - excluded, has strike/expiry populated.
	fixtureOption = "012010010478BBASA31     070BBASE   /EJ ON      NM000R$  000000000002400000000000250000000000024000000000002400000000000250000000000000000000000000000004000000000000001800000000000000044600000000000308402010011800000010000000000000BRBBASACNOR3208"
	// BOVA11, index/ETF fund (BDI 14).
	fixtureBOVA11 = "012010010414BOVA11      010ISHARES BOVACI           R$  000000000685500000000069450000000006830000000000691100000000069370000000006937000000000694500106000000000000168300000000001163259800000000000000009999123100000010000000000000BRBOVACTF003100"
)

func TestParseLine_SpotStock(t *testing.T) {
	ticker, bar, asset, ok, err := parseLine(fixtureABCB4Spot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if ticker != "ABCB4" {
		t.Errorf("ticker = %q, want %q", ticker, "ABCB4")
	}
	want := Bar{
		Date:     "2010-01-04",
		Open:     12.00,
		High:     12.42,
		Low:      12.00,
		Avg:      12.32,
		Close:    12.35,
		Quantity: 205500,
		Volume:   2532698.00,
		Trades:   421,
	}
	if bar != want {
		t.Errorf("bar = %+v, want %+v", bar, want)
	}

	wantAsset := AssetInfo{
		CompanyName:   "ABC BRASIL",
		Specification: "PN",
		Type:          domain.Stock,
		ISIN:          "BRABCBACNPR4",
	}
	if asset != wantAsset {
		t.Errorf("asset = %+v, want %+v", asset, wantAsset)
	}
}

func TestParseLine_ETF(t *testing.T) {
	ticker, _, asset, ok, err := parseLine(fixtureBOVA11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if ticker != "BOVA11" {
		t.Errorf("ticker = %q, want %q", ticker, "BOVA11")
	}

	wantAsset := AssetInfo{
		CompanyName:   "ISHARES BOVA",
		Specification: "CI",
		Type:          domain.ETF,
		ISIN:          "BRBOVACTF003",
	}
	if asset != wantAsset {
		t.Errorf("asset = %+v, want %+v", asset, wantAsset)
	}
}

func TestParseLine_SuffixedVariantsExcluded(t *testing.T) {
	for name, line := range map[string]string{
		"fractional (F)": fixtureABCB4Fractional,
		"forward (T)":    fixtureABCB4Forward,
	} {
		ticker, _, _, ok, err := parseLine(line)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if ok {
			t.Errorf("%s: expected ok=false, got ticker %q", name, ticker)
		}
	}
}

func TestParseLine_HeaderAndTrailerSkipped(t *testing.T) {
	for _, line := range []string{fixtureHeader, fixtureTrailer} {
		ticker, _, _, ok, err := parseLine(line)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", line[:2], err)
		}
		if ok {
			t.Errorf("expected ok=false for record type %q, got ticker %q", line[:2], ticker)
		}
	}
}

func TestParseLine_ExcludedInstrumentTypes(t *testing.T) {
	for name, line := range map[string]string{
		"FII":    fixtureFII,
		"option": fixtureOption,
	} {
		ticker, _, _, ok, err := parseLine(line)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if ok {
			t.Errorf("%s: expected ok=false, got ticker %q", name, ticker)
		}
	}
}

func TestParseLine_MalformedRecord(t *testing.T) {
	truncated := fixtureABCB4Spot[:100]
	_, _, _, ok, err := parseLine(truncated)
	if err == nil {
		t.Fatalf("expected error for truncated record")
	}
	if ok {
		t.Errorf("expected ok=false alongside error")
	}
}

func TestSpecificationOf(t *testing.T) {
	cases := []struct {
		especi string
		want   string
	}{
		{"PN", "PN"},
		{"PN  EJ  N2", "PN"},
		{"ON      NM", "ON"},
		{"UNT     N2", "UNT"},
		{"DR3", "DR3"},
		{"CI *", "CI"},
		{"PNA*", "PNA*"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		if got := specificationOf(c.especi); got != c.want {
			t.Errorf("specificationOf(%q) = %q, want %q", c.especi, got, c.want)
		}
	}
}

func TestDeriveAssetType(t *testing.T) {
	cases := []struct {
		name   string
		bdi    string
		especi string
		want   domain.AssetType
	}{
		{"stock, ordinary", "02", "ON      NM", domain.Stock},
		{"stock, preferred", "02", "PN", domain.Stock},
		{"stock, preferred class A", "08", "PNA*", domain.Stock},
		{"judicial recovery stock", "08", "ON", domain.Stock},
		{"concordata stock", "06", "PN", domain.Stock},
		{"catch-all admin regime stock", "58", "ON", domain.Stock},
		{"unit", "02", "UNT     N2", domain.Unit},
		{"bdr, DR3", "02", "DR3", domain.BDR},
		{"bdr, DRN", "02", "DRN     MB", domain.BDR},
		{"etf, CI", "14", "CI", domain.ETF},
		{"etf bucket, non-ETF rider (CEPAC)", "14", "CPA FLI MB", domain.ETF},
		{"one-off index ticker outside BDI 14", "02", "IBO", domain.Other},
		{"unrecognized specification", "02", "", domain.Other},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deriveAssetType(c.bdi, c.especi); got != c.want {
				t.Errorf("deriveAssetType(%q, %q) = %q, want %q", c.bdi, c.especi, got, c.want)
			}
		})
	}
}

func TestParseFile_GroupsByTicker(t *testing.T) {
	input := strings.Join([]string{
		fixtureHeader,
		fixtureABCB4Spot,
		fixtureABCB4Fractional,
		fixtureABCB4Forward,
		fixtureFII,
		fixtureOption,
		fixtureBOVA11,
		fixtureTrailer,
	}, "\n")

	grouped, assets, err := parseFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(grouped) != 2 {
		t.Fatalf("got %d tickers, want 2: %v", len(grouped), keysOf(grouped))
	}
	if len(grouped["ABCB4"]) != 1 {
		t.Errorf("ABCB4 bars = %d, want 1", len(grouped["ABCB4"]))
	}
	if _, present := grouped["ABCB4F"]; present {
		t.Errorf("ABCB4F (fractional) should have been excluded")
	}
	if _, present := grouped["ABCB4T"]; present {
		t.Errorf("ABCB4T (forward) should have been excluded")
	}
	if _, present := grouped["ALMI11B"]; present {
		t.Errorf("ALMI11B (FII) should have been excluded")
	}
	if _, present := grouped["BBASA31"]; present {
		t.Errorf("BBASA31 (option) should have been excluded")
	}

	if len(assets) != 2 {
		t.Fatalf("got %d assets, want 2: %v", len(assets), assets)
	}
	if got, want := assets["ABCB4"].Type, domain.Stock; got != want {
		t.Errorf("ABCB4 asset type = %q, want %q", got, want)
	}
	if got, want := assets["BOVA11"].Type, domain.ETF; got != want {
		t.Errorf("BOVA11 asset type = %q, want %q", got, want)
	}
}

func keysOf(m map[string][]Bar) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestGroupByYear_SplitsAndSortsPerYear(t *testing.T) {
	bars := []Bar{
		{Date: "2010-06-15"},
		{Date: "2009-12-30"},
		{Date: "2010-01-04"},
	}

	got := groupByYear(bars)

	if len(got) != 2 {
		t.Fatalf("got %d years, want 2: %v", len(got), got)
	}
	if len(got["2009"]) != 1 || got["2009"][0].Date != "2009-12-30" {
		t.Errorf("2009 bucket = %+v, want [2009-12-30]", got["2009"])
	}
	if len(got["2010"]) != 2 || got["2010"][0].Date != "2010-01-04" || got["2010"][1].Date != "2010-06-15" {
		t.Errorf("2010 bucket = %+v, want sorted [2010-01-04, 2010-06-15]", got["2010"])
	}
}

func TestWriteTickerFiles_OneFilePerTickerPerYear(t *testing.T) {
	dir := t.TempDir()

	grouped := map[string][]Bar{
		"ABCB4": {
			{Date: "2010-01-04", Close: 12.35},
			{Date: "2011-01-03", Close: 20.00},
		},
	}
	if err := writeTickerFiles(dir, grouped); err != nil {
		t.Fatalf("write: %v", err)
	}

	bars2010 := readBarsFile(t, dir, "ABCB4", "2010")
	if len(bars2010) != 1 || bars2010[0].Date != "2010-01-04" {
		t.Errorf("ABCB4_2010.json = %+v, want [2010-01-04]", bars2010)
	}

	bars2011 := readBarsFile(t, dir, "ABCB4", "2011")
	if len(bars2011) != 1 || bars2011[0].Date != "2011-01-03" {
		t.Errorf("ABCB4_2011.json = %+v, want [2011-01-03]", bars2011)
	}
}

func TestWriteTickerFiles_RerunOverwritesOnlyThatYear(t *testing.T) {
	dir := t.TempDir()

	first := map[string][]Bar{
		"ABCB4": {
			{Date: "2010-01-04", Close: 12.35},
			{Date: "2011-01-03", Close: 20.00},
		},
	}
	if err := writeTickerFiles(dir, first); err != nil {
		t.Fatalf("first write: %v", err)
	}

	rerun2010 := map[string][]Bar{
		"ABCB4": {{Date: "2010-01-04", Close: 99.0}},
	}
	if err := writeTickerFiles(dir, rerun2010); err != nil {
		t.Fatalf("second write: %v", err)
	}

	bars2010 := readBarsFile(t, dir, "ABCB4", "2010")
	if len(bars2010) != 1 || bars2010[0].Close != 99.0 {
		t.Errorf("ABCB4_2010.json = %+v, want overwritten close=99.0", bars2010)
	}

	bars2011 := readBarsFile(t, dir, "ABCB4", "2011")
	if len(bars2011) != 1 || bars2011[0].Date != "2011-01-03" {
		t.Errorf("ABCB4_2011.json = %+v, want untouched [2011-01-03]", bars2011)
	}
}

func readBarsFile(t *testing.T, dir, ticker, year string) []Bar {
	t.Helper()
	path := filepath.Join(dir, ticker, fmt.Sprintf("%s_%s.json", ticker, year))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var bars []Bar
	if err := json.Unmarshal(data, &bars); err != nil {
		t.Fatalf("unmarshaling %s: %v", path, err)
	}
	return bars
}

func TestLoadExistingAssets_MissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	got, err := loadExistingAssets(filepath.Join(dir, cotahist.AssetsFileName))
	if err != nil {
		t.Fatalf("loadExistingAssets: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestMergeAssets_UpsertsKnownFieldsAndPreservesUnknownOnes(t *testing.T) {
	existing := map[string]map[string]any{
		"PETR4": {
			"companyName":   "OLD NAME",
			"specification": "PN",
			"type":          "stock",
			"isin":          "OLDISIN",
			// Hand-added enrichment the importer doesn't know about - must
			// survive the merge untouched.
			"sector": "Oil & Gas",
		},
	}
	updates := map[string]AssetInfo{
		"PETR4": {
			CompanyName:   "PETROBRAS",
			Specification: "PN",
			Type:          domain.Stock,
			ISIN:          "BRPETRACNPR6",
		},
		"BOVA11": {
			CompanyName:   "ISHARES BOVESPA",
			Specification: "CI",
			Type:          domain.ETF,
			ISIN:          "BRBOVACTF008",
		},
	}

	got := mergeAssets(existing, updates)

	petr4 := got["PETR4"]
	if petr4["companyName"] != "PETROBRAS" {
		t.Errorf("PETR4 companyName = %v, want PETROBRAS (should be refreshed)", petr4["companyName"])
	}
	if petr4["isin"] != "BRPETRACNPR6" {
		t.Errorf("PETR4 isin = %v, want BRPETRACNPR6 (should be refreshed)", petr4["isin"])
	}
	if petr4["sector"] != "Oil & Gas" {
		t.Errorf("PETR4 sector = %v, want it preserved as %q", petr4["sector"], "Oil & Gas")
	}

	bova11 := got["BOVA11"]
	if bova11["companyName"] != "ISHARES BOVESPA" {
		t.Errorf("BOVA11 companyName = %v, want it to be added fresh", bova11["companyName"])
	}
}

func TestUpdateAssetsFile_BootstrapThenEnrichThenReimportPreservesEnrichment(t *testing.T) {
	dir := t.TempDir()

	// First import run: bootstraps assets.json from scratch.
	first := map[string]AssetInfo{
		"PETR4": {CompanyName: "PETROBRAS", Specification: "PN", Type: domain.Stock, ISIN: "BRPETRACNPR6"},
	}
	if err := updateAssetsFile(dir, first); err != nil {
		t.Fatalf("first updateAssetsFile: %v", err)
	}

	assetsPath := filepath.Join(dir, cotahist.AssetsFileName)
	raw, err := loadExistingAssets(assetsPath)
	if err != nil {
		t.Fatalf("loadExistingAssets: %v", err)
	}
	if raw["PETR4"]["companyName"] != "PETROBRAS" {
		t.Fatalf("bootstrapped PETR4 = %+v, want companyName PETROBRAS", raw["PETR4"])
	}

	// Simulate a hand-edit adding a field the importer doesn't produce.
	raw["PETR4"]["sector"] = "Oil & Gas"
	if err := writeAssets(assetsPath, raw); err != nil {
		t.Fatalf("writeAssets: %v", err)
	}

	// Second import run (e.g. a new year's COTAHIST file for the same
	// ticker): must not clobber the hand-added "sector" field.
	second := map[string]AssetInfo{
		"PETR4": {CompanyName: "PETROBRAS", Specification: "PN", Type: domain.Stock, ISIN: "BRPETRACNPR6"},
	}
	if err := updateAssetsFile(dir, second); err != nil {
		t.Fatalf("second updateAssetsFile: %v", err)
	}

	final, err := loadExistingAssets(assetsPath)
	if err != nil {
		t.Fatalf("loadExistingAssets: %v", err)
	}
	if final["PETR4"]["sector"] != "Oil & Gas" {
		t.Errorf("PETR4 sector = %v, want it preserved as %q across re-import", final["PETR4"]["sector"], "Oil & Gas")
	}
	if final["PETR4"]["companyName"] != "PETROBRAS" {
		t.Errorf("PETR4 companyName = %v, want PETROBRAS", final["PETR4"]["companyName"])
	}
}
