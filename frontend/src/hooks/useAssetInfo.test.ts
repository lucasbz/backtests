import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useAssetInfo } from './useAssetInfo';
import { ApiError, getAssetInfo, type AssetInfo } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getAssetInfo: vi.fn() };
});

const mockedGetAssetInfo = vi.mocked(getAssetInfo);

beforeEach(() => {
  mockedGetAssetInfo.mockReset();
});

describe('useAssetInfo', () => {
  it('makes no request and returns a blank result when asset is null', () => {
    const { result } = renderHook(() => useAssetInfo(null));

    expect(result.current).toEqual({ info: null, loading: false, error: null });
    expect(mockedGetAssetInfo).not.toHaveBeenCalled();
  });

  it('loads and returns the asset info', async () => {
    const info: AssetInfo = { asset: 'PETR4', earliest: '2010-01-01', latest: '2024-12-31' };
    mockedGetAssetInfo.mockResolvedValue(info);

    const { result } = renderHook(() => useAssetInfo('PETR4'));

    expect(result.current.loading).toBe(true);

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.info).toEqual(info);
    expect(result.current.error).toBeNull();
    expect(mockedGetAssetInfo).toHaveBeenCalledWith('PETR4');
  });

  it('surfaces an ApiError message on failure', async () => {
    mockedGetAssetInfo.mockRejectedValue(new ApiError(404, 'asset not found'));

    const { result } = renderHook(() => useAssetInfo('UNKNOWN'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('asset not found');
    expect(result.current.info).toBeNull();
  });

  it('falls back to a generic message for a non-ApiError failure', async () => {
    mockedGetAssetInfo.mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() => useAssetInfo('PETR4'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe('Unexpected error looking up asset.');
  });

  it('ignores a stale in-flight response when asset changes before it resolves', async () => {
    let resolveFirst: (info: AssetInfo) => void;
    const firstPromise = new Promise<AssetInfo>((resolve) => {
      resolveFirst = resolve;
    });
    const secondInfo: AssetInfo = { asset: 'VALE3', earliest: '2012-01-01', latest: '2024-06-01' };

    mockedGetAssetInfo.mockImplementationOnce(() => firstPromise);
    mockedGetAssetInfo.mockResolvedValueOnce(secondInfo);

    const { result, rerender } = renderHook(({ asset }) => useAssetInfo(asset), {
      initialProps: { asset: 'PETR4' as string | null },
    });

    rerender({ asset: 'VALE3' });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.info).toEqual(secondInfo);

    resolveFirst!({ asset: 'PETR4', earliest: '2010-01-01', latest: '2024-12-31' });
    await Promise.resolve();
    await Promise.resolve();

    expect(result.current.info).toEqual(secondInfo);
  });

  it('clears info/error and resets loading when asset goes back to null', () => {
    mockedGetAssetInfo.mockReturnValue(new Promise(() => {}));

    const { result, rerender } = renderHook(({ asset }) => useAssetInfo(asset), {
      initialProps: { asset: 'PETR4' as string | null },
    });

    rerender({ asset: null });

    expect(result.current).toEqual({ info: null, loading: false, error: null });
  });
});
