import { useCallback, useEffect, useRef, useState } from 'react';
import {
  CandlestickSeries,
  ColorType,
  HistogramSeries,
  LineSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
} from 'lightweight-charts';
import { useCandles } from '../hooks/useCandles';
import { useIndicators, type IndicatorSelection } from '../hooks/useIndicators';
import { buttonPrimaryClasses, chipClasses, errorBoxClasses, fieldClasses } from '../styles/ui';

export interface CandlestickChartProps {
  asset: string;
  start?: string;
  end?: string;
  // Indicator overlays (EMA/SMA) to show on top of the candles, in addition
  // to whatever the picker below the chart has toggled on. Defaults to none
  // - omitting this renders exactly as before this prop existed.
  indicators?: IndicatorSelection[];
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

// Colors for indicator overlay lines, deliberately distinct from
// `CHART_COLORS.gain`/`.loss` above - those specifically mean gain/loss
// elsewhere in the app, so an overlay line must never share a hue with
// them. Cycled by position in the current selection. Six entries (rather
// than just the three preset chips) since presets plus user-added custom
// indicators can now plausibly all be active at once.
const INDICATOR_LINE_COLORS = [
  '#60a5fa',
  '#fbbf24',
  '#c084fc',
  '#f472b6',
  '#22d3ee',
  '#818cf8',
];

// Fixed preset list the picker below the chart offers - chosen to line up
// with values already meaningful elsewhere in this app (`ema-trend-
// breakout`'s default period, `sma-crossover`'s default period).
const INDICATOR_PRESETS: { label: string; selection: IndicatorSelection }[] = [
  { label: 'EMA 8', selection: { type: 'ema', period: 8 } },
  { label: 'EMA 80', selection: { type: 'ema', period: 80 } },
  { label: 'SMA 20', selection: { type: 'sma', period: 20 } },
];

function indicatorKey(selection: IndicatorSelection): string {
  return `${selection.type}:${selection.period}`;
}

/**
 * Renders a candlestick chart (with a volume histogram beneath it) for
 * `asset` over `[start, end]`, via `useCandles`, plus optional EMA/SMA
 * overlay lines (via `useIndicators`) toggled through the preset chip
 * picker rendered above it. Mirrors `AssetInfoPanel`'s loading/error
 * handling; if `start`/`end` aren't available yet (e.g. the asset's info
 * hasn't loaded), nothing is fetched or rendered.
 */
export function CandlestickChart({ asset, start, end, indicators = [] }: CandlestickChartProps) {
  const { candles, loading, error } = useCandles(asset, start, end);

  const [selectedIndicators, setSelectedIndicators] = useState<IndicatorSelection[]>(indicators);
  const { series: indicatorSeries } = useIndicators(asset, start, end, selectedIndicators);

  const toggleIndicator = useCallback((selection: IndicatorSelection) => {
    setSelectedIndicators((current) => {
      const key = indicatorKey(selection);
      const isActive = current.some((entry) => indicatorKey(entry) === key);
      return isActive
        ? current.filter((entry) => indicatorKey(entry) !== key)
        : [...current, selection];
    });
  }, []);

  // Unlike `toggleIndicator` above, always ends with `selection` active -
  // used by the "Add indicator" form below, where re-adding an
  // already-active indicator should just leave it active rather than
  // toggling it off.
  const activateIndicator = useCallback((selection: IndicatorSelection) => {
    setSelectedIndicators((current) => {
      const key = indicatorKey(selection);
      if (current.some((entry) => indicatorKey(entry) === key)) return current;
      return [...current, selection];
    });
  }, []);

  // User-added indicators (via the "Add indicator" form below), each
  // rendered as its own permanent chip alongside `INDICATOR_PRESETS` -
  // "permanent" meaning toggling one off just deactivates it (removes its
  // line series) rather than removing the chip itself.
  const [customIndicators, setCustomIndicators] = useState<IndicatorSelection[]>([]);

  const [showAddIndicatorForm, setShowAddIndicatorForm] = useState(false);
  const [newIndicatorType, setNewIndicatorType] = useState<IndicatorSelection['type']>('ema');
  const [newIndicatorPeriod, setNewIndicatorPeriod] = useState('');

  const parsedNewIndicatorPeriod = Number(newIndicatorPeriod);
  const isNewIndicatorPeriodValid =
    newIndicatorPeriod.trim() !== '' &&
    Number.isInteger(parsedNewIndicatorPeriod) &&
    parsedNewIndicatorPeriod > 0;

  const resetAddIndicatorForm = useCallback(() => {
    setShowAddIndicatorForm(false);
    setNewIndicatorType('ema');
    setNewIndicatorPeriod('');
  }, []);

  const handleAddIndicator = useCallback(() => {
    if (!isNewIndicatorPeriodValid) return;

    const selection: IndicatorSelection = {
      type: newIndicatorType,
      period: parsedNewIndicatorPeriod,
    };
    const key = indicatorKey(selection);
    const alreadyKnown =
      INDICATOR_PRESETS.some(({ selection: preset }) => indicatorKey(preset) === key) ||
      customIndicators.some((entry) => indicatorKey(entry) === key);

    if (!alreadyKnown) {
      setCustomIndicators((current) => [...current, selection]);
    }
    activateIndicator(selection);
    resetAddIndicatorForm();
  }, [
    isNewIndicatorPeriodValid,
    newIndicatorType,
    parsedNewIndicatorPeriod,
    customIndicators,
    activateIndicator,
    resetAddIndicatorForm,
  ]);

  const chartRef = useRef<IChartApi | null>(null);
  const candleSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const volumeSeriesRef = useRef<ISeriesApi<'Histogram'> | null>(null);
  const lineSeriesRef = useRef<Map<string, ISeriesApi<'Line'>>>(new Map());

  // Bumped every time the container callback below (re)creates the chart,
  // so the line-series reconciliation effect further down - which otherwise
  // only depends on the indicator selection - also re-runs after a chart
  // recreation (e.g. `start`/`end` going missing then present again), when
  // `chartRef.current` is a fresh chart with no line series on it yet.
  const [chartMountTick, setChartMountTick] = useState(0);

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
      lineSeriesRef.current.clear();
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
    setChartMountTick((tick) => tick + 1);
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

  // Reconciles the line series actually on the chart against
  // `selectedIndicators`: removes ones no longer selected, creates ones
  // newly selected. Runs whenever the selection changes, or the chart
  // itself was just (re)created (`chartMountTick`) - `chartRef.current` may
  // still be null the very first time this runs (e.g. `start`/`end` aren't
  // available yet, so the container - and thus the chart - doesn't exist).
  const selectionKey = selectedIndicators.map(indicatorKey).join(',');
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;

    const selectedKeys = new Set(selectedIndicators.map(indicatorKey));

    for (const [key, lineSeries] of lineSeriesRef.current) {
      if (!selectedKeys.has(key)) {
        chart.removeSeries(lineSeries);
        lineSeriesRef.current.delete(key);
      }
    }

    selectedIndicators.forEach((selection, index) => {
      const key = indicatorKey(selection);
      if (lineSeriesRef.current.has(key)) return;

      const lineSeries = chart.addSeries(LineSeries, {
        color: INDICATOR_LINE_COLORS[index % INDICATOR_LINE_COLORS.length],
        lineWidth: 2,
        title: `${selection.type.toUpperCase()} ${selection.period}`,
      });
      lineSeriesRef.current.set(key, lineSeries);
    });
    // `selectedIndicators` is intentionally represented by `selectionKey`
    // here (see above) rather than listed directly.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chartMountTick, selectionKey]);

  // Populates each line series' data once it's been created above and its
  // points have arrived from `useIndicators` - the "setData on data change"
  // half of the same two-step split the candle/volume series use.
  useEffect(() => {
    indicatorSeries.forEach((entry) => {
      const lineSeries = lineSeriesRef.current.get(indicatorKey(entry));
      if (!lineSeries) return;

      lineSeries.setData(entry.points.map((point) => ({ time: point.date, value: point.value })));
    });
  }, [indicatorSeries]);

  if (!start || !end) {
    return null;
  }

  return (
    <section className="text-left">
      <div className="mb-3 flex flex-wrap items-center gap-2" role="group" aria-label="Indicators">
        {INDICATOR_PRESETS.map(({ label, selection }) => {
          const active = selectedIndicators.some(
            (entry) => indicatorKey(entry) === indicatorKey(selection),
          );
          return (
            <button
              key={label}
              type="button"
              className={chipClasses(active)}
              aria-pressed={active}
              onClick={() => toggleIndicator(selection)}
            >
              {label}
            </button>
          );
        })}

        {customIndicators.map((selection) => {
          const key = indicatorKey(selection);
          const label = `${selection.type.toUpperCase()} ${selection.period}`;
          const active = selectedIndicators.some((entry) => indicatorKey(entry) === key);
          return (
            <button
              key={key}
              type="button"
              className={chipClasses(active)}
              aria-pressed={active}
              onClick={() => toggleIndicator(selection)}
            >
              {label}
            </button>
          );
        })}

        <button
          type="button"
          className={chipClasses(showAddIndicatorForm)}
          aria-expanded={showAddIndicatorForm}
          onClick={() =>
            showAddIndicatorForm ? resetAddIndicatorForm() : setShowAddIndicatorForm(true)
          }
        >
          {showAddIndicatorForm ? '− Add indicator' : '+ Add indicator'}
        </button>

        {showAddIndicatorForm && (
          <div
            className="flex flex-wrap items-center gap-2"
            role="group"
            aria-label="Add custom indicator"
          >
            <label htmlFor="new-indicator-type" className="text-sm text-text">
              Type
            </label>
            <div className="w-20">
              <select
                id="new-indicator-type"
                className={`${fieldClasses} py-1.5`}
                value={newIndicatorType}
                onChange={(event) =>
                  setNewIndicatorType(event.target.value as IndicatorSelection['type'])
                }
              >
                <option value="ema">EMA</option>
                <option value="sma">SMA</option>
              </select>
            </div>

            <label htmlFor="new-indicator-period" className="text-sm text-text">
              Period
            </label>
            <div className="w-16">
              <input
                id="new-indicator-period"
                type="number"
                min="1"
                step="1"
                className={`${fieldClasses} py-1.5`}
                value={newIndicatorPeriod}
                onChange={(event) => setNewIndicatorPeriod(event.target.value)}
              />
            </div>

            <button
              type="button"
              className={`${buttonPrimaryClasses} px-3 py-1.5`}
              disabled={!isNewIndicatorPeriodValid}
              onClick={handleAddIndicator}
            >
              Add
            </button>

            <button
              type="button"
              className="rounded-xl border border-border bg-surface px-3 py-1.5 text-sm text-text transition-colors duration-150 hover:border-accent/40 hover:text-text-strong"
              onClick={resetAddIndicatorForm}
            >
              Cancel
            </button>
          </div>
        )}
      </div>

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
