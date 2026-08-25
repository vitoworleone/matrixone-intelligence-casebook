import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    root: __dirname,
    include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
    environment: 'happy-dom',
    globals: true,
    css: {
      modules: {
        classNameStrategy: 'non-scoped',
      },
    },
  },
});
