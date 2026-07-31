import { useCallback, useMemo, useState } from 'react';

// pt-BR "cents-first" masked currency input: the underlying value is always
// tracked as an integer number of cents, and the displayed string /
// API-format string are both derived from it. This avoids ever having to
// parse a formatted "R$ 10.000,00" string back into a number.

const displayFormatter = new Intl.NumberFormat('pt-BR', {
  style: 'currency',
  currency: 'BRL',
});

function centsToDisplay(cents: number): string {
  return displayFormatter.format(cents / 100);
}

function centsToApiValue(cents: number): string {
  return (cents / 100).toFixed(2);
}

export interface UseCurrencyInputResult {
  /** Formatted value to render in the input, e.g. "R$ 10.000,00". */
  display: string;
  /** Plain decimal string for the API, e.g. "10000.00". */
  value: string;
  /** Change handler to wire up to the input's onChange. */
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
}

/**
 * Hand-rolled BRL currency mask. As the user types digits, they fill in
 * from the right as cents (typing "1234" produces "R$ 12,34"), matching the
 * conventional pt-BR masked currency input pattern.
 *
 * @param initialValue starting amount in major units, e.g. 10000 for
 *   "R$ 10.000,00".
 */
export function useCurrencyInput(initialValue = 0): UseCurrencyInputResult {
  const [cents, setCents] = useState(() => Math.round(initialValue * 100));

  const onChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const digitsOnly = event.target.value.replace(/\D/g, '');
    setCents(digitsOnly === '' ? 0 : Number(digitsOnly));
  }, []);

  const display = useMemo(() => centsToDisplay(cents), [cents]);
  const value = useMemo(() => centsToApiValue(cents), [cents]);

  return { display, value, onChange };
}
