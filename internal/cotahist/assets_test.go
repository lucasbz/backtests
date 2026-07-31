package cotahist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucasbz/backtests/internal/domain"
)

func TestLoadAssetsFrom_MissingFileReturnsEmptyNotError(t *testing.T) {
	dir := t.TempDir()

	got, err := LoadAssetsFrom(dir)
	if err != nil {
		t.Fatalf("LoadAssetsFrom: %v", err)
	}
	if got == nil {
		t.Fatal("got nil, want empty non-nil map")
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestLoadAssetsFrom_FillsInTickerFromMapKey(t *testing.T) {
	dir := t.TempDir()
	writeAssetsFile(t, dir, `{
		"PETR4": {"companyName": "PETROBRAS", "specification": "PN", "type": "stock", "isin": "BRPETRACNPR6"},
		"BOVA11": {"companyName": "ISHARES BOVESPA", "specification": "CI", "type": "etf", "isin": "BRBOVACTF003"}
	}`)

	got, err := LoadAssetsFrom(dir)
	if err != nil {
		t.Fatalf("LoadAssetsFrom: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d assets, want 2: %+v", len(got), got)
	}

	petr4 := got["PETR4"]
	want := domain.Asset{
		Ticker:        "PETR4",
		CompanyName:   "PETROBRAS",
		Specification: "PN",
		Type:          domain.Stock,
		ISIN:          "BRPETRACNPR6",
	}
	if petr4 != want {
		t.Errorf("PETR4 = %+v, want %+v", petr4, want)
	}

	if got["BOVA11"].Type != domain.ETF {
		t.Errorf("BOVA11.Type = %q, want %q", got["BOVA11"].Type, domain.ETF)
	}
}

func TestLoadAssetsFrom_MalformedFileErrors(t *testing.T) {
	dir := t.TempDir()
	writeAssetsFile(t, dir, "not json")

	if _, err := LoadAssetsFrom(dir); err == nil {
		t.Fatal("expected error for malformed assets.json")
	}
}

func writeAssetsFile(t *testing.T, dir, contents string) {
	t.Helper()
	path := filepath.Join(dir, AssetsFileName)
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
