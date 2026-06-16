// Package promptassembly builds the prompt package that gets sent to Claude.
package promptassembly

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/memorystore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Request carries the inputs to a prompt build.
type Request struct {
	ProjectSlug string
	ZoneName    workspace.ZoneName
	AgentID     string
	Intent      string
	// PracticeAttempt switches the Practice agent from question generation
	// to batch evaluation for one submitted attempt.
	PracticeAttempt       int
	PracticeQuestionCount int

	// SourceRefs are confusion IDs to inject into the prompt as source references.
	SourceRefs   []string
	ParentPageID string

	// FollowupPriorResultPaths is set when this invocation is a follow-up.
	// Each path points to a prior run's result.md.
	FollowupPriorResultPaths []string
}

// Package is the assembled prompt package.
type Package struct {
	PromptMd    string // the full prompt text passed to Claude
	RunDirName  string // e.g. "2026-06-08T15-22-11-explain"
	PackageMeta PackageMeta
}

// PackageMeta is written to run dir / package.json.
type PackageMeta struct {
	ProjectSlug           string                       `json:"projectSlug"`
	ProjectFile           string                       `json:"projectFile,omitempty"`
	ZoneName              workspace.ZoneName           `json:"zoneName"`
	AgentID               string                       `json:"agentId"`
	PredecessorFiles      []workspace.PredecessorFile  `json:"predecessorFiles"`
	OutputTargets         []agentregistry.OutputTarget `json:"outputTargets"`
	MemorySnapshot        *memorystore.Snapshot        `json:"memorySnapshot,omitempty"`
	FollowupPrior         []string                     `json:"followupPriorResultPaths,omitempty"`
	ParentPageID          string                       `json:"parentPageId,omitempty"`
	PracticeAttempt       int                          `json:"practiceAttempt,omitempty"`
	PracticeQuestionCount int                          `json:"practiceQuestionCount,omitempty"`
	GeneratedAt           time.Time                    `json:"generatedAt"`
}

// Build constructs the prompt package. Returns error on validation failure.
func Build(req Request, reg *agentregistry.Registry) (*Package, error) {
	if !workspace.ValidateSlug(req.ProjectSlug) {
		return nil, fmt.Errorf("invalid project slug: %q", req.ProjectSlug)
	}
	if !workspace.ValidateZoneName(string(req.ZoneName)) {
		return nil, fmt.Errorf("invalid zone: %s", req.ZoneName)
	}
	agent, ok := reg.Get(req.AgentID)
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", req.AgentID)
	}
	if !agentZoneAllowed(agent, req.ZoneName) {
		return nil, fmt.Errorf("agent %s not allowed in zone %s", agent.ID, req.ZoneName)
	}

	preds, err := workspace.ResolvePredecessorFiles(req.ProjectSlug, req.ZoneName)
	if err != nil {
		return nil, fmt.Errorf("resolve predecessors: %w", err)
	}
	memSnap, err := memorystore.Read(req.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("read memory: %w", err)
	}
	projectFile, projectBrief, err := readProjectBrief(req.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("read project brief: %w", err)
	}
	if req.PracticeAttempt > 0 && agent.ID != "practice" {
		return nil, fmt.Errorf("practice attempt evaluation requires practice agent")
	}
	if req.PracticeQuestionCount > 0 {
		if agent.ID != "practice" || req.PracticeAttempt > 0 {
			return nil, fmt.Errorf("practice question count requires practice generation mode")
		}
		if req.PracticeQuestionCount > 50 {
			return nil, fmt.Errorf("practice question count must be between 1 and 50")
		}
	}
	outputTargets := outputTargetsFor(agent, req)

	promptMd := renderPrompt(agent, req, preds, memSnap, projectFile, projectBrief)

	runDirName := timestampRunDir(req.AgentID, time.Now().UTC())

	return &Package{
		PromptMd:   promptMd,
		RunDirName: runDirName,
		PackageMeta: PackageMeta{
			ProjectSlug:           req.ProjectSlug,
			ProjectFile:           projectFile,
			ZoneName:              req.ZoneName,
			AgentID:               agent.ID,
			PredecessorFiles:      preds,
			OutputTargets:         outputTargets,
			MemorySnapshot:        memSnap,
			FollowupPrior:         req.FollowupPriorResultPaths,
			ParentPageID:          req.ParentPageID,
			PracticeAttempt:       req.PracticeAttempt,
			PracticeQuestionCount: req.PracticeQuestionCount,
			GeneratedAt:           time.Now().UTC(),
		},
	}, nil
}

