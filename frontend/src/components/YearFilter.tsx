import { chipClasses } from '../styles/ui';

// The asset list only has imported data within this range; keep it in sync
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
 * Lets the user restrict the asset list (see `AssetList`) to symbols that
 * have imported data for a given year, via `GET /api/assets?year=`.
 * "All" (the default) omits the `year` param entirely.
 */
export function YearFilter({ value, onChange }: YearFilterProps) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-4">
      <div
        className="flex flex-wrap items-center gap-2"
        role="group"
        aria-label="Year"
      >
        <button
          type="button"
          className={chipClasses(value === 'all')}
          aria-pressed={value === 'all'}
          onClick={() => onChange('all')}
        >
          All
        </button>
        {YEARS.map((year) => (
          <button
            key={year}
            type="button"
            className={chipClasses(value === year)}
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
