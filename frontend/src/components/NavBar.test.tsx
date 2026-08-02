import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NavBar } from './NavBar';

describe('NavBar', () => {
  it('renders both nav items', () => {
    render(<NavBar active="home" onNavigate={vi.fn()} />);

    expect(screen.getByRole('button', { name: 'Home' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Charts' })).toBeInTheDocument();
  });

  it('marks the active item with aria-current, and not the inactive one', () => {
    render(<NavBar active="home" onNavigate={vi.fn()} />);

    expect(screen.getByRole('button', { name: 'Home' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('button', { name: 'Charts' })).not.toHaveAttribute('aria-current');
  });

  it('calls onNavigate("charts") when clicking Charts', async () => {
    const user = userEvent.setup();
    const onNavigate = vi.fn();
    render(<NavBar active="home" onNavigate={onNavigate} />);

    await user.click(screen.getByRole('button', { name: 'Charts' }));

    expect(onNavigate).toHaveBeenCalledWith('charts');
  });

  it('calls onNavigate("home") when clicking Home', async () => {
    const user = userEvent.setup();
    const onNavigate = vi.fn();
    render(<NavBar active="charts" onNavigate={onNavigate} />);

    await user.click(screen.getByRole('button', { name: 'Home' }));

    expect(onNavigate).toHaveBeenCalledWith('home');
  });
});
