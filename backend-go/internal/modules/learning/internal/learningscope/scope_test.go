package learningscope

import (
	"testing"
	"time"
)

func TestSnapshotFromTopicCopiesCanonicalBoundary(t *testing.T) {
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	topic := Topic{
		ID: "classical-mechanics", Title: "经典力学", ChapterTitle: "力学",
		Goal:    "解释宏观低速物体的运动规律",
		InScope: []string{"牛顿运动定律"}, OutOfScope: []string{"连续介质力学"},
		Prerequisites: []string{"向量"}, OwnedConcepts: []string{"惯性参考系"},
		ReusedConcepts: []string{"微积分"},
	}
	scope := SnapshotFromTopic("physics", topic, now)
	topic.InScope[0] = "被修改"
	if scope.Status != StatusReady || scope.Source.TopicID != "classical-mechanics" {
		t.Fatalf("unexpected scope: %+v", scope)
	}
	if scope.InScope[0] != "牛顿运动定律" {
		t.Fatal("scope must copy rather than alias the catalog topic")
	}
}

func TestValidateCatalogRejectsConceptOwnedBySiblingTopics(t *testing.T) {
	catalog := Catalog{SchemaVersion: 1, Topics: []Topic{
		{ID: "a", Title: "A", ChapterTitle: "C", Goal: "A goal", InScope: []string{"A"}, OwnedConcepts: []string{"shared"}},
		{ID: "b", Title: "B", ChapterTitle: "C", Goal: "B goal", InScope: []string{"B"}, OwnedConcepts: []string{"shared"}},
	}}
	if err := ValidateCatalog(&catalog); err == nil || err.Error() != "duplicate_owned_concept" {
		t.Fatalf("got %v, want duplicate_owned_concept", err)
	}
}

func TestNewDraftIsStandaloneAndExplicitlyEmpty(t *testing.T) {
	scope := NewDraft("概率论", time.Unix(0, 0))
	if scope.Status != StatusDraft || scope.Source.Type != SourceStandalone || scope.InScope == nil {
		t.Fatalf("unexpected draft: %+v", scope)
	}
}

func TestValidateAgainstOverviewRequiresExactH4Order(t *testing.T) {
	catalog := Catalog{SchemaVersion: 1, Topics: []Topic{
		{ID: "a", Title: "A", ChapterTitle: "C", Goal: "A goal", InScope: []string{"A"}, OwnedConcepts: []string{"a"}},
		{ID: "b", Title: "B", ChapterTitle: "C", Goal: "B goal", InScope: []string{"B"}, OwnedConcepts: []string{"b"}},
	}}
	if err := ValidateCatalog(&catalog); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAgainstOverview(catalog, "### C\n#### A\n#### B\n"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAgainstOverview(catalog, "### C\n#### B\n#### A\n"); err == nil {
		t.Fatal("expected overview/catalog mismatch")
	}
}