// MarshalPackageMeta returns indented JSON for package.json.
func (p *Package) MarshalPackageMeta() ([]byte, error) {
	return json.MarshalIndent(p.PackageMeta, "", "  ")
}

func agentZoneAllowed(a *agentregistry.Agent, zone workspace.ZoneName) bool {
	for _, z := range a.AllowedZones {
		if z == zone {
			return true
		}
	}
	return false
}

func renderPrompt(
	agent *agentregistry.Agent,
	req Request,
	preds []workspace.PredecessorFile,
	mem *memorystore.Snapshot,
	projectFile string,
	projectBrief string,
) string {
	var b strings.Builder
	b.WriteString("# Agent Identity\n\n")
	b.WriteString(fmt.Sprintf("**%s %s** — %s\n\n", agent.Icon, agent.Name, agent.Description))

	// # User Story — explicitly anchors the agent to the learner's story
	// before any mechanism content is loaded. Required by the registry
	// schema; if empty, validateAgent would have failed at registry load.
	if strings.TrimSpace(agent.UserStory) != "" {
		b.WriteString("# User Story\n\n")
		b.WriteString(strings.TrimSpace(agent.UserStory))
		b.WriteString("\n\n")
	}

	b.WriteString("# Charter\n\n")
	b.WriteString(agent.CharterText)
	b.WriteString("\n\n")

	// # Reasoning Primitives — mechanism files loaded from
	// agents/primitives/<name>.md. Required primitives are mandatory;
	// optional primitives are included unconditionally in v1 (see
	// primitives_loader.go for the policy).
	if len(agent.Primitives.Required) > 0 || len(agent.Primitives.Optional) > 0 {
		b.WriteString("# Reasoning Primitives\n\n")
		expanded, err := ExpandPrimitives(agent.Primitives.Required, agent.Primitives.Optional)
		if err != nil {
			// This should not happen — registry validation should have
			// caught missing required primitives at server start. If we
			// reach here we still surface a clear note in the prompt
			// rather than silently shipping a broken prompt.
			fmt.Println("promptassembly: expand primitives warning:", err)
			b.WriteString(fmt.Sprintf("_(primitive expansion failed: %s)_\n\n", err))
		} else {
			b.WriteString(expanded)
		}
	}

	if agent.ID == "explain" {
		b.WriteString("# Explain Tutorial Artifact Contract\n\n")
		b.WriteString("- 目标产物是可独立阅读的教程，不是智能体回复、对话记录或学习诊断报告。\n")
		b.WriteString("- 正文直接陈述知识，不称呼读者，不使用“你”“我们”等对话人称。\n")
		b.WriteString("- 项目背景与 Intro 诊断只用于控制深度和选择样例；禁止写入“用户说”“用户自述”“判断依据”“校准缺口”“根据 assessment/project.md”等元信息。\n")
		b.WriteString("- 第一性原理不得成为独立页面、章节、标题或逐步推导；若使用，只能融入最后一页的“核心观点”，限 2-4 句话。\n")
		b.WriteString("- 研究问题框架由整套教程整体覆盖，不得让每一页机械重复同一组栏目。\n")
		b.WriteString("- 不写文件协议、追问机制、生成过程、交付摘要、后续邀请或智能体自述。\n\n")
	}
	if agent.ID == "practice" && req.PracticeAttempt > 0 {
		b.WriteString("# Practice Evaluation Artifact Contract\n\n")
		b.WriteString(fmt.Sprintf("- This invocation evaluates submitted attempt %d. Do not generate or overwrite `practice/tasks.json` or `practice/answer-key.json`.\n", req.PracticeAttempt))
		b.WriteString(fmt.Sprintf("- Read `practice/tasks.json`, `practice/answer-key.json`, `practice/attempts/%d.json`, and `practice/submissions/%d.json` before evaluating.\n", req.PracticeAttempt, req.PracticeAttempt))
		b.WriteString(fmt.Sprintf("- Write pure JSON to `practice/evaluations/%d.json` and a readable Markdown copy to `practice/evaluations/%d.md`.\n", req.PracticeAttempt, req.PracticeAttempt))
		b.WriteString("- Evaluate every submitted subjective task. Compare against the private scoring points, but do not expose hidden chain-of-thought or file-operation narration.\n")
		b.WriteString("- The JSON object must contain `attempt`, `summary`, `results`, `overallScore`, and `generatedAt`.\n")
		b.WriteString("- Every result must contain `taskId`, integer `score` from 0 to 5, concise `feedback`, a concrete `suggestedAnswer`, concise `evidence`, and boolean `passed`.\n")
		b.WriteString("- `summary` must synthesize strengths, recurring gaps, and the next study action. `suggestedAnswer` must directly answer that task rather than merely describe how to answer it.\n\n")
	}
	if agent.ID == "practice" && req.PracticeAttempt == 0 && req.PracticeQuestionCount > 0 {
		b.WriteString("# Practice Generation Contract\n\n")
		b.WriteString(fmt.Sprintf("- Generate exactly %d questions. Do not silently add, remove, or truncate questions.\n", req.PracticeQuestionCount))
		b.WriteString("- Write a matching answer-key entry for every generated question.\n\n")
	}

	b.WriteString("# Behavior Rules\n\n")
	if agent.ID == "explain" {
		b.WriteString("- Silently use predecessor files to choose depth and examples; never cite their filenames or narrate learner-profile evidence in tutorial prose.\n")
		b.WriteString("- Missing learner context is not tutorial content. Use a neutral beginner explanation without reporting what the learner did or did not provide.\n")
	} else {
		b.WriteString("- Cite predecessor files when building on prior zone output.\n")
		b.WriteString("- If information is missing, say so explicitly rather than fabricating.\n")
	}
	b.WriteString("- Do NOT edit `summary/summary.md`. That file is learner-owned.\n")
	b.WriteString("- Write Markdown that renders cleanly with GitHub-flavored Markdown + Mermaid.\n")
	b.WriteString("- Never draw diagrams with ASCII or Unicode text characters, including box-drawing flowcharts, trees, timelines, maps, or relationship diagrams. Use a fenced Mermaid block for every diagram.\n")
	b.WriteString("- This diagram rule does not prohibit ordinary source-code examples, mathematical notation, or Markdown tables.\n")
	b.WriteString("- Follow the charter's file and heading contracts exactly; JSON deliverables must remain pure JSON.\n\n")

	b.WriteString("# Project Context\n\n")
	b.WriteString(fmt.Sprintf("- Project slug: `%s`\n", req.ProjectSlug))
	b.WriteString(fmt.Sprintf("- Active zone: `%s`\n", req.ZoneName))
	b.WriteString(fmt.Sprintf("- Project brief file: `%s`\n", projectFile))

	b.WriteString("\n# Project Brief\n\n")
	if projectBrief == "" {
		b.WriteString("_(project.md is not present; do not invent missing project background)_\n")
	} else {
		b.WriteString("The following project-creation fields are already answered. Use them as context and do not ask the learner to repeat them.\n\n")
		b.WriteString(projectBrief)
		if !strings.HasSuffix(projectBrief, "\n") {
			b.WriteString("\n")
		}
	}
	if agent.ID == "intro" {
		b.WriteString("\n## Intro Calibration Boundary\n\n")
		b.WriteString("- Do not ask again why the learner chose the topic, their self-rated current level, target level, or completion standard when those fields are present above.\n")
		b.WriteString("- Ask only topic-specific diagnostic questions needed to locate prerequisite gaps, such as terminology, causal understanding, and a concrete application.\n")
		b.WriteString("- A broad current-level label is context, not proof of mastery. Diagnose specific knowledge without repeating the project-creation interview.\n")
	}

	b.WriteString("\n# Predecessor Files\n\n")
	if len(preds) == 0 {
		b.WriteString("_(none — this zone has no predecessors in the dependency graph)_\n")
	} else {
		for _, p := range preds {
			marker := " "
			if p.Exists {
				marker = "✓"
			} else {
				marker = "○"
			}
			b.WriteString(fmt.Sprintf("- %s `%s` — %s zone\n", marker, p.Path, p.ZoneName))
		}
	}

	b.WriteString("\n# Memory Snapshot\n\n")
	if mem != nil {
		if mem.ProjectMemoryExists {
			b.WriteString(fmt.Sprintf("- Project memory: `%s` (exists — read for context)\n", mem.ProjectMemoryPath))
		} else {
			b.WriteString(fmt.Sprintf("- Project memory: `%s` (not yet present)\n", mem.ProjectMemoryPath))
		}
		if mem.ProjectStateExists {
			b.WriteString(fmt.Sprintf("- Project state: `%s`\n", mem.ProjectStatePath))
		}
	}

	if len(req.FollowupPriorResultPaths) > 0 {
		b.WriteString("\n# Follow-up Context\n\n")
		b.WriteString("This invocation is a follow-up within an existing session. The user's prior turns produced the following result files — read them before responding:\n\n")
		for _, p := range req.FollowupPriorResultPaths {
			b.WriteString(fmt.Sprintf("- `%s`\n", p))
		}
	}

	if strings.TrimSpace(req.ParentPageID) != "" {
		b.WriteString("\n# Explain Page Context\n\n")
		b.WriteString(fmt.Sprintf("- parentPageId: `%s`\n", req.ParentPageID))
		b.WriteString("- Treat this invocation as a follow-up page. Preserve existing pages and append one manifest entry.\n")
	}

	b.WriteString("\n# Output Targets\n\n")
	for _, t := range outputTargetsFor(agent, req) {
		b.WriteString(fmt.Sprintf("- `%s/%s`\n", strings.ToLower(string(t.ZoneName)), t.Filename))
	}

	// Source References — confusion markers the learner selected for batch ask.
	if len(req.SourceRefs) > 0 {
		b.WriteString("\n# Source References\n\n")
		b.WriteString("The learner marked the following confusion points. Address each one:\n\n")
		for _, ref := range req.SourceRefs {
			b.WriteString(fmt.Sprintf("- confusion ID: `%s`\n", ref))
		}
	}

	if intent := strings.TrimSpace(req.Intent); intent != "" {
		b.WriteString("\n# Additional Guidance\n\n")
		b.WriteString(intent)
		b.WriteString("\n")
	}

	return b.String()
}

