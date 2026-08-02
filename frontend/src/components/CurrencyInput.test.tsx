import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CurrencyInput } from './CurrencyInput';

/**
 * `toHaveValue` on a text input compares the raw string, which includes
 * `Intl.NumberFormat`'s non-breaking space between "R$" and the amount -
 * normalize it the same way the other result-card tests do for
 * `toHaveTextContent`.
 */
function normalizeText(value: string): string {
  return value.replace(/ /g, ' ');
}

describe('CurrencyInput', () => {
  it('renders the initial value formatted as BRL', () => {
    render(<CurrencyInput initialValue={10000} onValueChange={vi.fn()} />);

    expect(normalizeText((screen.getByRole('textbox') as HTMLInputElement).value)).toBe(
      'R$ 10.000,00',
    );
  });

  it('defaults to zero when no initial value is given', () => {
    render(<CurrencyInput onValueChange={vi.fn()} />);

    expect(normalizeText((screen.getByRole('textbox') as HTMLInputElement).value)).toBe(
      'R$ 0,00',
    );
  });

  it('fills digits in from the right as cents while the user types', async () => {
    const user = userEvent.setup();
    render(<CurrencyInput initialValue={0} onValueChange={vi.fn()} />);

    const input = screen.getByRole('textbox');
    await user.type(input, '1234');

    expect(normalizeText((input as HTMLInputElement).value)).toBe('R$ 12,34');
  });

  it('strips non-digit characters typed into the field', async () => {
    const user = userEvent.setup();
    render(<CurrencyInput initialValue={0} onValueChange={vi.fn()} />);

    const input = screen.getByRole('textbox');
    await user.type(input, 'abc123');

    expect(normalizeText((input as HTMLInputElement).value)).toBe('R$ 1,23');
  });

  it('calls onValueChange with the plain decimal API-format string as the value changes', async () => {
    const onValueChange = vi.fn();
    const user = userEvent.setup();
    render(<CurrencyInput initialValue={0} onValueChange={onValueChange} />);

    await user.type(screen.getByRole('textbox'), '150000');

    expect(onValueChange).toHaveBeenLastCalledWith('1500.00');
  });

  it('calls onValueChange with the initial value on mount', () => {
    const onValueChange = vi.fn();
    render(<CurrencyInput initialValue={10000} onValueChange={onValueChange} />);

    expect(onValueChange).toHaveBeenCalledWith('10000.00');
  });

  it('treats an empty field (all digits deleted) as zero', async () => {
    const user = userEvent.setup();
    render(<CurrencyInput initialValue={100} onValueChange={vi.fn()} />);

    const input = screen.getByRole('textbox');
    await user.clear(input);

    expect(normalizeText((input as HTMLInputElement).value)).toBe('R$ 0,00');
  });

  it('can be disabled', () => {
    render(<CurrencyInput onValueChange={vi.fn()} disabled />);

    expect(screen.getByRole('textbox')).toBeDisabled();
  });
});
