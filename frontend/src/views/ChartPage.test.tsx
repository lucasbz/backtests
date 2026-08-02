import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ChartPage } from './ChartPage';
import { getAssetInfo, getAssets, type AssetInfo, type AssetsResponse } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getAssets: vi.fn(), getAssetInfo: vi.fn() };
});

// `CandlestickChart` drives `lightweight-charts` against a real <canvas>,
// which jsdom doesn't support - stubbed here the same way `AssetBrowser`
// would need to, rendering its props as text so tests can assert on them
// without exercising the chart library itself (see `CandlestickChart.test.tsx`
// for where that's covered directly).
vi.mock('../components/CandlestickChart', () => ({
  CandlestickChart: ({ asset, start, end }: { asset: string; start?: string; end?: string }) => (
    <div data-testid="candlestick-chart-stub">
      {asset} | {start} | {end}
    </div>
  ),
}));

const mockedGetAssets = vi.mocked(getAssets);
const mockedGetAssetInfo = vi.mocked(getAssetInfo);

function assetsResponse(overrides: Partial<AssetsResponse> = {}): AssetsResponse {
  return { stocks: ['PETR4', 'VALE3'], others: ['BOVA11'], ...overrides };
}

function assetInfo(overrides: Partial<AssetInfo> = {}): AssetInfo {
  return { asset: 'PETR4', earliest: '2010-01-01', latest: '2026-01-01', ...overrides };
}

beforeEach(() => {
  mockedGetAssets.mockReset();
  mockedGetAssetInfo.mockReset();
});

describe('ChartPage', () => {
  it('shows a placeholder when no asset is selected', async () => {
    mockedGetAssets.mockResolvedValue(assetsResponse());

    render(<ChartPage year="all" />);

    expect(await screen.findByText('Select an asset to get started.')).toBeInTheDocument();
    expect(screen.queryByTestId('candlestick-chart-stub')).not.toBeInTheDocument();
  });

  it('renders the chart for the selected asset with its info range', async () => {
    mockedGetAssets.mockResolvedValue(assetsResponse());
    mockedGetAssetInfo.mockResolvedValue(assetInfo());
    const user = userEvent.setup();

    render(<ChartPage year="all" />);

    await user.click(await screen.findByRole('button', { name: 'PETR4' }));

    await waitFor(() =>
      expect(screen.getByTestId('candlestick-chart-stub')).toHaveTextContent(
        'PETR4 | 2010-01-01 | 2026-01-01',
      ),
    );
    expect(mockedGetAssetInfo).toHaveBeenCalledWith('PETR4');
  });
});
