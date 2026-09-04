// ESLint 9 flat config for the LLL frontend.
// Minimal recommended set: errors must stay at zero; stylistic reformatting
// is deliberately out of scope (prettier owns formatting).
import js from '@eslint/js';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  {
    ignores: ['dist/**', 'coverage/**', 'legacy/**', 'e2e/**', 'vite.config.*'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    rules: {
      // Existing code relies on these patterns; keep them as warnings so the
      // gate fails only on real errors, not legacy style.
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      'no-empty': ['warn', { allowEmptyCatch: true }],
    },
  },
);
