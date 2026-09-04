package assistanttask

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/agentexecution"
	"github.com/xmz14/lll/backend-go/internal/conversationstore"
	"github.com/xmz14/lll/backend-go/internal/idgen"
	assetstore "github.com/xmz14/lll/backend-go/internal/modules/assets"
	preferencestore "github.com/xmz14/lll/backend-go/internal/modules/preferences"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/workspace"
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
	notify    chan struct{}
	stop      chan struct{}
	done      chan struct{}
	instance  string
	once      sync.Once
}

func NewDispatcher(execution ExecutionService, events EventEmitter) *Dispatcher {
	return &Dispatcher{execution: execution, events: events, notify: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{}), instance: idgen.New("dispatcher")}
}

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

type inputManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	TaskID        string `json:"taskId"`
	Conversation  struct {
		ThroughSeq uint64 `json:"throughSeq"`
		Snapshot   string `json:"snapshot"`
		SHA256     string `json:"sha256"`
	} `json:"conversation"`
	Preferences struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"preferences"`
	Assets map[string]struct {
		VersionID string `json:"versionId"`
		Cursor    uint64 `json:"cursor"`
		Path      string `json:"path"`
		SHA256    string `json:"sha256"`
	} `json:"assets"`
	Sources   []struct{ SourceID, RevisionID, Path, SHA256 string } `json:"sources"`
	CreatedAt time.Time                                             `json:"createdAt"`
}

type resultManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	TaskID        string `json:"taskId"`
	RunID         string `json:"runId"`
	Summary       string `json:"summary"`
	Deliverables  []struct {
		Key        string `json:"key"`
		Descriptor string `json:"descriptor"`
	} `json:"deliverables"`
	AssetUpdates map[string]struct {
		Status    string `json:"status"`
		Candidate string `json:"candidate"`
		Code      string `json:"code"`
	} `json:"assetUpdates"`
	SourceUpdate *struct {
		SourceID   string `json:"sourceId"`
		RevisionID string `json:"revisionId"`
		Files      []struct {
			Key       string `json:"key"`
			Path      string `json:"path"`
			MediaType string `json:"mediaType"`
		} `json:"files"`
	} `json:"sourceUpdate,omitempty"`
}

type artifactDescriptor struct {
	SchemaVersion int      `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	EntryPoints   []string `json:"entryPoints"`
	Files         []struct {
		Path      string `json:"path"`
		MediaType string `json:"mediaType"`
		SHA256    string `json:"sha256"`
		Bytes     int64  `json:"bytes"`
	} `json:"files"`
}

type executorRecord struct {
	SchemaVersion int       `json:"schemaVersion"`
	PID           int       `json:"pid"`
	StartedAt     time.Time `json:"startedAt"`
}

type launchRecord struct {
	SchemaVersion int       `json:"schemaVersion"`
	TaskID        string    `json:"taskId"`
	RunID         string    `json:"runId"`
	CreatedAt     time.Time `json:"createdAt"`
}

const launchRecordGrace = 30 * time.Second
const resultManifestSettleDelay = time.Second

func stableResultManifest(attemptDir string) bool {
	info, err := os.Stat(filepath.Join(attemptDir, "workspace", "result-manifest.json"))
	return err == nil && info.Mode().IsRegular() && time.Since(info.ModTime()) >= resultManifestSettleDelay
}

func prepareLaunchRecord(task Task, runID string) error {
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		return err
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	if err := os.MkdirAll(attemptDir, 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(attemptDir, "launch.json"), launchRecord{SchemaVersion: 1, TaskID: task.ID, RunID: runID, CreatedAt: time.Now().UTC()})
}

type exitRecord struct {
	SchemaVersion int       `json:"schemaVersion"`
	PID           int       `json:"pid"`
	ExitCode      int       `json:"exitCode"`
	FinishedAt    time.Time `json:"finishedAt"`
}

