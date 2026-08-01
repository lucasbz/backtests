/**
 * Small set of reusable Tailwind utility-class strings shared across
 * components, so buttons/inputs/cards/chips stay visually consistent
 * without repeating (and slowly diverging) long class strings in every
 * file. Colors/spacing/radii referenced here come from the design tokens
 * declared in `src/index.css` (`@theme`), not one-off hex values.
 */

const focusRing = 'focus:outline-none focus:ring-2 focus:ring-accent/30';

/** Text/date/number inputs and `<select>`s. */
export const fieldClasses =
  `w-full rounded-xl border border-border bg-surface px-3 py-2 text-sm text-text-strong ` +
  `placeholder:text-muted transition-colors duration-150 hover:border-border-hover ` +
  `disabled:cursor-not-allowed disabled:opacity-50 ${focusRing} focus:border-accent`;

/** Primary call-to-action button (e.g. "Run comparison"). */
export const buttonPrimaryClasses =
  `inline-flex items-center justify-center rounded-xl bg-gradient-to-r from-accent-from ` +
  `to-accent-to px-5 py-2.5 text-sm font-medium text-white shadow-lg shadow-accent/20 ` +
  `transition-all duration-150 hover:brightness-110 active:scale-[0.98] ` +
  `disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:brightness-100 ${focusRing}`;

/** Low-emphasis icon/utility button (e.g. the collapse/expand toggles). */
export function buttonGhostClasses(active = false): string {
  return (
    `flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border text-sm ` +
    `shadow-sm transition-colors duration-150 disabled:cursor-not-allowed disabled:opacity-50 ${focusRing} ` +
    (active
      ? 'border-accent/50 bg-accent-soft text-text-strong'
      : 'border-border bg-surface text-text hover:border-accent/40 hover:text-text-strong')
  );
}

/** Pill/chip toggle, e.g. the year filter options. */
export function chipClasses(active: boolean): string {
  return (
    `inline-flex items-center justify-center rounded-full border px-3.5 py-1.5 text-xs ` +
    `font-medium transition-colors duration-150 ${focusRing} ` +
    (active
      ? 'border-accent/50 bg-accent-soft text-text-strong'
      : 'border-border bg-surface text-text hover:border-accent/40 hover:text-text-strong')
  );
}

/** Rounded, softly-elevated card surface. */
export const cardClasses =
  'rounded-2xl border border-border bg-surface p-5 shadow-lg shadow-black/20 ' +
  'transition-shadow duration-150 hover:shadow-xl hover:shadow-black/25';

/** Inline error/alert box. */
export const errorBoxClasses =
  'rounded-xl border border-error-border bg-error-bg px-3.5 py-3 text-sm text-error-text';
