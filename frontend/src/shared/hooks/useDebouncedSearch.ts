// useDebouncedSearch — wraps a search keyword with a debounce delay so
// filter-by-keyword UIs don't recompute on every keystroke. Returns the
// debounced value; consumers compare against the live value for an
// "input is dirty" indicator.

import { useEffect, useState } from 'react';

export function useDebouncedSearch(value: string, delayMs = 200): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(value), delayMs);
    return () => window.clearTimeout(t);
  }, [value, delayMs]);
  return debounced;
}
