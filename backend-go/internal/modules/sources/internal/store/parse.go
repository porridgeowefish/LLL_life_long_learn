package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xmz14/lll/backend-go/internal/modules/sources/internal/parser"
)

// OriginalPath resolves only the sealed original within its revision directory.
func (s *Store) OriginalPath(sourceID string) (Source, Revision, string, error) {
	source, revision, err := s.Get(sourceID)
	if err != nil {
		return source, revision, "", err
	}
	if !safeID(revision.RevisionID, "srev_") || source.Status == "deleted" || source.Status == "tombstoned" {
		return source, revision, "", os.ErrNotExist
	}
	rel := filepath.FromSlash(revision.Original.Path)
	if filepath.IsAbs(rel) || filepath.Dir(rel) != "original" || filepath.Base(rel) != revision.Original.Filename {
		return source, revision, "", errors.New("invalid original path")
	}
	base := filepath.Join(s.root, source.SourceID, "revisions", revision.RevisionID)
	baseReal, err := filepath.EvalSymlinks(base)
	if err != nil {
		return source, revision, "", err
	}
	original, err := filepath.EvalSymlinks(filepath.Join(base, rel))
	if err != nil {
		return source, revision, "", err
	}
	within, err := filepath.Rel(baseReal, original)
	if err != nil || filepath.IsAbs(within) || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return source, revision, "", errors.New("linked original outside revision")
	}
	return source, revision, original, nil
}

// CommitParsed validates the active revision and holds the same lock as deletion.
// A completed parser cannot resurrect an original tombstoned while it was running.
func (s *Store) CommitParsed(sourceID, revisionID, parser, content string, truncated bool) (Source, error) {
	return s.commitParsed(sourceID, revisionID, parser, content, truncated, nil)
}
func (s *Store) commitParsed(sourceID, revisionID, parserName, content string, truncated bool, structure *parser.Structure) (Source, error) {
	if !safeID(sourceID, "source_") || !safeID(revisionID, "srev_") || len(content) > 4<<20 || !utf8.ValidString(content) {
		return Source{}, errors.New("invalid parsed content")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.root, sourceID, "source.json")
	var source Source
	if err := readJSON(path, &source); err != nil {
		return source, err
	}
	if source.CurrentRevisionID != revisionID || source.Status != "processing" {
		return source, os.ErrNotExist
	}
	revisionPath := filepath.Join(s.root, sourceID, "revisions", revisionID, "revision.json")
	var revision Revision
	if err := readJSON(revisionPath, &revision); err != nil {
		return source, err
	}
	data := []byte(content)
	if err := s.publishDerivedLocked(&revision, map[string][]byte{"content.md": data}, map[string]string{"content.md": "text/markdown; charset=utf-8"}, structure, ""); err != nil {
		return source, err
	}
	revision.ParseTaskID = ""
	if err := writeJSON(revisionPath, revision); err != nil {
		return source, err
	}
	source.Status, source.FailureCode, source.Parser = "ready", "", parserName
	source.Retrieval = nil
	source.ExtractedBytes, source.ExtractionTruncated, source.UpdatedAt = int64(len(data)), truncated, time.Now().UTC()
	return source, writeJSON(path, source)
}

// BeginParse and FailParse guard every status transition against deletion.
func (s *Store) BeginParse(sourceID string) (Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !safeID(sourceID, "source_") {
		return Source{}, os.ErrNotExist
	}
	path := filepath.Join(s.root, sourceID, "source.json")
	var source Source
	if err := readJSON(path, &source); err != nil {
		return source, err
	}
	if source.Status == "tombstoned" || source.Status == "deleted" {
		return source, os.ErrNotExist
	}
	source.Status, source.FailureCode, source.UpdatedAt = "processing", "", time.Now().UTC()
	return source, writeJSON(path, source)
}
func (s *Store) FailParse(sourceID, revisionID, code string) (Source, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !safeID(sourceID, "source_") {
		return Source{}, os.ErrNotExist
	}
	path := filepath.Join(s.root, sourceID, "source.json")
	var source Source
	if err := readJSON(path, &source); err != nil {
		return source, err
	}
	if source.CurrentRevisionID != revisionID || source.Status != "processing" {
		return source, os.ErrNotExist
	}
	source.Status, source.FailureCode, source.UpdatedAt = "failed", code, time.Now().UTC()
	return source, writeJSON(path, source)
}
