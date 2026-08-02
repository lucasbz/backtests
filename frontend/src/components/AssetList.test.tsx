import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AssetList } from './AssetList';

function renderList(overrides: Partial<React.ComponentProps<typeof AssetList>> = {}) {
  return render(
    <AssetList
      label="Stocks"
      year="all"
      items={['PETR4', 'VALE3', 'ITUB4']}
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

    expect(screen.getByText('No assets match "nonexistent".')).toBeInTheDocument();
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
});
