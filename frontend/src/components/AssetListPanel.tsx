import { useState } from 'react';
import { AssetList } from './AssetList';
import type { YearFilterValue } from './YearFilter';
import { buttonGhostClasses, chipClasses } from '../styles/ui';

export interface AssetListPanelProps {
  /** Restricts context for the "no assets" empty-state message. */
  year: YearFilterValue;
  /** The "Stocks" group's assets - already fetched/split by the caller. */
  stocks: string[];
  /** The "Others" group's assets - already fetched/split by the caller. */
  others: string[];
  selected: string | null;
  onSelect: (asset: string) => void;
  loading: boolean;
  error: string | null;
}

type Tab = 'stocks' | 'others';

/**
 * Left-edge panel that hosts both asset groups ("Stocks" and "Others") as
 * tabs instead of side-by-side columns, plus a single collapse/expand
 * toggle for the whole panel (positioned on the right edge, facing the
 * detail panel in `AssetBrowser`).
 *
 * Both `AssetList` instances stay mounted at all times - the inactive tab
 * is just visually hidden - so switching tabs never loses either list's
 * own local search text or scroll position, and so does collapsing the
 * panel.
 */
export function AssetListPanel({
  year,
  stocks,
  others,
  selected,
  onSelect,
  loading,
  error,
}: AssetListPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('stocks');
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div
      className={`flex flex-shrink-0 items-start border-r border-border pr-6 transition-[gap] duration-300 ${
        collapsed ? 'gap-0' : 'gap-3'
      }`}
    >
      <div
        className={`overflow-hidden transition-all duration-300 ${
          collapsed ? 'w-0 opacity-0' : 'w-[260px] opacity-100'
        }`}
      >
        <div className="mb-3 flex items-center gap-2" role="group" aria-label="Asset group">
          <button
            type="button"
            className={chipClasses(activeTab === 'stocks')}
            aria-pressed={activeTab === 'stocks'}
            onClick={() => setActiveTab('stocks')}
          >
            Stocks
          </button>
          <button
            type="button"
            className={chipClasses(activeTab === 'others')}
            aria-pressed={activeTab === 'others'}
            onClick={() => setActiveTab('others')}
          >
            Others
          </button>
        </div>

        <div className={activeTab === 'stocks' ? '' : 'hidden'}>
          <AssetList
            label="Stocks"
            year={year}
            items={stocks}
            selected={selected}
            onSelect={onSelect}
            loading={loading}
            error={error}
          />
        </div>
        <div className={activeTab === 'others' ? '' : 'hidden'}>
          <AssetList
            label="Others"
            year={year}
            items={others}
            selected={selected}
            onSelect={onSelect}
            loading={loading}
            error={error}
          />
        </div>
      </div>
      <button
        type="button"
        className={buttonGhostClasses(collapsed)}
        onClick={() => setCollapsed((value) => !value)}
        aria-pressed={collapsed}
        aria-label={collapsed ? 'Expand asset list' : 'Collapse asset list'}
      >
        {collapsed ? '›' : '‹'}
      </button>
    </div>
  );
}
