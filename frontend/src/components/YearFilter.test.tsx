import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { YearFilter } from './YearFilter';

describe('YearFilter', () => {
  it('marks "All" as pressed when value is "all"', () => {
    render(<YearFilter value="all" onChange={vi.fn()} />);

    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('button', { name: '2024' })).toHaveAttribute('aria-pressed', 'false');
  });

  it('marks the matching year as pressed when value is a number', () => {
    render(<YearFilter value={2024} onChange={vi.fn()} />);

    expect(screen.getByRole('button', { name: '2024' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByRole('button', { name: '2023' })).toHaveAttribute('aria-pressed', 'false');
  });

  it('calls onChange with the numeric year when a year chip is clicked', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<YearFilter value="all" onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: '2024' }));

    expect(onChange).toHaveBeenCalledOnce();
    expect(onChange).toHaveBeenCalledWith(2024);
  });

  it('calls onChange with "all" when the All chip is clicked', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<YearFilter value={2024} onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: 'All' }));

    expect(onChange).toHaveBeenCalledOnce();
    expect(onChange).toHaveBeenCalledWith('all');
  });

  it('renders one chip for every year in the (hardcoded) 2010-2026 range, plus "All"', () => {
    render(<YearFilter value="all" onChange={vi.fn()} />);

    expect(screen.getAllByRole('button', { name: /^(All|\d{4})$/ })).toHaveLength(1 + (2026 - 2010 + 1));
  });
});
