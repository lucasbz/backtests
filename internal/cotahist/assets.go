package cotahist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lucasbz/backtests/internal/domain"
)

// AssetsFileName is the asset registry file bootstrapped and enriched by
// scripts/import_cotahist.go, sibling to the per-ticker OHLC directories
// inside DefaultCotahistDir (a registry covering the whole dataset, not
// per-ticker/per-year data).
const AssetsFileName = "assets.json"

var (
	defaultAssetsOnce sync.Once
	defaultAssets     map[string]domain.Asset
	defaultAssetsErr  error
)

// LoadAssets returns the asset registry from DefaultCotahistDir/assets.json,
// read once and cached in memory for the lifetime of the process (e.g. for
// the "serve" HTTP API - see internal/api). Call LoadAssetsFrom directly if
// you need a fresh read (e.g. in tests, or after the file changed on disk).
func LoadAssets() (map[string]domain.Asset, error) {
	defaultAssetsOnce.Do(func() {
		defaultAssets, defaultAssetsErr = LoadAssetsFrom(DefaultCotahistDir)
	})
	return defaultAssets, defaultAssetsErr
}

// LoadAssetsFrom reads dir/assets.json into a map keyed by ticker, filling
// in each domain.Asset's Ticker field from its map key (see domain.Asset's
// Ticker doc comment). A missing file returns an empty, non-nil map with no
// error - same convention as ListAssetsFrom for a missing directory, since
// assets.json is optional until the importer has been run at least once.
func LoadAssetsFrom(dir string) (map[string]domain.Asset, error) {
	path := filepath.Join(dir, AssetsFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]domain.Asset{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var raw map[string]domain.Asset
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	assets := make(map[string]domain.Asset, len(raw))
	for ticker, asset := range raw {
		asset.Ticker = ticker
		assets[ticker] = asset
	}
	return assets, nil
}