func outputTargetsFor(agent *agentregistry.Agent, req Request) []agentregistry.OutputTarget {
	if agent.ID == "practice" && req.PracticeAttempt > 0 {
		return []agentregistry.OutputTarget{
			{ZoneName: workspace.ZonePractice, Filename: fmt.Sprintf("evaluations/%d.json", req.PracticeAttempt)},
			{ZoneName: workspace.ZonePractice, Filename: fmt.Sprintf("evaluations/%d.md", req.PracticeAttempt)},
		}
	}
	return agent.DefaultOutputTargets
}

func readProjectBrief(projectSlug string) (string, string, error) {
	projectRoot, err := workspace.ProjectRootForSlug(projectSlug)
	if err != nil {
		return "", "", err
	}
	projectFile := filepath.Join(projectRoot, "project.md")
	data, err := os.ReadFile(projectFile)
	if err != nil {
		if os.IsNotExist(err) {
			return projectFile, "", nil
		}
		return "", "", err
	}
	return projectFile, strings.TrimSpace(string(data)), nil
}

func timestampRunDir(agentID string, at time.Time) string {
	// Filesystem-safe ISO timestamp (colons are forbidden in Windows folder names).
	stamp := at.Format("2006-01-02T15-04-05")
	return fmt.Sprintf("%s-%s", stamp, agentID)
}
