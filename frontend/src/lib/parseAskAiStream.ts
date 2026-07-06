/**
 * Split an SSE byte buffer into complete `data:` payloads plus the remaining
 * partial tail. Each SSE frame is terminated by a blank line (\n\n). Only
 * `data:` lines are extracted (event:/id:/comment lines are ignored).
 */
export function extractSSEData(buf: string): { payloads: string[]; rest: string } {
  const payloads: string[] = [];
  let rest = buf;
  let idx: number;
  while ((idx = rest.indexOf('\n\n')) >= 0) {
    const chunk = rest.slice(0, idx);
    rest = rest.slice(idx + 2);
    for (const line of chunk.split('\n')) {
      if (line.startsWith('data:')) {
        const payload = line.slice(5).trim();
        if (payload) payloads.push(payload);
      }
    }
  }
  return { payloads, rest };
}
