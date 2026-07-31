# Ticker → Asset migration plan

## Goal

Replace the bare "ticker string" concept with a proper `Asset` concept that
carries metadata (company name, instrument classification, and room to grow),
backed by a JSON file that's bootstrapped from data already parsed during
COTAHIST import and then manually/programmatically enriched over time.

## Key finding driving this plan

`scripts/import_cotahist.go` already parses the COTAHIST `CODBDI` field
(bytes 10-12) per row to decide *whether* to import it (`includedBDI`), but
never persists it — it's used only as a filter, then discarded. The raw
format also carries a company short name (`NOMRES`) and a specification code
(`ESPECI`, e.g. `ON`/`PN`/`PNA`/`UNT`) that aren't parsed at all today.

This means asset metadata doesn't need a separately curated source for its
first pass — it can be derived directly from fields the importer already
reads (for `CODBDI`) or trivially could read (for `NOMRES`/`ESPECI`), during
the same import pass that produces the per-ticker OHLC files. This directly
supersedes `cotahist.IsStock` (the ticker-suffix-digit heuristic added
recently for the stocks/others split in `GET /api/tickers`), replacing a
guess with real classification data.

Note: FIIs (BDI `12`) are currently excluded from import entirely (see
`includedBDI`'s doc comment), so they won't appear in the bootstrapped
`assets.json` either — consistent with "based on what was imported."

## Design

### `assets.json`

One consolidated file at `resources/cotahist/assets.json` (sibling to the
per-ticker OHLC directories, not nested inside them — it's a registry
covering the whole dataset, not per-ticker/per-year data). Shape:

```json
{
  "PETR4": {
    "companyName": "PETROBRAS",
    "specification": "PN",
    "type": "stock"
  },
  "BOVA11": {
    "companyName": "ISHARES BOVESPA",
    "specification": "CI",
    "type": "etf"
  }
}
```

`type` is derived from `CODBDI` + `ESPECI` (BDI `14` → `etf`; BDI `02`/`06`/
`08`/`58` with an `ESPECI` indicating a unit/BDR vs. plain stock →
`stock`/`unit`/`bdr` as appropriate). The exact `ESPECI` → sub-type mapping
needs to be worked out empirically against real COTAHIST sample data during
implementation, the same way `includedBDI` itself was originally verified
against `resources/COTAHIST_A2010.TXT` (per its existing doc comment) —
not something to guess correctly ahead of time in this doc.

### Bootstrap + enrich, not overwrite

Since the file is meant to be "populated with more information" afterward
(manually, or by some future enrichment step — e.g. sector data, which isn't
in COTAHIST at all and would need a separate source later), re-running the
importer must not clobber fields it didn't derive. Import behavior:

1. Load existing `assets.json` if present (empty map if not).
2. For each ticker encountered during this import run, upsert
   `companyName`/`specification`/`type` from the freshly parsed COTAHIST
   data, **leaving any other existing keys on that ticker's entry
   untouched** (a shallow merge per-ticker, not a wholesale
   replace-the-whole-file).
3. Write the merged result back out, sorted by ticker key for a stable,
   diffable file.

This keeps the file safe to hand-edit or extend later without import runs
silently discarding that work.

### Backend: load once, in memory

The Go server loads `assets.json` into memory once (e.g. at `serve` startup,
or lazily on first use with a cached result — implementation detail,
whichever fits the existing `cotahist` package's style best) rather than
re-reading it per request. `cotahist.ListTickers`/`ListTickersFrom` and
`IsStock` get replaced by asset-registry-backed equivalents once this
lands — `GET /api/tickers` groups by real `type` instead of the suffix
heuristic.

### Naming: `Ticker` → `Asset`

Rename internally (Go types/fields/function names: `Backtest.Ticker` →
`Asset`, `cotahist.ListTickers` → `ListAssets`, etc.), but **keep the public
API and frontend field names as `ticker` for now** — same precedent as the
earlier `Quote` → `Candle` rename, which was internal-only and preserved the
JSON wire format exactly. Flagging this as the assumed scope; say so if you
actually want the public-facing name to change too (that would be a
frontend-affecting change, separate from this backend plan).

## Implementation steps

1. Extend `scripts/import_cotahist.go` to parse `NOMRES`/`ESPECI` alongside
   the existing `CODBDI` parsing, and implement the load-merge-write
   `assets.json` bootstrap described above. Work out the `ESPECI`/`CODBDI` →
   `type` mapping against real sample data, documenting it the same way
   `includedBDI` is documented today.
2. Add an `Asset` type (`internal/domain` or a new `internal/asset` package —
   TBD at implementation time based on what reads most naturally) with
   `Ticker`, `CompanyName`, `Specification`, `Type` fields, plus a loader
   that reads `assets.json` into memory.
3. Rename `Ticker` → `Asset` across internal Go identifiers per the naming
   decision above (mechanical rename, no wire-format change — same
   discipline as the `Quote`→`Candle` rename: round-trip a real on-disk
   fixture to confirm zero behavior change).
4. Replace `cotahist.IsStock` (heuristic) with the real `Type` field from
   the loaded asset registry; update `internal/api/api.go`'s tickers
   handler and `openapi.yaml` accordingly (response shape can stay
   `{"stocks": [...], "others": [...]}` for now, just backed by real data —
   or evolve to expose the richer `type` values directly, open to either,
   flag a preference if you have one).
5. Tests: importer merge behavior (existing extra fields survive a
   re-import), asset-type derivation against known sample rows, and the
   updated tickers endpoint grouping.

Frontend changes (if any beyond what already consumes `stocks`/`others`)
are out of scope for this plan — separate follow-up once the backend shape
is settled.

## Open items

- Exact `ESPECI` → sub-type mapping (`stock` vs `unit` vs `bdr`) needs to be
  derived from real data during implementation, not guessed here.
- Whether `GET /api/tickers`'s response should keep the current
  `stocks`/`others` shape or expose richer per-type grouping once real
  classification data exists — no strong opinion yet, flag if you have one.
