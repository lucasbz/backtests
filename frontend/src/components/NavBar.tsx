import { chipClasses } from '../styles/ui';

/**
 * The app's two top-level pages, owned as plain local state by `App` (no
 * router is installed - see its module doc comment for why).
 */
export type Page = 'home' | 'charts';

export interface NavBarProps {
  active: Page;
  onNavigate: (page: Page) => void;
}

const NAV_ITEMS: { page: Page; label: string }[] = [
  { page: 'home', label: 'Home' },
  { page: 'charts', label: 'Charts' },
];

/**
 * Switches between the two top-level pages (`AssetBrowser` and
 * `ChartPage`), rendered next to the title in `App`'s header. There's no
 * routing library in this project, so this is plain client state, not
 * links - hence `<button>`s rather than `<a>`s, and `aria-current`
 * (matching `<nav>`'s usual semantics for "the current page") rather than
 * `aria-pressed` (which `AssetListPanel`'s tabs use for a non-navigational
 * toggle) to mark the active item.
 */
export function NavBar({ active, onNavigate }: NavBarProps) {
  return (
    <nav className="flex items-center gap-2" aria-label="Pages">
      {NAV_ITEMS.map(({ page, label }) => (
        <button
          key={page}
          type="button"
          className={chipClasses(active === page)}
          aria-current={active === page ? 'page' : undefined}
          onClick={() => onNavigate(page)}
        >
          {label}
        </button>
      ))}
    </nav>
  );
}
