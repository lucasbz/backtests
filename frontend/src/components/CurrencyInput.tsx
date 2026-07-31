import { useEffect } from 'react';
import { useCurrencyInput } from './useCurrencyInput';

export interface CurrencyInputProps {
  id?: string;
  /** Starting amount in major units, e.g. 10000 for "R$ 10.000,00". */
  initialValue?: number;
  /** Called with the plain decimal API-format string, e.g. "10000.00". */
  onValueChange: (value: string) => void;
  disabled?: boolean;
}

/**
 * A BRL-masked currency text input. Displays a formatted "R$ 10.000,00"
 * style value while reporting a plain decimal string ("10000.00") to the
 * caller via `onValueChange`, matching what the backtest API expects.
 */
export function CurrencyInput({
  id,
  initialValue = 0,
  onValueChange,
  disabled,
}: CurrencyInputProps) {
  const { display, value, onChange } = useCurrencyInput(initialValue);

  useEffect(() => {
    onValueChange(value);
  }, [value, onValueChange]);

  return (
    <input
      id={id}
      type="text"
      inputMode="numeric"
      autoComplete="off"
      value={display}
      onChange={onChange}
      disabled={disabled}
    />
  );
}
