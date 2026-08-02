import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useAssetList } from './useAssetList';
import { ApiError, getAssets, type AssetsResponse } from '../api/client';
import type { YearFilterValue } from '../components/YearFilter';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getAssets: vi.fn() };
});

const mockedGetAssets = vi.mocked(getAssets);

beforeEach(() => {
  mockedGetAssets.mockReset();
});

describe('useAssetList', () => {
  it('starts in a loading state with empty lists', () => {
    mockedGetAssets.mockReturnValue(new Promise<AssetsResponse>(() => {}));

    const { result } = renderHook(() => useAssetList('all'));

    expect(result.current).toEqual({ stocks: [], others: [], loading: true, error: null });
  });

  it('fetches without a year param for "all", and splits stocks/others', async () => {
    const response: AssetsResponse = {
      stocks: [{ ticker: 'PETR4' }],
      others: [{ ticker: 'BOVA11' }],
    };
    mockedGetAssets.mockResolvedValue(response);

    const { result } = renderHook(() => useAssetList('all'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.stocks).toEqual([{ ticker: 'PETR4' }]);
    expect(result.current.others).toEqual([{ ticker: 'BOVA11' }]);
    expect(mockedGetAssets).toHaveBeenCalledWith(undefined);
  });

  it('fetches with the given year when one is picked', async () => {
    mockedGetAssets.mockResolvedValue({ stocks: [], others: [] });

    renderHook(() => useAssetList(2024));

    await waitFor(() => expect(mockedGetAssets).toHaveBeenCalledWith(2024));
  });

  it('surfaces an ApiError message on failure', async () => {
    mockedGetAssets.mockRejectedValue(new ApiError(500, 'assets blew up'));

    const { result } = renderHook(() => useAssetList('all'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('assets blew up');
    expect(result.current.stocks).toEqual([]);
    expect(result.current.others).toEqual([]);
  });

  it('falls back to a generic message for a non-ApiError failure', async () => {
    mockedGetAssets.mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() => useAssetList('all'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('Could not load assets.');
  });

  it('ignores a stale in-flight response when the year changes before it resolves', async () => {
    let resolveFirst: (response: AssetsResponse) => void;
    const firstPromise = new Promise<AssetsResponse>((resolve) => {
      resolveFirst = resolve;
    });
    const secondResponse: AssetsResponse = { stocks: [{ ticker: 'VALE3' }], others: [] };

    mockedGetAssets.mockImplementationOnce(() => firstPromise);
    mockedGetAssets.mockResolvedValueOnce(secondResponse);

    const { result, rerender } = renderHook(({ year }) => useAssetList(year), {
      initialProps: { year: 'all' as YearFilterValue },
    });

    rerender({ year: 2024 });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.stocks).toEqual([{ ticker: 'VALE3' }]);

    resolveFirst!({ stocks: [{ ticker: 'STALE' }], others: [] });
    await Promise.resolve();
    await Promise.resolve();

    expect(result.current.stocks).toEqual([{ ticker: 'VALE3' }]);
  });
});
