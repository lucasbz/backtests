import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useIndicators } from './useIndicators';
import { ApiError, getIndicator, type IndicatorPoint } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getIndicator: vi.fn() };
});

const mockedGetIndicator = vi.mocked(getIndicator);

beforeEach(() => {
  mockedGetIndicator.mockReset();
});

const emaPoints: IndicatorPoint[] = [
  { date: '2024-01-08', value: 32.1 },
  { date: '2024-01-09', value: 32.4 },
];

const smaPoints: IndicatorPoint[] = [{ date: '2024-01-20', value: 31.9 }];

describe('useIndicators', () => {
  it('makes no request and returns a blank result when asset is null', () => {
    const { result } = renderHook(() =>
      useIndicators(null, '2024-01-01', '2024-12-31', [{ type: 'ema', period: 8 }]),
    );

    expect(result.current).toEqual({ series: [], loading: false, error: null });
    expect(mockedGetIndicator).not.toHaveBeenCalled();
  });

  it('makes no request when start is missing', () => {
    const { result } = renderHook(() =>
      useIndicators('PETR4', undefined, '2024-12-31', [{ type: 'ema', period: 8 }]),
    );

    expect(result.current).toEqual({ series: [], loading: false, error: null });
    expect(mockedGetIndicator).not.toHaveBeenCalled();
  });

  it('makes no request when end is missing', () => {
    const { result } = renderHook(() =>
      useIndicators('PETR4', '2024-01-01', undefined, [{ type: 'ema', period: 8 }]),
    );

    expect(result.current).toEqual({ series: [], loading: false, error: null });
    expect(mockedGetIndicator).not.toHaveBeenCalled();
  });

  it('makes no request when selected is empty', () => {
    const { result } = renderHook(() => useIndicators('PETR4', '2024-01-01', '2024-12-31', []));

    expect(result.current).toEqual({ series: [], loading: false, error: null });
    expect(mockedGetIndicator).not.toHaveBeenCalled();
  });

  it('fetches each selected indicator in parallel and pairs points back up with type/period', async () => {
    mockedGetIndicator.mockImplementation((type) =>
      Promise.resolve(type === 'ema' ? emaPoints : smaPoints),
    );

    const { result } = renderHook(() =>
      useIndicators('PETR4', '2024-01-01', '2024-12-31', [
        { type: 'ema', period: 8 },
        { type: 'sma', period: 20 },
      ]),
    );

    expect(result.current.loading).toBe(true);

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.series).toEqual([
      { type: 'ema', period: 8, points: emaPoints },
      { type: 'sma', period: 20, points: smaPoints },
    ]);
    expect(result.current.error).toBeNull();
    expect(mockedGetIndicator).toHaveBeenCalledWith('ema', 'PETR4', '2024-01-01', '2024-12-31', 8);
    expect(mockedGetIndicator).toHaveBeenCalledWith('sma', 'PETR4', '2024-01-01', '2024-12-31', 20);
  });

  it('surfaces an ApiError message on failure', async () => {
    mockedGetIndicator.mockRejectedValue(new ApiError(400, 'bad period'));

    const { result } = renderHook(() =>
      useIndicators('PETR4', '2024-01-01', '2024-12-31', [{ type: 'ema', period: 8 }]),
    );

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('bad period');
    expect(result.current.series).toEqual([]);
  });

  it('falls back to a generic message for a non-ApiError failure', async () => {
    mockedGetIndicator.mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() =>
      useIndicators('PETR4', '2024-01-01', '2024-12-31', [{ type: 'ema', period: 8 }]),
    );

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('Unexpected error loading indicators.');
  });

  it('treats one failed indicator request as an overall error, not partial success', async () => {
    mockedGetIndicator.mockImplementation((type) =>
      type === 'ema' ? Promise.resolve(emaPoints) : Promise.reject(new ApiError(400, 'bad period')),
    );

    const { result } = renderHook(() =>
      useIndicators('PETR4', '2024-01-01', '2024-12-31', [
        { type: 'ema', period: 8 },
        { type: 'sma', period: 20 },
      ]),
    );

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('bad period');
    expect(result.current.series).toEqual([]);
  });

  it('ignores a stale in-flight response when the asset changes before it resolves', async () => {
    let resolveFirst: (points: IndicatorPoint[]) => void;
    const firstPromise = new Promise<IndicatorPoint[]>((resolve) => {
      resolveFirst = resolve;
    });
    const secondPoints: IndicatorPoint[] = [{ date: '2024-02-08', value: 30.1 }];

    mockedGetIndicator.mockImplementationOnce(() => firstPromise);
    mockedGetIndicator.mockResolvedValueOnce(secondPoints);

    const { result, rerender } = renderHook(
      ({ asset }) => useIndicators(asset, '2024-01-01', '2024-12-31', [{ type: 'ema', period: 8 }]),
      { initialProps: { asset: 'PETR4' as string | null } },
    );

    rerender({ asset: 'VALE3' });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.series).toEqual([{ type: 'ema', period: 8, points: secondPoints }]);

    resolveFirst!(emaPoints);
    await Promise.resolve();
    await Promise.resolve();

    expect(result.current.series).toEqual([{ type: 'ema', period: 8, points: secondPoints }]);
  });

  it('clears series/error and resets loading when asset goes back to null', () => {
    mockedGetIndicator.mockReturnValue(new Promise(() => {}));

    const { result, rerender } = renderHook(
      ({ asset }) => useIndicators(asset, '2024-01-01', '2024-12-31', [{ type: 'ema', period: 8 }]),
      { initialProps: { asset: 'PETR4' as string | null } },
    );

    rerender({ asset: null });

    expect(result.current).toEqual({ series: [], loading: false, error: null });
  });
});
