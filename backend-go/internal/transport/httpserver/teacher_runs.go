package httpserver

import (
	"context"
	"sync"

	"github.com/xmz14/lll/backend-go/internal/teacherservice"
)

// activeTeacherRun owns a provider response independently from any one HTTP
// connection. Browsers may disconnect and subscribe again without cancelling
// the teacher; only the explicit stop endpoint invokes cancel.
type activeTeacherRun struct {
	slug   string
	cancel context.CancelFunc

	mu     sync.Mutex
	frames []teacherservice.StreamFrame
	done   bool
	notify chan struct{}
}

func newActiveTeacherRun(slug string, cancel context.CancelFunc) *activeTeacherRun {
	return &activeTeacherRun{slug: slug, cancel: cancel, notify: make(chan struct{})}
}

func (run *activeTeacherRun) append(frame teacherservice.StreamFrame) {
	run.mu.Lock()
	run.frames = append(run.frames, frame)
	close(run.notify)
	run.notify = make(chan struct{})
	run.mu.Unlock()
}

func (run *activeTeacherRun) finish() {
	run.mu.Lock()
	if !run.done {
		run.done = true
		close(run.notify)
	}
	run.mu.Unlock()
}

func (run *activeTeacherRun) snapshot(after int) ([]teacherservice.StreamFrame, bool, <-chan struct{}) {
	run.mu.Lock()
	defer run.mu.Unlock()
	if after < 0 || after > len(run.frames) {
		after = 0
	}
	frames := append([]teacherservice.StreamFrame(nil), run.frames[after:]...)
	return frames, run.done, run.notify
}
