// Formatting helpers — dates, sizes, durations. Centralized so we never
// have 4 different "5 minutes ago" implementations across pages.

import { formatDistanceToNow, format } from 'date-fns';
import { zhCN } from 'date-fns/locale';

export function formatRelativeTime(iso: string): string {
  try {
    const d = new Date(iso);
    return formatDistanceToNow(d, { addSuffix: true, locale: zhCN });
  } catch {
    return iso;
  }
}

export function formatDateTime(iso: string): string {
  try {
    const d = new Date(iso);
    return format(d, 'yyyy-MM-dd HH:mm', { locale: zhCN });
  } catch {
    return iso;
  }
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

export function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max - 1) + '…';
}
