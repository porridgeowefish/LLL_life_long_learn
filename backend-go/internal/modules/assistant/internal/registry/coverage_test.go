package agentregistry_test

import (
	"strings"
	"testing"

	agentregistry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
)

// TestPrimitivesCoverage_AllPresent is the regression gate for the
// deepthink-replacement initiative. Every mechanism that was once
// delivered by the legacy deepthink skill must resolve to an on-disk
// primitive file under agents/primitives/. If a primitive file goes
// missing, this test fails and CI blocks the merge.
//
// See docs/00-product-and-architecture/AGENT_PRIMITIVES.md for the
// authoritative ownership table.
func TestPrimitivesCoverage_AllPresent(t *testing.T) {
	requiredPrimitives := []string{
		// Explain required
		"mece_decompose",
		"first_principles",
		"concept_graph",
		"misconception",
		"boundary_map",
		"research_question_frame",
		"prerequisite_scaffold",
		// Explain optional
		"analogy",
		// Intro required (primitive ready, agent pending)
		"knowledge_anchor",
		// Practice required (primitive ready, agent pending)
		"transfer",
	}
	for _, name := range requiredPrimitives {
		if !agentregistry.PrimitiveExists(name) {
			t.Errorf("required primitive missing: %s (expected at agents/primitives/%s.md)", name, name)
		}
	}
}

// TestPrimitivesCoverage_ExplainDeclaresAll asserts the explain agent
// registry entry declares the five required primitives mandated by the
// AGENT_PRIMITIVES.md ownership table.
func TestPrimitivesCoverage_ExplainDeclaresAll(t *testing.T) {
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	a, ok := reg.Get("explain")
	if !ok {
		t.Fatal("explain agent not registered")
	}
	if strings.TrimSpace(a.UserStory) == "" {
		t.Error("explain agent missing userStory")
	}
	expected := []string{
		"research_question_frame",
		"prerequisite_scaffold",
		"misconception",
		"boundary_map",
	}
	for _, e := range expected {
		found := false
		for _, r := range a.Primitives.Required {
			if r == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("explain agent missing required primitive: %s", e)
		}
	}
	// Optional list must contain at least analogy.
	hasAnalogy := false
	for _, r := range a.Primitives.Optional {
		if r == "analogy" {
			hasAnalogy = true
		}
	}
	if !hasAnalogy {
		t.Error("explain agent missing optional primitive 'analogy'")
	}
}

func TestEncyclopediaAgentIsRegisteredForDisciplineMaps(t *testing.T) {
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	agent, ok := reg.Get("encyclopedia")
	if !ok {
		t.Fatal("encyclopedia agent not registered")
	}
	if len(agent.AllowedZones) != 0 {
		t.Fatalf("encyclopedia agent must not be a five-zone agent: %v", agent.AllowedZones)
	}
	if len(agent.AllowedProjectTypes) != 1 || agent.AllowedProjectTypes[0] != "discipline-map" {
		t.Fatalf("unexpected project types: %v", agent.AllowedProjectTypes)
	}
	for _, required := range []string{
		"百科式而非教程式",
		"先规划知识架构",
		"## 主要研究领域与知识架构",
		"### <大章节一>",
		"#### <可独立深入学习的细分主题一>",
		"百科 Agent 不生成学习顺序或任务清单",
		"标题就是学科目录的事实源",
		"每个四级标题旁增加",
	} {
		if !strings.Contains(agent.CharterText, required) {
			t.Errorf("encyclopedia charter missing %q", required)
		}
	}
}

// TestPrimitivesCoverage_LimitsEnforced asserts the explain agent stays
// within the MaxRequiredPrimitives cap, preventing regression where the
// agent slowly absorbs every mechanism and recreates a "god skill".
func TestPrimitivesCoverage_LimitsEnforced(t *testing.T) {
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	for _, a := range reg.List() {
		if len(a.Primitives.Required) > agentregistry.MaxRequiredPrimitives {
			t.Errorf("agent %s has %d required primitives > cap %d",
				a.ID, len(a.Primitives.Required), agentregistry.MaxRequiredPrimitives)
		}
	}
}

// TestPrimitivesCoverage_ActiveLearningAgentsShipped asserts that the
// distribution promise is real, not architectural. The active learning
// agents explain, intro, and practice must each
// must own a distinct primitive set. This test exists specifically to
// prevent regression where someone says "distribution" but ships only
// one agent with primitive files declared for the other four.
func TestPrimitivesCoverage_ActiveLearningAgentsShipped(t *testing.T) {
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	expected := map[string][]string{
		"explain":  {"research_question_frame", "prerequisite_scaffold", "misconception", "boundary_map"},
		"intro":    {"prerequisite_scaffold", "boundary_map"},
		"practice": {"transfer"},
	}
	for id, requiredPrims := range expected {
		a, ok := reg.Get(id)
		if !ok {
			t.Errorf("agent %s is NOT registered — distribution is incomplete", id)
			continue
		}
		if strings.TrimSpace(a.UserStory) == "" {
			t.Errorf("agent %s missing userStory", id)
		}
		for _, want := range requiredPrims {
			found := false
			for _, got := range a.Primitives.Required {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("agent %s missing required primitive %s", id, want)
			}
		}
	}
}

// TestPrimitivesCoverage_NoPrimitiveIsOrphaned asserts that every
// primitive .md file on disk is owned by at least one registered agent.
// An orphan primitive is dead weight — a file that no agent will ever
// load. This catches the regression where someone writes a primitive
// but forgets to wire it into a charter.
func TestPrimitivesCoverage_NoPrimitiveIsOrphaned(t *testing.T) {
	allPrimitives := []string{
		"mece_decompose", "first_principles", "concept_graph",
		"misconception", "boundary_map", "analogy",
		"knowledge_anchor", "transfer",
		"research_question_frame", "prerequisite_scaffold",
	}
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	owned := map[string]bool{}
	for _, a := range reg.List() {
		for _, name := range a.Primitives.Required {
			owned[name] = true
		}
		for _, name := range a.Primitives.Optional {
			owned[name] = true
		}
	}
	for _, name := range allPrimitives {
		if !owned[name] {
			t.Errorf("primitive %s is an orphan — no agent references it", name)
		}
	}
}

func TestIteration04CharterContracts(t *testing.T) {
	reg := agentregistry.New()
	if err := reg.Load(); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	checks := map[string][]string{
		"intro": {"3-5 个短校准问题", "intro/survey.json", "intro/assessment.json", "不得无证据", "每一项都必须写 `summary`"},
		"explain": {
			"研究目的",
			"explain/manifest.json",
			"更新相关已有页面",
			"kind: \"module\"",
			"重构整体结构",
			"可独立阅读的教程",
			"不得创建名为“第一性原理”",
			"不写“用户说”",
		},
		"practice": {"practice/answer-key.json", "true-false", "difficulty"},
	}
	for id, fragments := range checks {
		agent, ok := reg.Get(id)
		if !ok {
			t.Fatalf("agent %s missing", id)
		}
		for _, fragment := range fragments {
			if !strings.Contains(agent.CharterText, fragment) {
				t.Errorf("agent %s charter missing contract fragment %q", id, fragment)
			}
		}
	}
}
