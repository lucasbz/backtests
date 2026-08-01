// Registers `@testing-library/jest-dom`'s matchers (e.g. `toBeDisabled`,
// `toHaveTextContent`) with Vitest's `expect`, and provides the matching
// TypeScript types. Loaded for every test file via `vitest.config.ts`'s
// `test.setupFiles`.
import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// `@testing-library/react` only auto-registers this via a global `afterEach`
// when `test.globals` is enabled; this project imports test APIs explicitly
// instead, so unmount each rendered tree by hand between tests to avoid
// leftover DOM from a previous test leaking into the next one's queries.
afterEach(() => {
  cleanup();
});
