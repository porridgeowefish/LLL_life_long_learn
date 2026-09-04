package assistanttask

import (
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"os"
	"path/filepath"
	"time"
)

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
