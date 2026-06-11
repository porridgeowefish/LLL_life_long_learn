// sse.js — typed EventSource client.
// Dispatches typed events to handlers keyed by event name.

export function connectEvents(handlers, initialSnapshotHandler) {
  const es = new EventSource('/api/events');

  es.addEventListener('hello', e => {
    try {
      const data = JSON.parse(e.data);
      if (initialSnapshotHandler) initialSnapshotHandler(data);
    } catch (err) { console.error('hello parse:', err); }
  });

  Object.entries(handlers).forEach(([name, fn]) => {
    es.addEventListener(name, e => {
      try { fn(JSON.parse(e.data)); }
      catch (err) { console.error(name, 'parse:', err); }
    });
  });

  es.addEventListener('error', () => {
    // EventSource will auto-reconnect; nothing else to do.
  });

  return () => es.close();
}
