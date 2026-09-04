package assistanttask

import (
	"context"

	"errors"
	"sync"

	"sort"

	"github.com/xmz14/lll/backend-go/internal/workspace"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"

	agentexecution "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/execution"
)

const (
	GlobalConcurrency = 5
	UnitConcurrency   = 2
)

type EventEmitter interface{ Emit(string, any) }

type ExecutionService interface {
	StartTask(context.Context, agentexecution.TaskRequest) error
}

type Dispatcher struct {
	execution ExecutionService
	events    EventEmitter
	deps      Dependencies
	notify    chan struct{}
	stop      chan struct{}
	done      chan struct{}
	instance  string
	once      sync.Once
}

func NewDispatcher(execution ExecutionService, events EventEmitter) *Dispatcher {
	return &Dispatcher{execution: execution, events: events, notify: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{}), instance: idgen.New("dispatcher")}
}

func (d *Dispatcher) Configure(deps Dependencies) { d.deps = deps }

func (d *Dispatcher) Start() { go d.loop() }
func (d *Dispatcher) Close() { d.once.Do(func() { close(d.stop); <-d.done }) }
func (d *Dispatcher) Notify() {
	select {
	case d.notify <- struct{}{}:
	default:
	}
}

func (d *Dispatcher) loop() {
	defer close(d.done)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	d.reconcileLostRuns()
	for {
		d.reconcileLateResults()
		d.reconcilePartialDeliverables()
		d.dispatch()
		select {
		case <-d.stop:
			return
		case <-d.notify:
		case <-ticker.C:
		}
	}
}

type queuedTask struct {
	store *Store
	task  Task
}

func (d *Dispatcher) dispatch() {
	projects, err := workspace.IndexAll()
	if err != nil {
		return
	}
	var queued []queuedTask
	runningGlobal := 0
	runningUnits := map[string]int{}
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		store, err := New(project.Slug)
		if err != nil {
			continue
		}
		tasks, err := store.List("")
		if err != nil {
			continue
		}
		for _, task := range tasks {
			if task.Status == "running" {
				runningGlobal++
				runningUnits[task.UnitID]++
			}
			if task.Status == "queued" {
				queued = append(queued, queuedTask{store: store, task: task})
			}
		}
	}
	sort.Slice(queued, func(i, j int) bool { return queued[i].task.CreatedAt.Before(queued[j].task.CreatedAt) })
	for _, item := range queued {
		if runningGlobal >= GlobalConcurrency {
			break
		}
		if runningUnits[item.task.UnitID] >= UnitConcurrency {
			continue
		}
		runID := idgen.New("run")
		if err := prepareLaunchRecord(item.task, runID); err != nil {
			continue
		}
		started, err := item.store.Update(item.task.ID, func(task *Task) error {
			if task.Status != "queued" {
				return errors.New("task no longer queued")
			}
			now := time.Now().UTC()
			task.Status, task.Phase = "running", "preparing"
			task.AttemptIDs = append(task.AttemptIDs, runID)
			task.Lease = &Lease{RunID: runID, DispatcherInstance: d.instance, AcquiredAt: now, LastHeartbeatAt: now}
			return nil
		})
		if err != nil {
			continue
		}
		runningGlobal++
		runningUnits[started.UnitID]++
		d.emit(started)
		go d.execute(item.store, started, runID)
	}
}
