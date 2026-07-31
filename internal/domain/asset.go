package domain

// AssetType classifies an Asset's instrument category. Derived empirically
// from COTAHIST's CODBDI + ESPECI fields during import - see
// scripts/import_cotahist.go's deriveAssetType for the exact rules and the
// real sample data used to work them out.
type AssetType string

const (
	// Stock is a common equity - an ordinary or preferred share class (B3
	// ESPECI prefixes ON/PN/PNA/PNB/... ).
	Stock AssetType = "stock"
	// Unit is a unit combining multiple share classes into one instrument
	// (ESPECI prefix UNT).
	Unit AssetType = "unit"
	// BDR is a Brazilian Depositary Receipt for a foreign-listed company
	// (ESPECI prefix DR, e.g. DR3, DRN).
	BDR AssetType = "bdr"
	// ETF is an index/exchange-traded fund (CODBDI 14).
	ETF AssetType = "etf"
	// Other is anything that doesn't fit the categories above - a
	// deliberately coarse catch-all rather than a guess (e.g. CEPACs,
	// FIDCs, and other niche instruments that ride along under the same
	// included CODBDI codes as real stocks/ETFs; see deriveAssetType).
	Other AssetType = "other"
)

// Asset is a single tradable instrument's reference metadata: mostly-static
// descriptive data bootstrapped from COTAHIST import data and meant to be
// safe to hand-edit or enrich afterward without an import re-run clobbering
// it - see resources/cotahist/assets.json and scripts/import_cotahist.go's
// load-merge-write behavior.
type Asset struct {
	// Ticker is the B3 ticker symbol, e.g. "PETR4". It's the key under
	// which this Asset is stored in assets.json, so it's redundant on the
	// JSON value itself (json:"-") - callers that load the registry (see
	// internal/cotahist.LoadAssets) fill it in from the map key.
	Ticker string `json:"-"`
	// CompanyName is COTAHIST's NOMRES (issuer short name), e.g.
	// "PETROBRAS".
	CompanyName string `json:"companyName"`
	// Specification is COTAHIST's ESPECI leading token, e.g. "PN", "ON",
	// "UNT", "DR3", "CI". ESPECI is a 10-byte field that can carry a
	// trailing listing-segment code (e.g. "PN      N2") - only the
	// specification itself is kept, not the segment code.
	Specification string `json:"specification"`
	// Type is this asset's coarse classification - see AssetType.
	Type AssetType `json:"type"`
	// ISIN is COTAHIST's CODISI, the instrument's international securities
	// identification number, e.g. "BRPETRACNPR6".
	ISIN string `json:"isin"`
}