type commitJournal struct {
	SchemaVersion int       `json:"schemaVersion"`
	TaskID        string    `json:"taskId"`
	RunID         string    `json:"runId"`
	State         string    `json:"state"`
	ResultHash    string    `json:"resultHash"`
	Status        string    `json:"status,omitempty"`
	Result        *Result   `json:"result,omitempty"`
	Failure       *Failure  `json:"failure,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (d *Dispatcher) execute(store *Store, task Task, runID string) {
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		d.fail(store, task.ID, "workspace-unavailable", "无法打开学习单元工作区", false)
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	workDir := filepath.Join(attemptDir, "workspace")
	for _, dir := range []string{"scratch", "deliverables", "asset-updates", "source-updates", "inputs"} {
		if err := os.MkdirAll(filepath.Join(workDir, dir), 0o755); err != nil {
			d.fail(store, task.ID, "attempt-prepare-failed", "无法准备助教任务目录", true)
			return
		}
	}
	manifest, bases, err := d.sealInputs(task, workDir)
	if err != nil {
		d.fail(store, task.ID, "input-seal-failed", "无法封存任务输入", true)
		return
	}
	manifestPath := filepath.Join(projectRoot, "assistant-tasks", task.ID, "input-manifest.json")
	if _, statErr := os.Stat(manifestPath); errors.Is(statErr, os.ErrNotExist) {
		if err := writeJSON(manifestPath, manifest); err != nil {
			d.fail(store, task.ID, "input-seal-failed", "无法写入输入清单", true)
			return
		}
	}
	envelope := map[string]any{"schemaVersion": 1, "taskId": task.ID, "runId": runID, "taskType": task.Type, "objective": task.Objective, "executor": map[string]any{"kind": "native-cli", "profile": "default"}, "inputManifest": "../../../input-manifest.json", "workspace": "workspace", "resultManifest": "workspace/result-manifest.json", "policy": map[string]any{"allowNetwork": true, "formalAssetWrites": false, "absoluteOutputPaths": false}}
	if err := writeJSON(filepath.Join(attemptDir, "envelope.json"), envelope); err != nil {
		d.fail(store, task.ID, "attempt-prepare-failed", "无法写入任务信封", true)
		return
	}
	prompt := buildTaskPrompt(task, runID)
	promptPath := filepath.Join(attemptDir, "prompt.md")
	if err := workspace.AtomicWriteFile(promptPath, []byte(prompt), 0o644); err != nil {
		d.fail(store, task.ID, "attempt-prepare-failed", "无法写入助教说明", true)
		return
	}
	_, _ = store.Update(task.ID, func(t *Task) error { t.Phase = "executing"; return nil })
	d.emitCurrent(store, task.ID)
	exit := make(chan int, 1)
	wrapper := filepath.Join(attemptDir, "launch-assistant.ps1")
	if d.execution == nil {
		d.fail(store, task.ID, "executor-launch-failed", "Agent 执行服务不可用", true)
		return
	}
	if err := d.execution.StartTask(context.Background(), agentexecution.TaskRequest{WorkspacePath: workDir, PromptPath: promptPath, WrapperPath: wrapper, OnExit: func(code int) { exit <- code }}); err != nil {
		d.fail(store, task.ID, "executor-launch-failed", "无法启动可见的 CLI 助教终端", true)
		return
	}
	var exitCode int
	select {
	case exitCode = <-exit:
	case <-d.stop:
		return
	}
	if exitCode == 130 || exitCode == -1073741510 {
		d.finish(store, task.ID, "cancelled", nil, &Failure{Code: "terminal-interrupted", Message: "任务已在 CLI 终端中止", Retryable: false, Suggestion: "如需重试，请在教师对话中重新确认任务，或在 CLI 中手动执行保存的 prompt.md。"})
		return
	}
	_, _ = store.Update(task.ID, func(t *Task) error { t.Phase = "validating"; return nil })
	d.emitCurrent(store, task.ID)
	resultPath := filepath.Join(workDir, "result-manifest.json")
	result, err := readResult(resultPath, task.ID, runID)
	if err != nil {
		d.fail(store, task.ID, "invalid-result-manifest", "CLI 未产生有效的 result-manifest.json", false)
		return
	}
	if err := validateResult(workDir, task, manifest, result); err != nil {
		d.fail(store, task.ID, "invalid-result-manifest", "CLI 结果清单或声明产物未通过工作区校验", false)
		return
	}
	if exitCode != 0 {
		d.fail(store, task.ID, "executor-exit-failed", fmt.Sprintf("CLI 以状态 %d 结束", exitCode), false)
		return
	}
	_, _ = store.Update(task.ID, func(t *Task) error { t.Phase = "committing"; return nil })
	d.emitCurrent(store, task.ID)
	status, taskResult, failure := d.commitResult(projectRoot, task, runID, workDir, manifest, bases, result)
	d.finish(store, task.ID, status, taskResult, failure)
}

func (d *Dispatcher) sealInputs(task Task, workDir string) (inputManifest, map[string]assetstore.Asset, error) {
	var manifest inputManifest
	manifest.SchemaVersion, manifest.TaskID, manifest.CreatedAt = 1, task.ID, time.Now().UTC()
	manifest.Assets = map[string]struct {
		VersionID string `json:"versionId"`
		Cursor    uint64 `json:"cursor"`
		Path      string `json:"path"`
		SHA256    string `json:"sha256"`
	}{}
	conversation, err := conversationstore.New(task.ProjectSlug)
	if err != nil {
		return manifest, nil, err
	}
	projection, err := conversation.SnapshotThrough(task.ConversationCutoffSeq)
	if err != nil {
		return manifest, nil, err
	}
	conversationBytes, _ := json.MarshalIndent(projection, "", "  ")
	conversationPath := filepath.Join(workDir, "inputs", "conversation.json")
	if err := workspace.AtomicWriteFile(conversationPath, conversationBytes, 0o644); err != nil {
		return manifest, nil, err
	}
	manifest.Conversation.ThroughSeq, manifest.Conversation.Snapshot, manifest.Conversation.SHA256 = task.ConversationCutoffSeq, "workspace/inputs/conversation.json", hashBytes(conversationBytes)
	preferences, err := preferencestore.Read()
	if err != nil {
		return manifest, nil, err
	}
	preferenceBytes := []byte(preferences.Content)
	preferencePath := filepath.Join(workDir, "inputs", preferencestore.Filename)
	if err := workspace.AtomicWriteFile(preferencePath, preferenceBytes, 0o444); err != nil {
		return manifest, nil, err
	}
	manifest.Preferences.Path, manifest.Preferences.SHA256 = "workspace/inputs/"+preferencestore.Filename, hashBytes(preferenceBytes)
	assets, err := assetstore.New(task.ProjectSlug)
	if err != nil {
		return manifest, nil, err
	}
	bases := map[string]assetstore.Asset{}
	for _, key := range assetstore.CoreKeys() {
		asset, err := assets.Get(key)
		if err != nil {
			return manifest, nil, err
		}
		bases[key] = asset
		rel := filepath.ToSlash(filepath.Join("workspace", "inputs", "assets", key, "current.md"))
		if err := workspace.AtomicWriteFile(filepath.Join(workDir, "inputs", "assets", key, "current.md"), []byte(asset.Content), 0o644); err != nil {
			return manifest, nil, err
		}
		manifest.Assets[key] = struct {
			VersionID string `json:"versionId"`
			Cursor    uint64 `json:"cursor"`
			Path      string `json:"path"`
			SHA256    string `json:"sha256"`
		}{asset.Meta.CurrentVersionID, asset.Meta.ConversationCursor, rel, hashBytes([]byte(asset.Content))}
	}
	sources, err := sourcestore.New(task.ProjectSlug)
	if err != nil {
		return manifest, nil, err
	}
	projectRoot, _ := workspace.ProjectRootForSlug(task.ProjectSlug)
	for _, sourceID := range task.SourceRefs {
		_, revision, err := sources.Get(sourceID)
		if err != nil {
			return manifest, nil, err
		}
		sourcePath := filepath.Join(projectRoot, "sources", sourceID, "revisions", revision.RevisionID)
		destination := filepath.Join(workDir, "inputs", "sources", sourceID, revision.RevisionID)
		if err := copyTree(sourcePath, destination); err != nil {
			return manifest, nil, err
		}
		manifest.Sources = append(manifest.Sources, struct{ SourceID, RevisionID, Path, SHA256 string }{sourceID, revision.RevisionID, filepath.ToSlash(filepath.Join("workspace", "inputs", "sources", sourceID, revision.RevisionID)), revision.Original.SHA256})
	}
	return manifest, bases, nil
}

func buildTaskPrompt(task Task, runID string) string {
	return fmt.Sprintf(`# LLL 助教任务

你是异步助教，负责重活；不要扮演正在聊天的教师。教师和学习者已经确认本任务。

- taskId: %s
- runId: %s
- taskType: %s
- 目标：%s

## 工作边界

1. 当前目录就是本次 attempt 的 workspace。只在此目录工作，不修改学习单元正式文件。
2. 输入位于 inputs/；conversation.json 是已封存对话，preferences.md 是学习者维护的全局偏好只读快照，assets/ 是三类资产基线，sources/ 是本次授权资料。不得修改 preferences.md，也不得推断或回写新偏好。
3. 可调研、运行代码、作图、生成任意必要文件。一般过程放 scratch/；值得保留的成果放 deliverables/<key>/。
4. 对引入、正文、练习的候选更新分别写入 asset-updates/intro/current.md、asset-updates/body/current.md、asset-updates/practice/current.md。没有变化就声明 unchanged，不要为了填满而修改。
5. 资料解析任务的派生文件放 source-updates/<revision-id>/。
6. 完成后必须写 result-manifest.json。它是清单，不是成果容器；不得使用绝对路径或 ..。

result-manifest.json 示例：

{
  "schemaVersion": 1,
  "taskId": %q,
  "runId": %q,
  "summary": "完成了什么，以及它对学习的意义",
  "deliverables": [{"key":"report","descriptor":"deliverables/report/artifact.json"}],
  "assetUpdates": {
    "intro": {"status":"unchanged","candidate":""},
    "body": {"status":"updated","candidate":"asset-updates/body/current.md"},
    "practice": {"status":"unchanged","candidate":""}
  }
}

artifact.json 必须完整声明每个保留文件，不能留下未声明文件。例如：

{
  "schemaVersion": 1,
  "kind": "report",
  "title": "实验报告",
  "description": "产出及其用途",
  "entryPoints": ["files/report.md"],
  "files": [
    {
      "path": "files/report.md",
      "mediaType": "text/markdown",
      "sha256": "sha256:<64位小写十六进制>",
      "bytes": 123
    }
  ]
}

sha256 和 bytes 必须按最终文件实际内容计算。SVG 不得含脚本、事件处理器、foreignObject 或外部资源。`, task.ID, runID, task.Type, task.Objective, task.ID, runID)
}

func readResult(path, taskID, runID string) (resultManifest, error) {
	var result resultManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	if result.SchemaVersion != 1 || result.TaskID != taskID || result.RunID != runID || strings.TrimSpace(result.Summary) == "" || len([]byte(result.Summary)) > 16<<10 {
		return result, errors.New("result identity or summary is invalid")
	}
	return result, nil
}

func validateResult(workDir string, task Task, manifest inputManifest, result resultManifest) error {
	core := map[string]bool{"intro": true, "body": true, "practice": true}
	if len(result.AssetUpdates) != len(core) {
		return errors.New("every core asset must have an explicit result")
	}
	for key, update := range result.AssetUpdates {
		if !core[key] {
			return fmt.Errorf("unknown core asset: %s", key)
		}
		candidate := strings.TrimSpace(update.Candidate)
		switch update.Status {
		case "updated":
			expected := filepath.ToSlash(filepath.Join("asset-updates", key, "current.md"))
			if filepath.ToSlash(candidate) != expected {
				return fmt.Errorf("invalid candidate for %s", key)
			}
			path, ok := safeJoin(workDir, candidate)
			if !ok || !regularFile(path) {
				return fmt.Errorf("missing candidate for %s", key)
			}
		case "unchanged":
			if candidate != "" {
				return fmt.Errorf("unchanged asset %s declares a candidate", key)
			}
		case "failed":
			if candidate != "" || strings.TrimSpace(update.Code) == "" || len(update.Code) > 128 {
				return fmt.Errorf("invalid failed asset result for %s", key)
			}
		default:
			return fmt.Errorf("invalid asset status for %s", key)
		}
	}
	seenKeys := map[string]bool{}
	for _, deliverable := range result.Deliverables {
		if !safeOutputKey(deliverable.Key) || seenKeys[deliverable.Key] {
			return errors.New("invalid or duplicate deliverable key")
		}
		seenKeys[deliverable.Key] = true
		expected := filepath.ToSlash(filepath.Join("deliverables", deliverable.Key, "artifact.json"))
		if filepath.ToSlash(deliverable.Descriptor) != expected {
			return fmt.Errorf("descriptor does not match deliverable key %s", deliverable.Key)
		}
		if _, err := validateArtifactDescriptor(workDir, deliverable); err != nil {
			return err
		}
	}
	if result.SourceUpdate != nil {
		if task.Type != "source-processing" || result.SourceUpdate.SourceID == "" || result.SourceUpdate.RevisionID == "" {
			return errors.New("source update is not allowed for this task")
		}
		allowed := false
		for _, source := range manifest.Sources {
			if source.SourceID == result.SourceUpdate.SourceID && source.RevisionID == result.SourceUpdate.RevisionID {
				allowed = true
			}
		}
		if !allowed {
			return errors.New("source update is outside sealed input")
		}
		seen := map[string]bool{}
		prefix := filepath.ToSlash(filepath.Join("source-updates", result.SourceUpdate.RevisionID)) + "/"
		for _, file := range result.SourceUpdate.Files {
			clean := filepath.ToSlash(file.Path)
			if !safeOutputKey(file.Key) || seen[file.Key] || !strings.HasPrefix(clean, prefix) || strings.TrimSpace(file.MediaType) == "" {
				return errors.New("invalid source-derived file declaration")
			}
			seen[file.Key] = true
			path, ok := safeJoin(workDir, file.Path)
			if !ok || !regularFile(path) {
				return errors.New("missing source-derived file")
			}
		}
	} else if task.Type == "source-processing" {
		return errors.New("source-processing task omitted source update")
	}
	return nil
}

func validateArtifactDescriptor(workDir string, deliverable struct {
	Key        string `json:"key"`
	Descriptor string `json:"descriptor"`
}) (artifactDescriptor, error) {
	var descriptor artifactDescriptor
	descriptorPath, ok := safeJoin(workDir, deliverable.Descriptor)
	if !ok || !regularFile(descriptorPath) {
		return descriptor, errors.New("deliverable descriptor is missing")
	}
	if err := readJSON(descriptorPath, &descriptor); err != nil {
		return descriptor, err
	}
	if descriptor.SchemaVersion != 1 || strings.TrimSpace(descriptor.Kind) == "" || strings.TrimSpace(descriptor.Title) == "" || len(descriptor.Files) == 0 {
		return descriptor, errors.New("invalid artifact descriptor identity")
	}
	dir := filepath.Dir(descriptorPath)
	declared := map[string]bool{}
	declaredSources := map[string]bool{}
	for _, file := range descriptor.Files {
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(file.Path)))
		if clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(filepath.FromSlash(file.Path)) || declared[clean] || strings.TrimSpace(file.MediaType) == "" {
			return descriptor, errors.New("invalid artifact file declaration")
		}
		path, _, ok := resolveArtifactFile(workDir, dir, clean)
		if !ok || !regularFile(path) {
			return descriptor, errors.New("declared artifact file is missing")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return descriptor, err
		}
		if file.Bytes != int64(len(data)) || file.SHA256 != hashBytes(data) {
			return descriptor, errors.New("artifact file size or hash mismatch")
		}
		if strings.EqualFold(file.MediaType, "image/svg+xml") && unsafeSVG(data) {
			return descriptor, errors.New("artifact SVG contains active content")
		}
		declared[clean] = true
		declaredSources[filepath.Clean(path)] = true
	}
	for _, entry := range descriptor.EntryPoints {
		if !declared[filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry)))] {
			return descriptor, errors.New("artifact entry point is not a declared file")
		}
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return errors.New("deliverable contains unsupported filesystem entry")
		}
		if info.IsDir() || path == descriptorPath {
			return nil
		}
		if !declaredSources[filepath.Clean(path)] {
			return errors.New("deliverable contains undeclared file")
		}
		return nil
	})
	return descriptor, err
}

// resolveArtifactFile accepts both the canonical descriptor-relative form
// (files/report.md) and the workspace-relative form that interactive CLIs
// naturally produce (deliverables/report/files/report.md). The latter may
// reference a related deliverable such as an animation; it remains confined
// to workspace/deliverables and is repackaged under the same safe path.
func resolveArtifactFile(workDir, descriptorDir, declared string) (source, packageRel string, ok bool) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(declared)))
	if clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(filepath.FromSlash(declared)) {
		return "", "", false
	}
	if strings.HasPrefix(clean, "deliverables/") {
		source, ok = safeJoin(workDir, clean)
		return source, clean, ok
	}
	source, ok = safeJoin(descriptorDir, clean)
	return source, clean, ok
}

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func safeOutputKey(key string) bool {
	if key == "" || len(key) > 80 {
		return false
	}
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func unsafeSVG(data []byte) bool {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	styleDepth := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
		switch value := token.(type) {
		case xml.Directive, xml.ProcInst:
			return true
		case xml.StartElement:
			name := strings.ToLower(value.Name.Local)
			if name == "script" || name == "foreignobject" || name == "iframe" || name == "object" || name == "embed" {
				return true
			}
			if name == "style" {
				styleDepth++
			}
			for _, attribute := range value.Attr {
				attrName := strings.ToLower(attribute.Name.Local)
				attrValue := strings.ToLower(strings.TrimSpace(attribute.Value))
				if strings.HasPrefix(attrName, "on") && len(attrName) > 2 {
					return true
				}
				if attrName == "style" && unsafeSVGStyle(attrValue) {
					return true
				}
				if attrName == "href" || attrName == "src" {
					if strings.HasPrefix(attrValue, "#") || strings.HasPrefix(attrValue, "data:image/png") || strings.HasPrefix(attrValue, "data:image/jpeg") || strings.HasPrefix(attrValue, "data:image/jpg") {
						continue
					}
					if attrValue != "" {
						return true
					}
				}
			}
		case xml.CharData:
			if styleDepth > 0 && unsafeSVGStyle(string(value)) {
				return true
			}
		case xml.EndElement:
			if strings.EqualFold(value.Name.Local, "style") && styleDepth > 0 {
				styleDepth--
			}
		}
	}
}

func unsafeSVGStyle(value string) bool {
	compact := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "\t", "")
	if strings.Contains(compact, "javascript:") || strings.Contains(compact, "expression(") || strings.Contains(compact, "@import") {
		return true
	}
	for start := 0; ; {
		index := strings.Index(compact[start:], "url(")
		if index < 0 {
			return false
		}
		index += start
		if !strings.HasPrefix(compact[index:], "url(#") && !strings.HasPrefix(compact[index:], "url('#") && !strings.HasPrefix(compact[index:], "url(\"#") {
			return true
		}
		start = index + len("url(")
	}
}

func (d *Dispatcher) commitResult(projectRoot string, task Task, runID, workDir string, manifest inputManifest, bases map[string]assetstore.Asset, result resultManifest) (string, *Result, *Failure) {
	resultBytes, _ := json.Marshal(result)
	journalPath := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID, "commit.json")
	journal := commitJournal{SchemaVersion: 1, TaskID: task.ID, RunID: runID, State: "committing", ResultHash: hashBytes(resultBytes), UpdatedAt: time.Now().UTC()}
	_ = writeJSON(journalPath, journal)
	taskResult := &Result{Summary: result.Summary, AssetUpdates: map[string]string{}}
	failed, updated := 0, 0
	assets, _ := assetstore.New(task.ProjectSlug)
	for _, key := range assetstore.CoreKeys() {
		update, ok := result.AssetUpdates[key]
		if !ok || update.Status == "unchanged" {
			_, _ = assets.AdvanceCursor(key, task.ConversationCutoffSeq)
			taskResult.AssetUpdates[key] = "unchanged"
			continue
		}
		if update.Status == "failed" {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			continue
		}
		candidatePath, ok := safeJoin(workDir, update.Candidate)
		if !ok {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			continue
		}
		candidate, err := os.ReadFile(candidatePath)
		if err != nil {
			taskResult.AssetUpdates[key] = "failed"
			failed++
			continue
		}
		base := bases[key]
		commit, err := assets.CommitCandidate(assetstore.CandidateInput{Key: key, BaseVersionID: base.Meta.CurrentVersionID, BaseContent: base.Content, CandidateContent: string(candidate), TaskID: task.ID, RunID: runID, FromSeq: base.Meta.ConversationCursor + 1, ThroughSeq: task.ConversationCutoffSeq, SourceRevisionIDs: sourceRevisionIDs(manifest), ChangeSummary: result.Summary})
		if err != nil || commit.Status != "updated" {
			taskResult.AssetUpdates[key] = "failed"
			failed++
		} else {
			taskResult.AssetUpdates[key] = "updated"
			updated++
			d.emitAsset(task.ProjectSlug, commit.Asset)
		}
	}
	for _, deliverable := range result.Deliverables {
		artifactID, err := d.commitGeneratedArtifact(projectRoot, task, runID, workDir, manifest, deliverable)
		if err != nil {
			failed++
			continue
		}
		taskResult.Deliverables = append(taskResult.Deliverables, artifactID)
		updated++
		d.emitGenerated(task.ProjectSlug, task.ID, artifactID)
	}
	if result.SourceUpdate != nil && task.Type == "source-processing" {
		files, media := map[string][]byte{}, map[string]string{}
		for _, file := range result.SourceUpdate.Files {
			path, ok := safeJoin(workDir, file.Path)
			if !ok || !strings.Contains(filepath.ToSlash(file.Path), "source-updates/") {
				failed++
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				failed++
				continue
			}
			files[file.Path[strings.LastIndex(filepath.ToSlash(file.Path), "/")+1:]] = data
			media[file.Path[strings.LastIndex(filepath.ToSlash(file.Path), "/")+1:]] = file.MediaType
		}
		sources, err := sourcestore.New(task.ProjectSlug)
		if err != nil {
			failed++
		} else if _, err := sources.CommitDerived(result.SourceUpdate.SourceID, result.SourceUpdate.RevisionID, files, media); err != nil {
			failed++
		} else {
			updatedSource, _ := sources.SetStatus(result.SourceUpdate.SourceID, "ready", "", task.ID)
			d.emitSource(task.ProjectSlug, updatedSource)
			updated++
		}
	}
	if failed > 0 && updated > 0 {
		failure := &Failure{Code: "partial-commit", Message: "部分成果已提交，另有输出未通过校验", Retryable: false, Suggestion: "可在 CLI 中查看本次 attempt 的 result-manifest.json 和输出目录。"}
		writeCompletedCommit(journalPath, journal, "partial", taskResult, failure)
		return "partial", taskResult, failure
	}
	if failed > 0 {
		failure := &Failure{Code: "commit-failed", Message: "助教产出未通过提交校验", Retryable: false, Suggestion: "建议在 CLI 中检查保存的 prompt、结果清单与相对路径。"}
		writeCompletedCommit(journalPath, journal, "failed", taskResult, failure)
		return "failed", taskResult, failure
	}
	writeCompletedCommit(journalPath, journal, "succeeded", taskResult, nil)
	return "succeeded", taskResult, nil
}

func writeCompletedCommit(path string, journal commitJournal, status string, result *Result, failure *Failure) {
	journal.State, journal.Status, journal.Result, journal.Failure, journal.UpdatedAt = "completed", status, result, failure, time.Now().UTC()
	_ = writeJSON(path, journal)
}

func (d *Dispatcher) fail(store *Store, taskID, code, message string, retryable bool) {
	d.finish(store, taskID, "failed", nil, &Failure{Code: code, Message: message, Retryable: retryable, Suggestion: "如要继续，建议在可见 CLI 中检查本次 attempt；需要重新发起时再回到教师对话确认。"})
}
func (d *Dispatcher) finish(store *Store, taskID, status string, result *Result, failure *Failure) {
	task, err := store.Update(taskID, func(t *Task) error {
		t.Status, t.Phase, t.Result, t.Failure, t.Lease = status, "", result, failure, nil
		return nil
	})
	if err == nil {
		d.emit(task)
	}
	d.Notify()
}
func (d *Dispatcher) emitCurrent(store *Store, taskID string) {
	if task, err := store.Get(taskID); err == nil {
		d.emit(task)
	}
}
func (d *Dispatcher) emit(task Task) {
	if d.events != nil {
		d.events.Emit("assistant-task-updated", map[string]any{"projectSlug": task.ProjectSlug, "task": task})
	}
}
func (d *Dispatcher) emitAsset(slug string, asset *assetstore.Asset) {
	if d.events != nil && asset != nil {
		d.events.Emit("learning-asset-updated", map[string]any{"projectSlug": slug, "asset": asset.Meta})
	}
}

func (d *Dispatcher) emitGenerated(slug, taskID, artifactID string) {
	if d.events != nil {
		d.events.Emit("generated-artifact-updated", map[string]any{"projectSlug": slug, "taskId": taskID, "artifactId": artifactID})
	}
}

func (d *Dispatcher) emitSource(slug string, source sourcestore.Source) {
	if d.events != nil && source.SourceID != "" {
		d.events.Emit("source-updated", map[string]any{"projectSlug": slug, "source": source})
	}
}

func (d *Dispatcher) reconcileLostRuns() {
	projects, _ := workspace.IndexAll()
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		store, err := New(project.Slug)
		if err != nil {
			continue
		}
		tasks, _ := store.List("running")
		for _, task := range tasks {
			d.reconcileRunningTask(store, task)
		}
	}
}

// reconcileLateResults repairs the durable boundary between a visible CLI and
// LLL. A terminal window may outlive the process handle observed by a previous
// server instance and write its final manifest later. A valid late result is
// therefore authoritative and is committed idempotently instead of remaining
// stranded inside the attempt workspace.
func (d *Dispatcher) reconcileLateResults() {
	projects, _ := workspace.IndexAll()
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		store, err := New(project.Slug)
		if err != nil {
			continue
		}
		tasks, _ := store.List("failed")
		for _, task := range tasks {
			if task.Failure == nil || (task.Failure.Code != "executor-state-lost" && task.Failure.Code != "invalid-result-manifest") || len(task.AttemptIDs) == 0 {
				continue
			}
			runID := task.AttemptIDs[len(task.AttemptIDs)-1]
			projectRoot, rootErr := workspace.ProjectRootForSlug(task.ProjectSlug)
			if rootErr != nil {
				continue
			}
			attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
			resultPath := filepath.Join(attemptDir, "workspace", "result-manifest.json")
			if !stableResultManifest(attemptDir) {
				continue
			}
			var journal commitJournal
			if readJSON(filepath.Join(attemptDir, "commit.json"), &journal) == nil && journal.State == "completed" {
				continue
			}
			workDir := filepath.Join(attemptDir, "workspace")
			manifest, bases, loadErr := loadSealedInputs(task, projectRoot, workDir)
			if loadErr != nil {
				continue
			}
			result, readErr := readResult(resultPath, task.ID, runID)
			if readErr != nil || validateResult(workDir, task, manifest, result) != nil {
				continue
			}
			_, _ = store.Update(task.ID, func(current *Task) error {
				current.Status, current.Phase, current.Result, current.Failure = "running", "committing", nil, nil
				return nil
			})
			d.emitCurrent(store, task.ID)
			status, taskResult, failure := d.commitResult(projectRoot, task, runID, workDir, manifest, bases, result)
			d.finish(store, task.ID, status, taskResult, failure)
		}
	}
}

func (d *Dispatcher) reconcilePartialDeliverables() {
	projects, _ := workspace.IndexAll()
	for _, project := range projects {
		if project.ProjectType != workspace.ProjectTypeSystemLearning {
			continue
		}
		store, err := New(project.Slug)
		if err != nil {
			continue
		}
		tasks, _ := store.List("partial")
		for _, task := range tasks {
			if task.Failure == nil || task.Failure.Code != "partial-commit" || task.Result == nil || len(task.AttemptIDs) == 0 {
				continue
			}
			runID := task.AttemptIDs[len(task.AttemptIDs)-1]
			projectRoot, rootErr := workspace.ProjectRootForSlug(task.ProjectSlug)
			if rootErr != nil {
				continue
			}
			workDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID, "workspace")
			manifest, _, loadErr := loadSealedInputs(task, projectRoot, workDir)
			result, readErr := readResult(filepath.Join(workDir, "result-manifest.json"), task.ID, runID)
			if loadErr != nil || readErr != nil || validateResult(workDir, task, manifest, result) != nil || result.SourceUpdate != nil {
				continue
			}
			cleanAssets := true
			for _, status := range task.Result.AssetUpdates {
				if status == "failed" {
					cleanAssets = false
				}
			}
			if !cleanAssets {
				continue
			}
			deliverables := append([]string(nil), task.Result.Deliverables...)
			allCommitted := true
			for _, declared := range result.Deliverables {
				artifactID, commitErr := d.commitGeneratedArtifact(projectRoot, task, runID, workDir, manifest, declared)
				if commitErr != nil {
					allCommitted = false
					continue
				}
				if !containsString(deliverables, artifactID) {
					deliverables = append(deliverables, artifactID)
				}
			}
			if !allCommitted {
				continue
			}
			recovered, updateErr := store.Update(task.ID, func(current *Task) error {
				current.Status, current.Failure = "succeeded", nil
				current.Result.Deliverables = deliverables
				return nil
			})
			if updateErr == nil {
				d.emit(recovered)
			}
		}
	}
}

func (d *Dispatcher) reconcileRunningTask(store *Store, task Task) {
	if len(task.AttemptIDs) == 0 {
		d.fail(store, task.ID, "executor-state-lost", "运行中的任务没有可恢复的 attempt", false)
		return
	}
	runID := task.AttemptIDs[len(task.AttemptIDs)-1]
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		d.fail(store, task.ID, "workspace-unavailable", "无法打开学习单元工作区", false)
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	var finished exitRecord
	if readJSON(filepath.Join(attemptDir, "exit.json"), &finished) == nil {
		go d.recoverAttempt(store, task, runID, finished.ExitCode)
		return
	}
	// Interactive CLIs intentionally remain open after finishing a turn. The
	// result manifest is the assistant protocol's completion signal; terminal
	// process exit only represents a later human action and must not gate commit.
	if stableResultManifest(attemptDir) {
		go d.recoverAttempt(store, task, runID, 0)
		return
	}
	var executor executorRecord
	if readJSON(filepath.Join(attemptDir, "executor.json"), &executor) == nil && processMatches(executor.PID, executor.StartedAt) {
		_, _ = store.Update(task.ID, func(current *Task) error {
			if current.Lease == nil {
				current.Lease = &Lease{RunID: runID, AcquiredAt: executor.StartedAt}
			}
			current.Lease.DispatcherInstance = d.instance
			current.Lease.LastHeartbeatAt = time.Now().UTC()
			return nil
		})
		go d.monitorRecovered(store, task, runID, executor.PID)
		return
	}
	var launch launchRecord
	if readJSON(filepath.Join(attemptDir, "launch.json"), &launch) == nil && launch.SchemaVersion == 1 && launch.TaskID == task.ID && launch.RunID == runID {
		deadline := launch.CreatedAt.Add(launchRecordGrace)
		if time.Now().UTC().Before(deadline) {
			go d.monitorLaunching(store, task, runID, deadline)
			return
		}
	}
	d.fail(store, task.ID, "executor-state-lost", "应用重启后确认原 CLI 进程已经结束，且没有可提交结果", false)
}

func (d *Dispatcher) monitorLaunching(store *Store, task Task, runID string, deadline time.Time) {
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-ticker.C:
			var finished exitRecord
			if readJSON(filepath.Join(attemptDir, "exit.json"), &finished) == nil {
				d.recoverAttempt(store, task, runID, finished.ExitCode)
				return
			}
			if stableResultManifest(attemptDir) {
				d.recoverAttempt(store, task, runID, 0)
				return
			}
			var executor executorRecord
			if readJSON(filepath.Join(attemptDir, "executor.json"), &executor) == nil && processMatches(executor.PID, executor.StartedAt) {
				d.monitorRecovered(store, task, runID, executor.PID)
				return
			}
			if !time.Now().UTC().Before(deadline) {
				d.fail(store, task.ID, "executor-state-lost", "CLI 启动记录存在，但在宽限期内没有出现可观察执行器", false)
				return
			}
		}
	}
}

func (d *Dispatcher) monitorRecovered(store *Store, task Task, runID string, pid int) {
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-ticker.C:
			var finished exitRecord
			if readJSON(filepath.Join(attemptDir, "exit.json"), &finished) == nil {
				d.recoverAttempt(store, task, runID, finished.ExitCode)
				return
			}
			if stableResultManifest(attemptDir) {
				d.recoverAttempt(store, task, runID, 0)
				return
			}
			var executor executorRecord
			if readJSON(filepath.Join(attemptDir, "executor.json"), &executor) != nil || !processMatches(pid, executor.StartedAt) {
				if _, err := os.Stat(filepath.Join(attemptDir, "workspace", "result-manifest.json")); err == nil {
					d.recoverAttempt(store, task, runID, 0)
				} else {
					d.fail(store, task.ID, "executor-state-lost", "恢复观察的 CLI 进程已经结束，且没有写出结果", false)
				}
				return
			}
			_, _ = store.Update(task.ID, func(current *Task) error {
				if current.Lease != nil {
					current.Lease.LastHeartbeatAt = time.Now().UTC()
				}
				return nil
			})
		}
	}
}

func (d *Dispatcher) recoverAttempt(store *Store, task Task, runID string, exitCode int) {
	current, err := store.Get(task.ID)
	if err != nil || IsTerminal(current.Status) {
		return
	}
	if exitCode == 130 || exitCode == -1073741510 {
		d.finish(store, task.ID, "cancelled", nil, &Failure{Code: "terminal-interrupted", Message: "任务已在 CLI 终端中止", Retryable: false, Suggestion: "如需重试，请在教师对话中重新确认任务，或在 CLI 中手动执行保存的 prompt.md。"})
		return
	}
	projectRoot, err := workspace.ProjectRootForSlug(task.ProjectSlug)
	if err != nil {
		d.fail(store, task.ID, "workspace-unavailable", "无法打开学习单元工作区", false)
		return
	}
	attemptDir := filepath.Join(projectRoot, "assistant-tasks", task.ID, "attempts", runID)
	workDir := filepath.Join(attemptDir, "workspace")
	var committed commitJournal
	if readJSON(filepath.Join(attemptDir, "commit.json"), &committed) == nil && committed.SchemaVersion == 1 && committed.TaskID == task.ID && committed.RunID == runID && committed.State == "completed" && IsTerminal(committed.Status) {
		d.finish(store, task.ID, committed.Status, committed.Result, committed.Failure)
		return
	}
	manifest, bases, err := loadSealedInputs(task, projectRoot, workDir)
	if err != nil {
		d.fail(store, task.ID, "input-recovery-failed", "无法恢复助教任务的封存输入", false)
		return
	}
	_, _ = store.Update(task.ID, func(current *Task) error { current.Phase = "validating"; return nil })
	d.emitCurrent(store, task.ID)
	result, err := readResult(filepath.Join(workDir, "result-manifest.json"), task.ID, runID)
	if err != nil {
		d.fail(store, task.ID, "invalid-result-manifest", "CLI 未产生有效的 result-manifest.json", false)
		return
	}
	if err := validateResult(workDir, task, manifest, result); err != nil {
		d.fail(store, task.ID, "invalid-result-manifest", "恢复的 CLI 结果清单或声明产物未通过工作区校验", false)
		return
	}
	if exitCode != 0 {
		d.fail(store, task.ID, "executor-exit-failed", fmt.Sprintf("CLI 以状态 %d 结束", exitCode), false)
		return
	}
	_, _ = store.Update(task.ID, func(current *Task) error { current.Phase = "committing"; return nil })
	d.emitCurrent(store, task.ID)
	status, taskResult, failure := d.commitResult(projectRoot, task, runID, workDir, manifest, bases, result)
	d.finish(store, task.ID, status, taskResult, failure)
}

func loadSealedInputs(task Task, projectRoot, workDir string) (inputManifest, map[string]assetstore.Asset, error) {
	var manifest inputManifest
	if err := readJSON(filepath.Join(projectRoot, "assistant-tasks", task.ID, "input-manifest.json"), &manifest); err != nil {
		return manifest, nil, err
	}
	if manifest.SchemaVersion != 1 || manifest.TaskID != task.ID {
		return manifest, nil, errors.New("sealed input identity mismatch")
	}
	bases := map[string]assetstore.Asset{}
	for key, entry := range manifest.Assets {
		content, err := os.ReadFile(filepath.Join(workDir, "inputs", "assets", key, "current.md"))
		if err != nil || hashBytes(content) != entry.SHA256 {
			return manifest, nil, errors.New("sealed asset input mismatch")
		}
		bases[key] = assetstore.Asset{Meta: assetstore.Meta{CurrentVersionID: entry.VersionID, ConversationCursor: entry.Cursor}, Content: string(content)}
	}
	return manifest, bases, nil
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func safeJoin(root, rel string) (string, bool) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", false
	}
	path := filepath.Join(root, clean)
	back, err := filepath.Rel(root, path)
	return path, err == nil && !strings.HasPrefix(back, "..")
}
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func sourceRevisionIDs(manifest inputManifest) []string {
	out := make([]string, 0, len(manifest.Sources))
	for _, source := range manifest.Sources {
		out = append(out, source.RevisionID)
	}
	return out
}

func artifactIDFor(taskID, runID, key string) string {
	sum := sha256.Sum256([]byte(taskID + "\x00" + runID + "\x00" + key))
	return "artifact_" + hex.EncodeToString(sum[:16])
}

func artifactOwnedBy(dir, taskID, runID string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "artifact.json"))
	if err != nil {
		return false
	}
	var artifact struct {
		Provenance struct {
			TaskID string `json:"taskId"`
			RunID  string `json:"runId"`
		} `json:"provenance"`
	}
	return json.Unmarshal(data, &artifact) == nil && artifact.Provenance.TaskID == taskID && artifact.Provenance.RunID == runID
}

func (d *Dispatcher) commitGeneratedArtifact(projectRoot string, task Task, runID, workDir string, manifest inputManifest, deliverable struct {
	Key        string `json:"key"`
	Descriptor string `json:"descriptor"`
}) (string, error) {
	descriptorPath, ok := safeJoin(workDir, deliverable.Descriptor)
	if !ok || !strings.HasPrefix(filepath.Clean(descriptorPath), filepath.Join(workDir, "deliverables")+string(os.PathSeparator)) {
		return "", errors.New("artifact descriptor is outside deliverables")
	}
	artifactID := artifactIDFor(task.ID, runID, deliverable.Key)
	target := filepath.Join(projectRoot, "assets", "generated", artifactID)
	if artifactOwnedBy(target, task.ID, runID) {
		return artifactID, nil
	}
	validated, err := validateArtifactDescriptor(workDir, deliverable)
	if err != nil {
		return "", err
	}
	staging := target + ".staging-" + runID
	_ = os.RemoveAll(staging)
	if err := stageArtifactPackage(workDir, filepath.Dir(descriptorPath), staging, validated); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	var artifact map[string]any
	if data, readErr := os.ReadFile(filepath.Join(staging, "artifact.json")); readErr != nil || json.Unmarshal(data, &artifact) != nil {
		_ = os.RemoveAll(staging)
		return "", errors.New("staged artifact descriptor is invalid")
	}
	artifact["schemaVersion"] = 1
	artifact["artifactId"] = artifactID
	artifact["provenance"] = map[string]any{"taskId": task.ID, "runId": runID, "conversationRange": map[string]uint64{"throughSeq": task.ConversationCutoffSeq}, "sourceRevisionIds": sourceRevisionIDs(manifest)}
	artifact["createdAt"] = time.Now().UTC()
	if err := writeJSON(filepath.Join(staging, "artifact.json"), artifact); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	if err := os.Rename(staging, target); err != nil {
		_ = os.RemoveAll(staging)
		if !artifactOwnedBy(target, task.ID, runID) {
			return "", err
		}
	}
	return artifactID, nil
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if errors.Is(err, os.ErrExist) {
			return err
		}
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		syncErr := out.Sync()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	})
}

func stageArtifactPackage(workDir, descriptorDir, destination string, descriptor artifactDescriptor) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	for _, file := range descriptor.Files {
		source, packageRel, ok := resolveArtifactFile(workDir, descriptorDir, file.Path)
		if !ok {
			return errors.New("invalid staged artifact path")
		}
		target, ok := safeJoin(destination, packageRel)
		if !ok {
			return errors.New("invalid artifact package path")
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := workspace.AtomicWriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return writeJSON(filepath.Join(destination, "artifact.json"), descriptor)
}
