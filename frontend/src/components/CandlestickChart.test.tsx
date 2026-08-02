import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CandlestickChart } from './CandlestickChart';
import { getCandles, getIndicator, type Candle, type IndicatorPoint } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, getCandles: vi.fn(), getIndicator: vi.fn() };
});

const mockedGetCandles = vi.mocked(getCandles);
const mockedGetIndicator = vi.mocked(getIndicator);

// `lightweight-charts` manipulates a real <canvas> in ways jsdom doesn't
// support, so the whole module is faked here - these tests only assert the
// component wires it up (creates a chart, adds a candlestick + volume +
// line series, feeds them data, tears down on unmount), not that pixels
// render.
const setDataMock = vi.fn();
const volumeSetDataMock = vi.fn();
const applyPriceScaleOptionsMock = vi.fn();
const removeMock = vi.fn();
const removeSeriesMock = vi.fn();
const addSeriesMock = vi.fn();
const createChartMock = vi.fn();

// Keyed by the `title` option each line series is created with (e.g. "EMA
// 8"), so tests can assert what data a *specific* overlay line received,
// even when multiple are active at once.
let lineSeriesByTitle: Record<string, { setData: ReturnType<typeof vi.fn> }>;

vi.mock('lightweight-charts', () => ({
  ColorType: { Solid: 'solid' },
  CandlestickSeries: 'CandlestickSeries',
  HistogramSeries: 'HistogramSeries',
  LineSeries: 'LineSeries',
  createChart: (...args: unknown[]) => createChartMock(...args),
}));

beforeEach(() => {
  mockedGetCandles.mockReset();
  mockedGetIndicator.mockReset();
  setDataMock.mockReset();
  volumeSetDataMock.mockReset();
  applyPriceScaleOptionsMock.mockReset();
  removeMock.mockReset();
  removeSeriesMock.mockReset();
  addSeriesMock.mockReset();
  createChartMock.mockReset();
  lineSeriesByTitle = {};

  addSeriesMock.mockImplementation((definition: string, options?: { title?: string }) => {
    if (definition === 'HistogramSeries') {
      return {
        setData: volumeSetDataMock,
        priceScale: () => ({ applyOptions: applyPriceScaleOptionsMock }),
      };
    }
    if (definition === 'LineSeries') {
      const series = { setData: vi.fn() };
      lineSeriesByTitle[options?.title ?? ''] = series;
      return series;
    }
    return { setData: setDataMock };
  });

  createChartMock.mockReturnValue({
    addSeries: addSeriesMock,
    removeSeries: removeSeriesMock,
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

const ema8Points: IndicatorPoint[] = [
  { date: '2024-01-09', value: 32.4 },
  { date: '2024-01-10', value: 32.5 },
];

const sma20Points: IndicatorPoint[] = [{ date: '2024-01-30', value: 31.9 }];

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

  it('creates no line series and fetches no indicators when none are selected by default', async () => {
    mockedGetCandles.mockResolvedValue(sampleCandles);

    render(<CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />);

    await waitFor(() => expect(setDataMock).toHaveBeenCalled());

    expect(mockedGetIndicator).not.toHaveBeenCalled();
    expect(addSeriesMock).not.toHaveBeenCalledWith('LineSeries', expect.any(Object));
  });

  it('clicking an EMA chip fetches it and creates a line series with the fetched data', async () => {
    mockedGetCandles.mockResolvedValue(sampleCandles);
    mockedGetIndicator.mockResolvedValue(ema8Points);
    const user = userEvent.setup();

    render(<CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />);
    await waitFor(() => expect(setDataMock).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: 'EMA 8' }));

    expect(mockedGetIndicator).toHaveBeenCalledWith('ema', 'PETR4', '2024-01-01', '2024-12-31', 8);

    await waitFor(() =>
      expect(addSeriesMock).toHaveBeenCalledWith(
        'LineSeries',
        expect.objectContaining({ title: 'EMA 8' }),
      ),
    );

    await waitFor(() =>
      expect(lineSeriesByTitle['EMA 8'].setData).toHaveBeenCalledWith([
        { time: '2024-01-09', value: 32.4 },
        { time: '2024-01-10', value: 32.5 },
      ]),
    );
  });

  it('clicking an active chip again removes that line series', async () => {
    mockedGetCandles.mockResolvedValue(sampleCandles);
    mockedGetIndicator.mockResolvedValue(ema8Points);
    const user = userEvent.setup();

    render(<CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />);
    await waitFor(() => expect(setDataMock).toHaveBeenCalled());

    const chip = screen.getByRole('button', { name: 'EMA 8' });
    await user.click(chip);

    await waitFor(() =>
      expect(addSeriesMock).toHaveBeenCalledWith(
        'LineSeries',
        expect.objectContaining({ title: 'EMA 8' }),
      ),
    );
    const lineSeries = lineSeriesByTitle['EMA 8'];

    await user.click(chip);

    await waitFor(() => expect(removeSeriesMock).toHaveBeenCalledWith(lineSeries));
  });

  it('supports multiple active chips at once, each with its own line series', async () => {
    mockedGetCandles.mockResolvedValue(sampleCandles);
    mockedGetIndicator.mockImplementation((type) =>
      Promise.resolve(type === 'ema' ? ema8Points : sma20Points),
    );
    const user = userEvent.setup();

    render(<CandlestickChart asset="PETR4" start="2024-01-01" end="2024-12-31" />);
    await waitFor(() => expect(setDataMock).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: 'EMA 8' }));
    await user.click(screen.getByRole('button', { name: 'SMA 20' }));

    expect(mockedGetIndicator).toHaveBeenCalledWith('ema', 'PETR4', '2024-01-01', '2024-12-31', 8);
    expect(mockedGetIndicator).toHaveBeenCalledWith(
      'sma',
      'PETR4',
      '2024-01-01',
      '2024-12-31',
      20,
    );

    await waitFor(() => expect(lineSeriesByTitle['EMA 8']?.setData).toHaveBeenCalled());
    await waitFor(() => expect(lineSeriesByTitle['SMA 20']?.setData).toHaveBeenCalled());

    expect(lineSeriesByTitle['EMA 8'].setData).toHaveBeenCalledWith([
      { time: '2024-01-09', value: 32.4 },
      { time: '2024-01-10', value: 32.5 },
    ]);
    expect(lineSeriesByTitle['SMA 20'].setData).toHaveBeenCalledWith([
      { time: '2024-01-30', value: 31.9 },
    ]);
  });
});
