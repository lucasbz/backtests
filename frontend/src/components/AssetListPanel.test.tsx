import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AssetListPanel } from './AssetListPanel';

function renderPanel() {
  return render(
    <AssetListPanel
      year="all"
      stocks={['PETR4', 'VALE3']}
      others={['BOVA11']}
      selected={null}
      onSelect={vi.fn()}
      loading={false}
      error={null}
    />,
  );
}

/**
 * Both `AssetList` instances stay mounted at all times (per the component's
 * doc comment) - the inactive tab's group is just wrapped in a div toggled
 * between `''` and `'hidden'`. Since Tailwind's CSS isn't loaded in this
 * test environment, `hidden`'s `display: none` effect can't be observed via
 * `toBeVisible()`; assert on that wrapper `div`'s class list directly
 * instead, which is what actually drives the visual toggle.
 */
function tabWrapperClassName(assetButtonName: string): string | undefined {
  return screen.getByRole('button', { name: assetButtonName }).closest('aside')?.parentElement
    ?.className;
}

describe('AssetListPanel', () => {
  it('defaults to the "Stocks" tab, showing stock assets and hiding others', () => {
    renderPanel();

    expect(tabWrapperClassName('PETR4')).toBe('');
    expect(tabWrapperClassName('BOVA11')).toBe('hidden');
  });

  it("switches tabs on click, revealing the other group's assets", async () => {
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: 'Others' }));

    expect(tabWrapperClassName('BOVA11')).toBe('');
    expect(tabWrapperClassName('PETR4')).toBe('hidden');
  });

  it('keeps both asset lists mounted (not unmounted) across tab switches', async () => {
    const user = userEvent.setup();
    renderPanel();

    // Both are already in the DOM before any interaction, per the "always
    // mounted" behavior this test is meant to lock in.
    expect(screen.getByRole('button', { name: 'PETR4' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'BOVA11' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Others' }));

    expect(screen.getByRole('button', { name: 'PETR4' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'BOVA11' })).toBeInTheDocument();
  });

  it('marks the active tab as pressed via aria-pressed', async () => {
    const user = userEvent.setup();
    renderPanel();

    expect(screen.getByRole('button', { name: 'Stocks' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.getByRole('button', { name: 'Others' })).toHaveAttribute(
      'aria-pressed',
      'false',
    );

    await user.click(screen.getByRole('button', { name: 'Others' }));

    expect(screen.getByRole('button', { name: 'Stocks' })).toHaveAttribute(
      'aria-pressed',
      'false',
    );
    expect(screen.getByRole('button', { name: 'Others' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
  });

  it('toggles the collapse button label/aria-label/aria-pressed on click', async () => {
    const user = userEvent.setup();
    renderPanel();

    const toggle = screen.getByRole('button', { name: 'Collapse asset list' });
    expect(toggle).toHaveAttribute('aria-pressed', 'false');
    expect(toggle).toHaveTextContent('‹');

    await user.click(toggle);

    const expanded = screen.getByRole('button', { name: 'Expand asset list' });
    expect(expanded).toBe(toggle);
    expect(expanded).toHaveAttribute('aria-pressed', 'true');
    expect(expanded).toHaveTextContent('›');

    await user.click(expanded);

    expect(screen.getByRole('button', { name: 'Collapse asset list' })).toHaveAttribute(
      'aria-pressed',
      'false',
    );
  });

  it("collapsing hides the panel's asset-group container (width/opacity classes)", async () => {
    const user = userEvent.setup();
    renderPanel();

    const groupContainer = screen.getByRole('group', { name: 'Asset group' }).parentElement;
    expect(groupContainer?.className).toContain('w-[260px]');
    expect(groupContainer?.className).toContain('opacity-100');

    await user.click(screen.getByRole('button', { name: 'Collapse asset list' }));

    expect(groupContainer?.className).toContain('w-0');
    expect(groupContainer?.className).toContain('opacity-0');
  });

  it('preserves the previously active tab across a collapse/expand cycle', async () => {
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: 'Others' }));
    await user.click(screen.getByRole('button', { name: 'Collapse asset list' }));
    await user.click(screen.getByRole('button', { name: 'Expand asset list' }));

    expect(screen.getByRole('button', { name: 'Others' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(tabWrapperClassName('BOVA11')).toBe('');
  });
});
