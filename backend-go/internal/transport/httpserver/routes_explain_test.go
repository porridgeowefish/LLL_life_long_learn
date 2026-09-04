package server

import (
	"testing"
	"time"
)

// TestEmitInfographicReadyBroadcastsArtifactUpdated pins the SSE contract the
// frontend ExplainInfographic component subscribes to: on completion the
// infographic pipeline must emit "artifact-updated" carrying the project slug
// and the explain infographic artifact path. Drift here would silently break
// Stage 2 delivery.
func TestEmitInfographicReadyBroadcastsArtifactUpdated(t *testing.T) {
	ch, unsub := broadcaster.Subscribe()
	defer unsub()

	const slug = "test-proj"
	emitInfographicReady(slug)

	select {
	case ev := <-ch:
		if ev.Type != "artifact-updated" {
			t.Fatalf("event type = %q, want %q", ev.Type, "artifact-updated")
		}
		m, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload type = %T, want map[string]any", ev.Payload)
		}
		if m["slug"] != slug {
			t.Fatalf("payload slug = %v, want %q", m["slug"], slug)
		}
		if m["artifact"] != "explain/infographic.png" {
			t.Fatalf("payload artifact = %v, want %q", m["artifact"], "explain/infographic.png")
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive artifact-updated event within 1s")
	}
}

// TestEmitInfographicReadyDoesNotBlockWithoutSubscribers guards the pipeline:
// it calls emit unconditionally on completion, so Emit must never block when no
// SSE client is subscribed (Broadcaster drops to a buffered channel / silently).
func TestEmitInfographicReadyDoesNotBlockWithoutSubscribers(t *testing.T) {
	done := make(chan struct{})
	go func() {
		emitInfographicReady("solo-proj")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("emitInfographicReady blocked with no subscribers")
	}
}
