import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { CandlestickChart } from './CandlestickChart';
import { getCandles, type Candle } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getCandles: vi.fn() };
});

const mockedGetCandles = vi.mocked(getCandles);

// `lightweight-charts` manipulates a real <canvas> in ways jsdom doesn't
// support, so the whole module is faked here - these tests only assert the
// component wires it up (creates a chart, adds a candlestick + volume
// series, feeds them data, tears down on unmount), not that pixels render.
const setDataMock = vi.fn();
const volumeSetDataMock = vi.fn();
const applyPriceScaleOptionsMock = vi.fn();
const removeMock = vi.fn();
const addSeriesMock = vi.fn();
const createChartMock = vi.fn();

vi.mock('lightweight-charts', () => ({
  ColorType: { Solid: 'solid' },
  CandlestickSeries: 'CandlestickSeries',
  HistogramSeries: 'HistogramSeries',
  createChart: (...args: unknown[]) => createChartMock(...args),
}));

beforeEach(() => {
  mockedGetCandles.mockReset();
  setDataMock.mockReset();
  volumeSetDataMock.mockReset();
  applyPriceScaleOptionsMock.mockReset();
  removeMock.mockReset();
  addSeriesMock.mockReset();
  createChartMock.mockReset();

  addSeriesMock.mockImplementation((definition: string) => {
    if (definition === 'HistogramSeries') {
      return {
        setData: volumeSetDataMock,
        priceScale: () => ({ applyOptions: applyPriceScaleOptionsMock }),
      };
    }
    return { setData: setDataMock };
  });

  createChartMock.mockReturnValue({
    addSeries: addSeriesMock,
    remove: removeMock,
  });
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
  {
    date: '2024-01-03',
    open: 32.9,
    high: 33.5,
    low: 32.7,
    close: 32.6,
    avg: 33.0,
    quantity: 987600,
    volume: 32500000,
    trades: 654,
  },
];

describe('CandlestickChart', () => {
  it('renders nothing and makes no request when start/end are missing', () => {
    const { container } = render(<CandlestickChart asset="PETR4" />);

    expect(container).toBeEmptyDOMElement();
    expect(mockedGetCandles).not.toHaveBeenCalled();
    expect(createChartMock).not.toHaveBeenCalled();
  });

  it('shows a loading state while fetching', () => {
    mockedGetCandles.mockReturnValue(new Promise(() => {}));

    render(<CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />);

    expect(screen.getByText(/loading chart/i)).toBeInTheDocument();
  });

  it('shows an error state on failure', async () => {
    mockedGetCandles.mockRejectedValue(new Error('boom'));

    render(<CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />);

    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument());
    expect(screen.getByRole('alert')).toHaveTextContent(/unexpected error loading candles/i);
  });

  it('creates the chart with candlestick and volume series and feeds them data', async () => {
    mockedGetCandles.mockResolvedValue(sampleCandles);

    render(<CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />);

    expect(screen.getByTestId('candlestick-chart-container')).toBeInTheDocument();
    expect(createChartMock).toHaveBeenCalledTimes(1);
    expect(addSeriesMock).toHaveBeenCalledWith('CandlestickSeries', expect.any(Object));
    expect(addSeriesMock).toHaveBeenCalledWith('HistogramSeries', expect.any(Object));
    expect(applyPriceScaleOptionsMock).toHaveBeenCalled();

    await waitFor(() => expect(setDataMock).toHaveBeenCalled());

    expect(setDataMock).toHaveBeenCalledWith([
      { time: '2024-01-02', open: 32.5, high: 33.1, low: 32.1, close: 32.9 },
      { time: '2024-01-03', open: 32.9, high: 33.5, low: 32.7, close: 32.6 },
    ]);
    expect(volumeSetDataMock).toHaveBeenCalledWith([
      { time: '2024-01-02', value: 40600000, color: '#34d399' },
      { time: '2024-01-03', value: 32500000, color: '#f87171' },
    ]);
  });

  it('removes the chart on unmount', () => {
    mockedGetCandles.mockReturnValue(new Promise(() => {}));

    const { unmount } = render(
      <CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />,
    );

    expect(createChartMock).toHaveBeenCalledTimes(1);

    unmount();

    expect(removeMock).toHaveBeenCalledTimes(1);
  });
});
