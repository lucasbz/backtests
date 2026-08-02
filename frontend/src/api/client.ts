// API client for the Go backtesting backend.
//
// The contract for every call below has been checked against `openapi.yaml`
// at the repo root, which is the source of truth. Keep this file in sync
// with it - it is intentionally the single place API calls live.

const API_BASE = '/api';

export interface ApiErrorBody {
  error: string;
}

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`;
    try {
      const body = (await response.json()) as ApiErrorBody;
      if (body?.error) {
        message = body.error;
      }
    } catch {
      // Response body wasn't JSON (or was empty) - fall back to the
      // generic status-based message above.
    }
    throw new ApiError(response.status, message);
  }

  // Some endpoints (e.g. a hypothetical 204) may have no body.
  const text = await response.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

// -- GET /api/info --------------------------------------------------------

export interface AssetInfo {
  asset: string;
  earliest: string; // YYYY-MM-DD
  latest: string; // YYYY-MM-DD
}

export function getAssetInfo(asset: string): Promise<AssetInfo> {
  const params = new URLSearchParams({ asset });
  return request<AssetInfo>(`/info?${params.toString()}`);
}

// -- GET /api/assets -------------------------------------------------------

export interface AssetEntry {
  ticker: string;
  // Trading volume (major currency units, e.g. reais) for the requested
  // year. Present only when a `year` was passed to `getAssets` - absent
  // (not `null`) for the "all years" request, since volume data doesn't
  // exist outside a specific year.
  volume?: number;
}

export interface AssetsResponse {
  stocks: AssetEntry[];
  others: AssetEntry[];
}

/**
 * List available assets, optionally filtered to those with data for a
 * given year. `stocks` holds common equities, `others` holds
 * units/ETFs/FIIs/BDRs/index-tracking tickers. Both arrays are `[]` (never
 * `null`) when nothing matches. Already sorted server-side: by descending
 * `volume` when `year` is given, alphabetically by `ticker` otherwise.
 */
export function getAssets(year?: number): Promise<AssetsResponse> {
  const params = year !== undefined ? `?${new URLSearchParams({ year: String(year) }).toString()}` : '';
  return request<AssetsResponse>(`/assets${params}`);
}

// -- GET /api/strategies ---------------------------------------------------

// One parameter a strategy's constructor expects, describing how a UI
// should render an input for it. `max` is present only when the param has
// an upper bound (absent, not `null`, otherwise).
export interface StrategyParam {
  key: string;
  label: string;
  default: number;
  min: number;
  max?: number;
  step: number;
}

// A strategy's name plus what params (if any) it expects. `params` is `[]`
// (never absent/null) for strategies that take none (e.g. `buy-and-hold`).
export interface StrategyInfo {
  name: string;
  params: StrategyParam[];
}

// Array of per-strategy descriptors, sorted alphabetically by `name`.
export function getStrategies(): Promise<StrategyInfo[]> {
  return request<StrategyInfo[]>('/strategies');
}

// -- GET /api/stop-losses ---------------------------------------------------

// Bare array of stop-loss type name strings, sorted alphabetically (e.g.
// `["fixed-amount", "percent"]`).
export function getStopLosses(): Promise<string[]> {
  return request<string[]>('/stop-losses');
}

// -- POST /api/backtest -----------------------------------------------------

export interface StopLossRequest {
  // One of the names returned by `GET /api/stop-losses`.
  type: string;
  // Meaning depends on `type` (percent below entry price, or a fixed BRL
  // amount below entry price). Must be > 0.
  value: number;
}

export interface BacktestRequest {
  asset: string;
  start: string; // YYYY-MM-DD
  end: string; // YYYY-MM-DD
  strategy: string;
  balance: string;
  // Omit entirely for no stop-loss (the default, unchanged behavior).
  stopLoss?: StopLossRequest;
  // Required keys depend on `strategy` (see the params `GET /api/strategies`
  // reports for it). Omit entirely for parameterless strategies (e.g.
  // `buy-and-hold`, `two-candle-breakout`).
  strategyParams?: Record<string, number>;
  verbose: boolean;
}

export interface Order {
  date: string; // YYYY-MM-DD
  price: number;
  quantity: number;
}

export interface Operation {
  date: string;
  buyOrder: Order;
  sellOrder: Order;
  days: number;
}

export interface BacktestResult {
  strategyName: string;
  startingBalance: number;
  endingBalance: number;
  profit: number;
  profitPercentage: number;
  totalOperations: number;
  gains: number;
  losses: number;
  winRate: number;
  maxDrawdownAmount: number;
  maxDrawdownPercentage: number;
  // Present only when the request set `verbose: true`.
  operations?: Operation[];
}

export function runBacktest(payload: BacktestRequest): Promise<BacktestResult> {
  return request<BacktestResult>('/backtest', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

// -- GET /api/candles --------------------------------------------------------

export interface Candle {
  date: string; // YYYY-MM-DD
  open: number;
  high: number;
  low: number;
  close: number;
  avg: number;
  quantity: number;
  volume: number;
  trades: number;
}

export function getCandles(asset: string, start: string, end: string): Promise<Candle[]> {
  const params = new URLSearchParams({ asset, start, end });
  return request<Candle[]>(`/candles?${params.toString()}`);
}

// -- GET /api/indicators/{sma,ema} ------------------------------------------

export interface IndicatorPoint {
  date: string; // YYYY-MM-DD
  value: number;
}

/**
 * Fetches a single indicator series (`type`, `period`) for `asset` over
 * `[start, end]`. Only covers dates where the indicator has warmed up - the
 * response may start partway through the requested range, with no
 * null/placeholder entries for the warm-up prefix.
 */
export function getIndicator(
  type: 'sma' | 'ema',
  asset: string,
  start: string,
  end: string,
  period: number,
): Promise<IndicatorPoint[]> {
  const params = new URLSearchParams({ asset, start, end, period: String(period) });
  return request<IndicatorPoint[]>(`/indicators/${type}?${params.toString()}`);
}
