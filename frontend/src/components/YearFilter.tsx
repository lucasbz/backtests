import './YearFilter.css';

// The ticker list only has imported data within this range; keep it in sync
// with the actual imported dataset if that ever changes. There is no API
// endpoint to discover the range dynamically, so it's hardcoded here.
const EARLIEST_YEAR = 2010;
const LATEST_YEAR = 2026;

const YEARS = Array.from(
  { length: LATEST_YEAR - EARLIEST_YEAR + 1 },
  (_, index) => EARLIEST_YEAR + index,
);

export type YearFilterValue = 'all' | number;

export interface YearFilterProps {
  value: YearFilterValue;
  onChange: (value: YearFilterValue) => void;
}

/**
 * Lets the user restrict the ticker list (see `TickerList`) to symbols that
 * have imported data for a given year, via `GET /api/tickers?year=`.
 * "All" (the default) omits the `year` param entirely.
 */
export function YearFilter({ value, onChange }: YearFilterProps) {
  return (
    <div className="year-filter">
      <span className="year-filter__label">Year</span>
      <div className="year-filter__options" role="group" aria-label="Year">
        <button
          type="button"
          className={
            value === 'all'
              ? 'year-filter__option year-filter__option--active'
              : 'year-filter__option'
          }
          aria-pressed={value === 'all'}
          onClick={() => onChange('all')}
        >
          All
        </button>
        {YEARS.map((year) => (
          <button
            key={year}
            type="button"
            className={
              value === year
                ? 'year-filter__option year-filter__option--active'
                : 'year-filter__option'
            }
            aria-pressed={value === year}
            onClick={() => onChange(year)}
          >
            {year}
          </button>
        ))}
      </div>
    </div>
  );
}
