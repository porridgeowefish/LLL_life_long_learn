// HTML-escape a string for safe innerHTML insertion. Replaces the legacy
// inline `esc` helper that was duplicated in 4 files.

const ESC_MAP: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
};

const ESC_REGEX = /[&<>"']/g;

export function esc(input: string): string {
  return input.replace(ESC_REGEX, (ch) => ESC_MAP[ch] ?? ch);
}
