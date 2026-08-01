import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config.ts';

// Kept separate from `vite.config.ts` rather than merging the `test` block
// in there directly, so the dev/build config stays focused on Vite-only
// concerns and doesn't need to know about the test runner at all.
export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      css: false,
    },
  }),
);
