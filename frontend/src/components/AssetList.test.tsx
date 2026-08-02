import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AssetList } from './AssetList';
import type { AssetEntry } from '../api/client';

const NO_VOLUME_ITEMS: AssetEntry[] = [
  { ticker: 'PETR4' },
  { ticker: 'VALE3' },
  { ticker: 'ITUB4' },
];

function renderList(overrides: Partial<React.ComponentProps<typeof AssetList>> = {}) {
  return render(
    <AssetList
      label="Stocks"
      year="all"
      items={NO_VOLUME_ITEMS}
      selected={null}
      onSelect={vi.fn()}
      loading={false}
      error={null}
      {...overrides}
    />,
  );
}

describe('AssetList', () => {
  it('renders every item when the search box is empty', () => {
    renderList();

    expect(screen.getByRole('button', { name: 'PETR4' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'VALE3' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'ITUB4' })).toBeInTheDocument();
  });

  it('narrows the list as the user types, case-insensitively', async () => {
    const user = userEvent.setup();
    renderList();

    await user.type(screen.getByRole('searchbox', { name: 'Search stocks' }), 'petr');

    expect(screen.getByRole('button', { name: 'PETR4' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'VALE3' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'ITUB4' })).not.toBeInTheDocument();
  });

  it('shows an empty-state message when the query matches nothing', async () => {
    const user = userEvent.setup();
    renderList();

    await user.type(screen.getByRole('searchbox', { name: 'Search stocks' }), 'nonexistent');

    expect(screen.getByText('No assets match the current filters.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'PETR4' })).not.toBeInTheDocument();
  });

  it('calls onSelect with the clicked asset', async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    renderList({ onSelect });

    await user.click(screen.getByRole('button', { name: 'VALE3' }));

    expect(onSelect).toHaveBeenCalledOnce();
    expect(onSelect).toHaveBeenCalledWith('VALE3');
  });

  it('shows a loading message and no search box or list while loading', () => {
    renderList({ loading: true });

    expect(screen.getByText('Loading assets…')).toBeInTheDocument();
    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'PETR4' })).not.toBeInTheDocument();
  });

  it('shows the error message as an alert instead of the list', () => {
    renderList({ error: 'network blew up' });

    expect(screen.getByRole('alert')).toHaveTextContent('network blew up');
    expect(screen.queryByRole('button', { name: 'PETR4' })).not.toBeInTheDocument();
  });

  it('shows a year-specific empty-state message when there are no items for a picked year', () => {
    renderList({ items: [], year: 2020 });

    expect(screen.getByText('No assets have data for 2020.')).toBeInTheDocument();
  });

  it('shows a generic empty-state message when there are no items and year is "all"', () => {
    renderList({ items: [], year: 'all' });

    expect(screen.getByText('No assets available.')).toBeInTheDocument();
  });

  describe('volume filter', () => {
    const VOLUME_ITEMS: AssetEntry[] = [
      { ticker: 'PETR4', volume: 1_000_000 },
      { ticker: 'VALE3', volume: 500_000 },
      { ticker: 'ITUB4', volume: 100_000 },
    ];

    it('is absent when no item in the group has volume data', () => {
      renderList({ items: NO_VOLUME_ITEMS });

      expect(screen.queryByRole('spinbutton', { name: 'Minimum volume for Stocks' })).not.toBeInTheDocument();
    });

    it('renders when at least one item has volume data', () => {
      renderList({ items: VOLUME_ITEMS, year: 2024 });

      expect(screen.getByRole('spinbutton', { name: 'Minimum volume for Stocks' })).toBeInTheDocument();
    });

    it('filters out items below the typed threshold, keeping items at or above it', async () => {
      const user = userEvent.setup();
      renderList({ items: VOLUME_ITEMS, year: 2024 });

      await user.type(
        screen.getByRole('spinbutton', { name: 'Minimum volume for Stocks' }),
        '500000',
      );

      expect(screen.getByRole('button', { name: 'PETR4' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'VALE3' })).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: 'ITUB4' })).not.toBeInTheDocument();
    });

    it('lets items without volume data through regardless of the threshold', async () => {
      const user = userEvent.setup();
      renderList({
        items: [...VOLUME_ITEMS, { ticker: 'NOVOL3' }],
        year: 2024,
      });

      await user.type(
        screen.getByRole('spinbutton', { name: 'Minimum volume for Stocks' }),
        '999999999',
      );

      expect(screen.getByRole('button', { name: 'NOVOL3' })).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: 'PETR4' })).not.toBeInTheDocument();
    });

    it('treats an empty/zero threshold as "no filter applied"', () => {
      renderList({ items: VOLUME_ITEMS, year: 2024 });

      expect(screen.getByRole('button', { name: 'PETR4' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'VALE3' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'ITUB4' })).toBeInTheDocument();
    });

    it('combines with the search text filter (AND, not OR)', async () => {
      const user = userEvent.setup();
      renderList({ items: VOLUME_ITEMS, year: 2024 });

      await user.type(screen.getByRole('searchbox', { name: 'Search stocks' }), 'vale');
      await user.type(
        screen.getByRole('spinbutton', { name: 'Minimum volume for Stocks' }),
        '600000',
      );

      // VALE3 matches the search text but not the volume threshold.
      expect(screen.queryByRole('button', { name: 'VALE3' })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: 'PETR4' })).not.toBeInTheDocument();
      expect(screen.getByText('No assets match the current filters.')).toBeInTheDocument();
    });
  });
});
