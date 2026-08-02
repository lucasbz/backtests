import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from './App';

// This is a light smoke test for the page-switching wiring only - each
// page's own behavior is covered by `AssetBrowser`'s dedicated tests (via
// its children) and `ChartPage.test.tsx`. Stub both out so this test isn't
// coupled to their internals (data fetching, `lightweight-charts`, etc.).
vi.mock('./views/AssetBrowser', () => ({
  AssetBrowser: () => <div>AssetBrowser content</div>,
}));
vi.mock('./views/ChartPage', () => ({
  ChartPage: () => <div>ChartPage content</div>,
}));

/**
 * Both pages stay mounted at all times (per `App`'s "keep both mounted,
 * just hide the inactive one" convention, matching `AssetListPanel`) - so
 * assert on the hidden wrapper's class rather than presence/absence in the
 * DOM.
 */
function wrapperClassName(contentText: string): string | undefined {
  return screen.getByText(contentText).parentElement?.className;
}

describe('App', () => {
  it('defaults to the Home page', () => {
    render(<App />);

    expect(wrapperClassName('AssetBrowser content')).toBe('');
    expect(wrapperClassName('ChartPage content')).toBe('hidden');
  });

  it('switches to the Charts page on nav click, keeping Home mounted but hidden', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getByRole('button', { name: 'Charts' }));

    expect(wrapperClassName('ChartPage content')).toBe('');
    expect(wrapperClassName('AssetBrowser content')).toBe('hidden');
  });

  it('switches back to Home on nav click', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getByRole('button', { name: 'Charts' }));
    await user.click(screen.getByRole('button', { name: 'Home' }));

    expect(wrapperClassName('AssetBrowser content')).toBe('');
    expect(wrapperClassName('ChartPage content')).toBe('hidden');
  });
});
