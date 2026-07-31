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

export interface TickerInfo {
  ticker: string;
  earliest: string; // YYYY-MM-DD
  latest: string; // YYYY-MM-DD
}

export function getTickerInfo(ticker: string): Promise<TickerInfo> {
  const params = new URLSearchParams({ ticker });
  return request<TickerInfo>(`/info?${params.toString()}`);
}

// -- GET /api/tickers -------------------------------------------------------

export interface TickersResponse {
  stocks: string[];
  others: string[];
}

/**
 * List available tickers, optionally filtered to those with data for a
 * given year. `stocks` holds common equities, `others` holds
 * units/ETFs/FIIs/BDRs/index-tracking tickers. Both arrays are `[]` (never
 * `null`) when nothing matches.
 */
export function getTickers(year?: number): Promise<TickersResponse> {
  const params = year !== undefined ? `?${new URLSearchParams({ year: String(year) }).toString()}` : '';
  return request<TickersResponse>(`/tickers${params}`);
}

// -- GET /api/strategies ---------------------------------------------------

// Bare array of strategy name strings, sorted alphabetically (e.g.
// `["buy-and-hold"]`).
export function getStrategies(): Promise<string[]> {
  return request<string[]>('/strategies');
}

// -- POST /api/backtest -----------------------------------------------------

export interface BacktestRequest {
  ticker: string;
  start: string; // YYYY-MM-DD
  end: string; // YYYY-MM-DD
  strategy: string;
  balance: string;
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
  // Present only when the request set `verbose: true`.
  operations?: Operation[];
}

export function runBacktest(payload: BacktestRequest): Promise<BacktestResult> {
  return request<BacktestResult>('/backtest', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}
