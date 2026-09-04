// Package learning exposes the public boundary for learning scope, progress,
// activity, and live run status. Implementations remain private to the module.
package learning

import (
	"time"

	"github.com/xmz14/lll/backend-go/internal/modules/learning/internal/learningscope"
	"github.com/xmz14/lll/backend-go/internal/modules/learning/internal/progressstore"
	"github.com/xmz14/lll/backend-go/internal/modules/learning/internal/runprogress"
)

const (
	SchemaVersion         = learningscope.SchemaVersion
	PolicyVersion         = progressstore.PolicyVersion
	LearningPolicyVersion = progressstore.LearningPolicyVersion
	StatusDraft           = learningscope.StatusDraft
	StatusReady           = learningscope.StatusReady
	SourceStandalone      = learningscope.SourceStandalone
	SourceDisciplineMap   = learningscope.SourceDisciplineMap
)

type Source = learningscope.Source
type Topic = learningscope.Topic
type Catalog = learningscope.Catalog
type Scope = learningscope.Scope

type ProgressEvent = progressstore.Event
type ProgressSummary = progressstore.Summary
type ProgressStore = progressstore.Store
type ActivityEvent = progressstore.ActivityEvent
type ActivityDay = progressstore.ActivityDay
type ActivitySummary = progressstore.ActivitySummary

type RunStatus = runprogress.Status
type RunStore = runprogress.Store

var NewDraft = learningscope.NewDraft
var SnapshotFromTopic = learningscope.SnapshotFromTopic
var ReadCatalog = learningscope.ReadCatalog
var MarshalScope = learningscope.MarshalScope
var EmptyCatalog = learningscope.EmptyCatalog
var FindTopic = learningscope.FindTopic
var ValidateCatalog = learningscope.ValidateCatalog
var ValidateAgainstOverview = learningscope.ValidateAgainstOverview

var NewProgressStore = progressstore.New
var NewRunStore = runprogress.New

func Aggregate(projectSlug string, weeks int, now time.Time) (ActivitySummary, error) {
	return progressstore.Aggregate(projectSlug, weeks, now)
}
