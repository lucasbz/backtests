import { useCallback, useEffect, useRef } from 'react';
import {
  CandlestickSeries,
  ColorType,
  HistogramSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
} from 'lightweight-charts';
import { useCandles } from '../hooks/useCandles';
import { errorBoxClasses } from '../styles/ui';

export interface CandlestickChartProps {
  asset: string;
  start?: string;
  end?: string;
}

// `lightweight-charts` draws to a canvas, so it can't consume Tailwind
// classes or CSS custom properties directly - these mirror the design
// tokens declared in `src/index.css` (`@theme`), including the same
// gain/loss green/red used for P&L elsewhere (see `BacktestResultCard`).
const CHART_COLORS = {
  text: '#a2a0ae',
  border: 'rgba(255, 255, 255, 0.08)',
  gain: '#34d399',
  loss: '#f87171',
};

/**
 * Renders a candlestick chart (with a volume histogram beneath it) for
 * `asset` over `[start, end]`, via `useCandles`. Mirrors `AssetInfoPanel`'s
 * loading/error handling; if `start`/`end` aren't available yet (e.g. the
 * asset's info hasn't loaded), nothing is fetched or rendered.
 */
export function CandlestickChart({ asset, start, end }: CandlestickChartProps) {
  const { candles, loading, error } = useCandles(asset, start, end);

  const chartRef = useRef<IChartApi | null>(null);
  const candleSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const volumeSeriesRef = useRef<ISeriesApi<'Histogram'> | null>(null);

  // A ref callback (rather than `useRef` + a mount-only `useEffect`) so the
  // chart is (re)created whenever the container div itself mounts/unmounts -
  // which happens whenever `start`/`end` go from missing to present (see the
  // conditional render below), not just on the component's own first mount.
  const containerRef = useCallback((node: HTMLDivElement | null) => {
    if (chartRef.current) {
      chartRef.current.remove();
      chartRef.current = null;
      candleSeriesRef.current = null;
      volumeSeriesRef.current = null;
    }

    if (!node) return;

    const chart = createChart(node, {
      autoSize: true,
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor: CHART_COLORS.text,
      },
      grid: {
        vertLines: { color: CHART_COLORS.border },
        horzLines: { color: CHART_COLORS.border },
      },
      timeScale: { borderColor: CHART_COLORS.border },
      rightPriceScale: { borderColor: CHART_COLORS.border },
    });

    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: CHART_COLORS.gain,
      downColor: CHART_COLORS.loss,
      borderUpColor: CHART_COLORS.gain,
      borderDownColor: CHART_COLORS.loss,
      wickUpColor: CHART_COLORS.gain,
      wickDownColor: CHART_COLORS.loss,
    });

    const volumeSeries = chart.addSeries(HistogramSeries, {
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
    });
    // Confine the volume histogram to the bottom of the pane, underneath the
    // candlesticks, rather than sharing the full vertical scale with them.
    volumeSeries.priceScale().applyOptions({ scaleMargins: { top: 0.8, bottom: 0 } });

    chartRef.current = chart;
    candleSeriesRef.current = candleSeries;
    volumeSeriesRef.current = volumeSeries;
  }, []);

  useEffect(() => {
    if (!candles || !candleSeriesRef.current || !volumeSeriesRef.current) return;

    candleSeriesRef.current.setData(
      candles.map((candle) => ({
        time: candle.date,
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
      })),
    );

    volumeSeriesRef.current.setData(
      candles.map((candle) => ({
        time: candle.date,
        value: candle.volume,
        color: candle.close >= candle.open ? CHART_COLORS.gain : CHART_COLORS.loss,
      })),
    );
  }, [candles]);

  if (!start || !end) {
    return null;
  }

  return (
    <section className="text-left">
      {loading && <p className="text-sm text-text">Loading chart…</p>}

      {error && (
        <div className={errorBoxClasses} role="alert">
          {error}
        </div>
      )}

      <div
        ref={containerRef}
        data-testid="candlestick-chart-container"
        className="h-[400px] w-full overflow-hidden rounded-2xl border border-border bg-surface"
      />
    </section>
  );
}
