package integration

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/xmz14/lll/backend-go/internal/modules/assets"
	"github.com/xmz14/lll/backend-go/internal/modules/assistant"
	"github.com/xmz14/lll/backend-go/internal/modules/preferences"
	"github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

func AssistantDependencies() assistant.Dependencies {
	return assistant.Dependencies{
		ConversationSnapshot: func(slug string, through uint64) ([]byte, error) {
			store, err := teacher.NewConversation(slug)
			if err != nil {
				return nil, err
			}
			projection, err := store.SnapshotThrough(through)
			if err != nil {
				return nil, err
			}
			return json.MarshalIndent(projection, "", "  ")
		},
		PreferencesSnapshot: func() (string, []byte, error) {
			snapshot, err := preferences.Read()
			return preferences.Filename, []byte(snapshot.Content), err
		},
		AssetKeys: assets.CoreKeys,
		ReadAsset: func(slug, key string) (assistant.AssetSnapshot, error) {
			store, err := assets.New(slug)
			if err != nil {
				return assistant.AssetSnapshot{}, err
			}
			asset, err := store.Get(key)
			return assistant.AssetSnapshot{VersionID: asset.Meta.CurrentVersionID, ConversationCursor: asset.Meta.ConversationCursor, Content: asset.Content}, err
		},
		AdvanceAsset: func(slug, key string, through uint64) error {
			store, err := assets.New(slug)
			if err != nil {
				return err
			}
			_, err = store.AdvanceCursor(key, through)
			return err
		},
		CommitAsset: func(slug string, input assistant.AssetCommitInput) (string, any, error) {
			store, err := assets.New(slug)
			if err != nil {
				return "", nil, err
			}
			result, err := store.CommitCandidate(assets.CandidateInput{Key: input.Key, BaseVersionID: input.BaseVersionID, BaseContent: input.BaseContent, CandidateContent: input.CandidateContent, TaskID: input.TaskID, RunID: input.RunID, FromSeq: input.FromSeq, ThroughSeq: input.ThroughSeq, SourceRevisionIDs: input.SourceRevisionIDs, ChangeSummary: input.ChangeSummary})
			if err != nil {
				return "", nil, err
			}
			return result.Status, result.Asset.Meta, nil
		},
		SealSource: func(slug, sourceID, destination string) (assistant.SourceSnapshot, error) {
			store, err := sources.New(slug)
			if err != nil {
				return assistant.SourceSnapshot{}, err
			}
			_, revision, err := store.Get(sourceID)
			if err != nil {
				return assistant.SourceSnapshot{}, err
			}
			root, err := workspace.ProjectRootForSlug(slug)
			if err != nil {
				return assistant.SourceSnapshot{}, err
			}
			source := filepath.Join(root, "sources", sourceID, "revisions", revision.RevisionID)
			if err := copyDirectory(source, filepath.Join(destination, revision.RevisionID)); err != nil {
				return assistant.SourceSnapshot{}, err
			}
			return assistant.SourceSnapshot{RevisionID: revision.RevisionID, SHA256: revision.Original.SHA256}, nil
		},
		CommitSourceDerived: func(slug, sourceID, revisionID, taskID string, files map[string][]byte, media map[string]string) (any, error) {
			store, err := sources.New(slug)
			if err != nil {
				return nil, err
			}
			if _, err := store.CommitDerived(sourceID, revisionID, files, media); err != nil {
				return nil, err
			}
			return store.SetStatus(sourceID, "ready", "", taskID)
		},
	}
}

func copyDirectory(source, destination string) error {
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
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
