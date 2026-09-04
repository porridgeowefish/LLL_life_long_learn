package assistanttask

import (
	agentexecution "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/execution"
	"os"
	"path/filepath"

	"context"

	"fmt"

	"errors"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"time"
)

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

func (d *Dispatcher) sealInputs(task Task, workDir string) (inputManifest, map[string]AssetSnapshot, error) {
	var manifest inputManifest
	manifest.SchemaVersion, manifest.TaskID, manifest.CreatedAt = 1, task.ID, time.Now().UTC()
	manifest.Assets = map[string]struct {
		VersionID string `json:"versionId"`
		Cursor    uint64 `json:"cursor"`
		Path      string `json:"path"`
		SHA256    string `json:"sha256"`
	}{}
	if d.deps.ConversationSnapshot == nil || d.deps.PreferencesSnapshot == nil || d.deps.ReadAsset == nil || d.deps.SealSource == nil {
		return manifest, nil, errors.New("assistant dispatcher dependencies are not configured")
	}
	conversationBytes, err := d.deps.ConversationSnapshot(task.ProjectSlug, task.ConversationCutoffSeq)
	if err != nil {
		return manifest, nil, err
	}
	conversationPath := filepath.Join(workDir, "inputs", "conversation.json")
	if err := workspace.AtomicWriteFile(conversationPath, conversationBytes, 0o644); err != nil {
		return manifest, nil, err
	}
	manifest.Conversation.ThroughSeq, manifest.Conversation.Snapshot, manifest.Conversation.SHA256 = task.ConversationCutoffSeq, "workspace/inputs/conversation.json", hashBytes(conversationBytes)
	preferenceFilename, preferenceBytes, err := d.deps.PreferencesSnapshot()
	if err != nil {
		return manifest, nil, err
	}
	preferencePath := filepath.Join(workDir, "inputs", preferenceFilename)
	if err := workspace.AtomicWriteFile(preferencePath, preferenceBytes, 0o444); err != nil {
		return manifest, nil, err
	}
	manifest.Preferences.Path, manifest.Preferences.SHA256 = "workspace/inputs/"+preferenceFilename, hashBytes(preferenceBytes)
	bases := map[string]AssetSnapshot{}
	for _, key := range d.deps.AssetKeys() {
		asset, err := d.deps.ReadAsset(task.ProjectSlug, key)
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
		}{asset.VersionID, asset.ConversationCursor, rel, hashBytes([]byte(asset.Content))}
	}
	for _, sourceID := range task.SourceRefs {
		destinationRoot := filepath.Join(workDir, "inputs", "sources", sourceID)
		source, err := d.deps.SealSource(task.ProjectSlug, sourceID, destinationRoot)
		if err != nil {
			return manifest, nil, err
		}
		manifest.Sources = append(manifest.Sources, struct{ SourceID, RevisionID, Path, SHA256 string }{sourceID, source.RevisionID, filepath.ToSlash(filepath.Join("workspace", "inputs", "sources", sourceID, source.RevisionID)), source.SHA256})
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
