import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useCandles } from './useCandles';
import { ApiError, getCandles, type Candle } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getCandles: vi.fn() };
});

const mockedGetCandles = vi.mocked(getCandles);

beforeEach(() => {
  mockedGetCandles.mockReset();
});

const sampleCandles: Candle[] = [
  {
    date: '2024-01-02',
    open: 32.5,
    high: 33.1,
    low: 32.1,
    close: 32.9,
    avg: 32.7,
    quantity: 1234500,
    volume: 40600000,
    trades: 812,
  },
];

describe('useCandles', () => {
  it('makes no request and returns a blank result when asset is null', () => {
    const { result } = renderHook(() => useCandles(null, '2024-01-01', '2024-12-31'));

    expect(result.current).toEqual({ candles: null, loading: false, error: null });
    expect(mockedGetCandles).not.toHaveBeenCalled();
  });

  it('makes no request when start is missing', () => {
    const { result } = renderHook(() => useCandles('PETR4', undefined, '2024-12-31'));

    expect(result.current).toEqual({ candles: null, loading: false, error: null });
    expect(mockedGetCandles).not.toHaveBeenCalled();
  });

  it('makes no request when end is missing', () => {
    const { result } = renderHook(() => useCandles('PETR4', '2024-01-01', undefined));

    expect(result.current).toEqual({ candles: null, loading: false, error: null });
    expect(mockedGetCandles).not.toHaveBeenCalled();
  });

  it('loads and returns candles once asset/start/end are all present', async () => {
    mockedGetCandles.mockResolvedValue(sampleCandles);

    const { result } = renderHook(() => useCandles('PETR4', '2024-01-01', '2024-12-31'));

    expect(result.current.loading).toBe(true);

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.candles).toEqual(sampleCandles);
    expect(result.current.error).toBeNull();
    expect(mockedGetCandles).toHaveBeenCalledWith('PETR4', '2024-01-01', '2024-12-31');
  });

  it('surfaces an ApiError message on failure', async () => {
    mockedGetCandles.mockRejectedValue(new ApiError(404, 'not found'));

    const { result } = renderHook(() => useCandles('PETR4', '2024-01-01', '2024-12-31'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('not found');
    expect(result.current.candles).toBeNull();
  });

  it('falls back to a generic message for a non-ApiError failure', async () => {
    mockedGetCandles.mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() => useCandles('PETR4', '2024-01-01', '2024-12-31'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('Unexpected error loading candles.');
  });

  it('ignores a stale in-flight response when the asset changes before it resolves', async () => {
    let resolveFirst: (candles: Candle[]) => void;
    const firstPromise = new Promise<Candle[]>((resolve) => {
      resolveFirst = resolve;
    });
    const secondCandles: Candle[] = [{ ...sampleCandles[0], date: '2024-02-01' }];

    mockedGetCandles.mockImplementationOnce(() => firstPromise);
    mockedGetCandles.mockResolvedValueOnce(secondCandles);

    const { result, rerender } = renderHook(
      ({ asset }) => useCandles(asset, '2024-01-01', '2024-12-31'),
      { initialProps: { asset: 'PETR4' as string | null } },
    );

    rerender({ asset: 'VALE3' });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.candles).toEqual(secondCandles);

    resolveFirst!(sampleCandles);
    await Promise.resolve();
    await Promise.resolve();

    expect(result.current.candles).toEqual(secondCandles);
  });

  it('clears candles/error and resets loading when asset goes back to null', () => {
    mockedGetCandles.mockReturnValue(new Promise(() => {}));

    const { result, rerender } = renderHook(
      ({ asset }) => useCandles(asset, '2024-01-01', '2024-12-31'),
      { initialProps: { asset: 'PETR4' as string | null } },
    );

    rerender({ asset: null });

    expect(result.current).toEqual({ candles: null, loading: false, error: null });
  });
});
