import { describe, it, expect, vi } from 'vitest';
import { streamAskAi, type AskFrame } from './askAi';

// Minimal ReadableStream stub that yields the given chunks once.
function fakeBody(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  let i = 0;
  return new ReadableStream({
    pull(controller) {
      if (i < chunks.length) {
        controller.enqueue(encoder.encode(chunks[i++]));
      } else {
        controller.close();
      }
    },
  });
}

describe('streamAskAi', () => {
  it('parses streamed frames via extractSSEData', async () => {
    const frames: AskFrame[] = [];
    const original = global.fetch;
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: fakeBody([
        'data: {"type":"text","content":"Hel"}\n\n',
        'data: {"type":"text","content":"lo"}\n\ndata: {"type":"done"}\n\n',
      ]),
    } as Response);
    await streamAskAi('p', 'c', { content: 'hi' }, { onFrame: (f) => frames.push(f) });
    global.fetch = original;

    expect(frames).toEqual([
      { type: 'text', content: 'Hel' },
      { type: 'text', content: 'lo' },
      { type: 'done', content: '' },
    ]);
  });

  it('throws on non-2xx', async () => {
    const original = global.fetch;
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      body: null,
      text: async () => '{"error":"ask-ai not configured"}',
    } as Response);
    await expect(
      streamAskAi('p', 'c', { content: 'hi' }, { onFrame: () => {} }),
    ).rejects.toThrow('ask-ai not configured');
    global.fetch = original;
  });
});
