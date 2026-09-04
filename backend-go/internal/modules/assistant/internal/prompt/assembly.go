// Package promptassembly builds the prompt package that gets sent to Claude.
package promptassembly

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	agentregistry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
	preferencestore "github.com/xmz14/lll/backend-go/internal/modules/preferences"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
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

// ProjectAgentRequest carries inputs for a project-level agent that is bound to
// a project type rather than one of the five learning zones.
type ProjectAgentRequest struct {
	ProjectSlug string
	AgentID     string
	Intent      string
	OutputPaths []string
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
	LearningScopeFile     string                       `json:"learningScopeFile,omitempty"`
	ZoneName              workspace.ZoneName           `json:"zoneName,omitempty"`
	ProjectType           workspace.ProjectType        `json:"projectType,omitempty"`
	ProjectOutputPaths    []string                     `json:"projectOutputPaths,omitempty"`
	AgentID               string                       `json:"agentId"`
	PredecessorFiles      []workspace.PredecessorFile  `json:"predecessorFiles"`
	OutputTargets         []agentregistry.OutputTarget `json:"outputTargets"`
	PreferenceSnapshot    *preferencestore.Snapshot    `json:"preferenceSnapshot,omitempty"`
	FollowupPrior         []string                     `json:"followupPriorResultPaths,omitempty"`
	ParentPageID          string                       `json:"parentPageId,omitempty"`
	PracticeAttempt       int                          `json:"practiceAttempt,omitempty"`
	PracticeQuestionCount int                          `json:"practiceQuestionCount,omitempty"`
	GeneratedAt           time.Time                    `json:"generatedAt"`
}

// BuildProjectAgent constructs a prompt package for an agent bound directly to
// a project type. It deliberately does not synthesize a sixth learning zone.
func BuildProjectAgent(req ProjectAgentRequest, reg *agentregistry.Registry) (*Package, error) {
	if !workspace.ValidateSlug(req.ProjectSlug) {
		return nil, fmt.Errorf("invalid project slug: %q", req.ProjectSlug)
	}
	agent, ok := reg.Get(req.AgentID)
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", req.AgentID)
	}
	state, err := workspace.ReadProjectState(req.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("read project state: %w", err)
	}
	if !agentProjectTypeAllowed(agent, state.ProjectType) {
		return nil, fmt.Errorf("agent %s not allowed for project type %s", agent.ID, state.ProjectType)
	}
	projectFile, projectBrief, err := readProjectBrief(req.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("read project brief: %w", err)
	}
	outputPaths := sanitizeProjectOutputPaths(req.OutputPaths)
	if len(outputPaths) == 0 {
		return nil, errors.New("project-level agent requires an output path")
	}

	preferenceSnapshot, err := preferencestore.Read()
	if err != nil {
		return nil, fmt.Errorf("read preferences: %w", err)
	}
	generatedAt := time.Now().UTC()
	return &Package{
		PromptMd:   renderProjectAgentPrompt(agent, req, state.ProjectType, projectFile, projectBrief, outputPaths, &preferenceSnapshot),
		RunDirName: timestampRunDir(req.AgentID, generatedAt),
		PackageMeta: PackageMeta{
			ProjectSlug:        req.ProjectSlug,
			ProjectFile:        projectFile,
			ProjectType:        state.ProjectType,
			AgentID:            agent.ID,
			ProjectOutputPaths: outputPaths,
			PreferenceSnapshot: &preferenceSnapshot,
			GeneratedAt:        generatedAt,
		},
	}, nil
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
	preferenceSnapshot, err := preferencestore.Read()
	if err != nil {
		return nil, fmt.Errorf("read preferences: %w", err)
	}
	projectFile, projectBrief, err := readProjectBrief(req.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("read project brief: %w", err)
	}
	learningScopeFile, learningScope, err := readLearningScope(req.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("read learning scope: %w", err)
	}
	introSurvey, err := readIntroSurvey(req.ProjectSlug, agent.ID)
	if err != nil {
		return nil, fmt.Errorf("read intro survey: %w", err)
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

	promptMd := renderPrompt(agent, req, preds, &preferenceSnapshot, projectFile, projectBrief, learningScopeFile, learningScope, introSurvey)

	runDirName := timestampRunDir(req.AgentID, time.Now().UTC())

	return &Package{
		PromptMd:   promptMd,
		RunDirName: runDirName,
		PackageMeta: PackageMeta{
			ProjectSlug:           req.ProjectSlug,
			ProjectFile:           projectFile,
			LearningScopeFile:     learningScopeFile,
			ZoneName:              req.ZoneName,
			AgentID:               agent.ID,
			PredecessorFiles:      preds,
			OutputTargets:         outputTargets,
			PreferenceSnapshot:    &preferenceSnapshot,
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

func agentProjectTypeAllowed(a *agentregistry.Agent, projectType workspace.ProjectType) bool {
	for _, allowed := range a.AllowedProjectTypes {
		if allowed == projectType {
			return true
		}
	}
	return false
}

func sanitizeProjectOutputPaths(paths []string) []string {
	result := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, value := range paths {
		value = filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
		if value == "." || value == "" || strings.HasPrefix(value, "../") || filepath.IsAbs(value) || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func renderProjectAgentPrompt(
	agent *agentregistry.Agent,
	req ProjectAgentRequest,
	projectType workspace.ProjectType,
	projectFile string,
	projectBrief string,
	outputPaths []string,
	preferences *preferencestore.Snapshot,
) string {
	var b strings.Builder
	b.WriteString("# Agent Identity\n\n")
	b.WriteString(fmt.Sprintf("**%s %s** — %s\n\n", agent.Icon, agent.Name, agent.Description))
	b.WriteString("# User Story\n\n")
	b.WriteString(strings.TrimSpace(agent.UserStory))
	b.WriteString("\n\n# Charter\n\n")
	b.WriteString(strings.TrimSpace(agent.CharterText))
	b.WriteString("\n\n# Project-Level Invocation Contract\n\n")
	b.WriteString("- This is an explicit native Agent CLI invocation recorded as a normal session and run.\n")
	b.WriteString("- It is bound to the project type, not to Intro / Explain / Practice / Extend / Summary.\n")
	b.WriteString("- Read the project brief and the current output artifact before revising it.\n")
	b.WriteString("- Write the final learner-facing Markdown directly to the declared project-root output path.\n")
	b.WriteString("- Do not merely paste the proposed artifact into the terminal response.\n\n")
	b.WriteString("# Project Context\n\n")
	b.WriteString(fmt.Sprintf("- Project slug: `%s`\n", req.ProjectSlug))
	b.WriteString(fmt.Sprintf("- Project type: `%s`\n", projectType))
	b.WriteString(fmt.Sprintf("- Project brief file: `%s`\n", projectFile))
	b.WriteString("\n# Project Brief\n\n")
	if strings.TrimSpace(projectBrief) == "" {
		b.WriteString("_(project.md is not present; do not invent missing learner background)_\n")
	} else {
		b.WriteString(projectBrief)
		b.WriteString("\n")
	}
	appendPreferences(&b, preferences)
	b.WriteString("\n# Project-Root Output Paths\n\n")
	for _, outputPath := range outputPaths {
		b.WriteString(fmt.Sprintf("- `%s`\n", outputPath))
	}
	if intent := strings.TrimSpace(req.Intent); intent != "" {
		b.WriteString("\n# Additional Guidance\n\n")
		b.WriteString(intent)
		b.WriteString("\n")
	}
	return b.String()
}

func renderPrompt(
	agent *agentregistry.Agent,
	req Request,
	preds []workspace.PredecessorFile,
	preferences *preferencestore.Snapshot,
	projectFile string,
	projectBrief string,
	learningScopeFile string,
	learningScope string,
	introSurvey string,
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
		b.WriteString("- 第一性原理不得成为独立页面、章节、标题或逐步推导；若使用，只能融入最后一页“批判性思维总结”的“核心观点与视角”维度，限 2-4 句话。\n")
		b.WriteString("- 研究问题框架由整套教程整体覆盖，不得让每一页机械重复同一组栏目。\n")
		b.WriteString("- 不写文件协议、追问机制、生成过程、交付摘要、后续邀请或智能体自述。\n\n")
	}
	if agent.ID == "intro" {
		b.WriteString("# Intro Iteration Contract\n\n")
		b.WriteString("- 如果用户在终端里质疑、补充或修正 Intro 判断，把已有 Intro 产物视为可改进草稿。\n")
		b.WriteString("- 先直接回应用户疑问，再按证据更新 `intro/output.md` 与 `intro/assessment.json`。\n")
		b.WriteString("- 只改 Intro 产物；standalone draft 范围可在校准完成后写入 `learning-scope.json`，不得改 Explain、Practice、Extend 或 Summary 产物。\n")
		b.WriteString("- 本轮不新增前端追问入口，也不实现快照；必要更新直接原地写入 Intro 文件。\n\n")
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
		b.WriteString("- Generate topic-specific diagnostic questions as `intro/survey.json`; do not ask the learner to answer calibration questions in the CLI/TUI.\n")
		b.WriteString("- Ask only topic-specific diagnostic questions needed to locate prerequisite gaps, such as terminology, causal understanding, and a concrete application.\n")
		b.WriteString("- A broad current-level label is context, not proof of mastery. Diagnose specific knowledge without repeating the project-creation interview.\n")
		b.WriteString("- If `intro/survey.json` below contains learner answers, use those answers as evidence and write `intro/output.md` plus `intro/assessment.json`.\n")
	}

	b.WriteString("\n# Learning Scope\n\n")
	b.WriteString(fmt.Sprintf("- Scope file: `%s`\n", learningScopeFile))
	if strings.TrimSpace(learningScope) == "" {
		b.WriteString("- `learning-scope.json` is absent because this is a legacy project. Use the project title as a conservative draft boundary; do not broaden it speculatively.\n")
	} else {
		b.WriteString("The following JSON is the authoritative content boundary for this project:\n\n")
		b.WriteString("```json\n")
		b.WriteString(strings.TrimSpace(learningScope))
		b.WriteString("\n```\n")
	}
	b.WriteString("\n## Scope Enforcement\n\n")
	b.WriteString("- `inScope` and `ownedConcepts` are the concepts this project may teach in full depth.\n")
	b.WriteString("- `prerequisites` and `reusedConcepts` may appear only as the minimum support needed for this topic; do not turn them into parallel core modules.\n")
	b.WriteString("- `outOfScope` must not become a core page, exercise objective, extension branch, or summary claim for this project.\n")
	b.WriteString("- If a useful adjacent concept is outside the boundary, name the boundary briefly instead of teaching that sibling topic here.\n")
	if agent.ID == "intro" {
		b.WriteString("- When `source.type` is `discipline-map` and status is `ready`, preserve this objective boundary. Intro only calibrates prerequisite readiness, depth, examples, scaffolding, and practice difficulty.\n")
		b.WriteString("- When `source.type` is `standalone` and status is `draft`, use the completed survey to finalize `learning-scope.json`: keep the learner's chosen title, set status to `ready`, and fill goal/inScope/outOfScope/prerequisites/ownedConcepts/reusedConcepts conservatively.\n")
		b.WriteString("- Never expand a map-origin scope from learner calibration. A broader scope requires an explicit project-level decision outside Intro.\n")
	}

	if agent.ID == "intro" {
		b.WriteString("\n# Intro Survey State\n\n")
		if strings.TrimSpace(introSurvey) == "" {
			b.WriteString("`intro/survey.json` does not exist yet. First produce that file only, so the frontend can render the calibration questions as a page.\n")
		} else {
			b.WriteString("Current `intro/survey.json` content follows. If answers are present, treat them as the learner's calibration evidence.\n\n")
			b.WriteString("```json\n")
			b.WriteString(strings.TrimSpace(introSurvey))
			b.WriteString("\n```\n")
		}
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

	appendPreferences(&b, preferences)

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
		b.WriteString("- Treat `parentPageId` as optional context for the terminal follow-up, not as an instruction to append a page.\n")
		b.WriteString("- Default to updating the relevant existing Explain page. Create a new `kind: \"module\"` page only for an independent knowledge module.\n")
		b.WriteString("- If the follow-up exposes a better logical structure, revise page titles, order, splits, merges, sections, and manifest in place.\n")
		b.WriteString("- Preserve useful learner hypotheses or objections inside the artifact, then explain their limits or corrections.\n")
		b.WriteString("- Do not create revision snapshots in this iteration.\n")
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

func appendPreferences(b *strings.Builder, preferences *preferencestore.Snapshot) {
	b.WriteString("\n# Global Learner Preferences (read-only)\n\n")
	if preferences == nil || !preferences.Exists || strings.TrimSpace(preferences.Content) == "" {
		b.WriteString("_(no global preferences supplied)_\n")
		return
	}
	b.WriteString("Use this user-authored context when relevant. Never modify the canonical preferences file or infer new preferences.\n\n")
	b.WriteString(preferences.Content)
	if !strings.HasSuffix(preferences.Content, "\n") {
		b.WriteString("\n")
	}
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

func readIntroSurvey(projectSlug string, agentID string) (string, error) {
	if agentID != "intro" {
		return "", nil
	}
	projectRoot, err := workspace.ProjectRootForSlug(projectSlug)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "intro", "survey.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readLearningScope(projectSlug string) (string, string, error) {
	projectRoot, err := workspace.ProjectRootForSlug(projectSlug)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(projectRoot, "learning-scope.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, "", nil
		}
		return "", "", err
	}
	return path, strings.TrimSpace(string(data)), nil
}

// MakeRunDirName constructs a timestamped run directory name for a given agentID.
// The format is "YYYY-MM-DDTHH-MM-SS-agentID" (filesystem-safe ISO timestamp).
func MakeRunDirName(agentID string, at time.Time) string {
	return timestampRunDir(agentID, at)
}

func timestampRunDir(agentID string, at time.Time) string {
	// Filesystem-safe ISO timestamp (colons are forbidden in Windows folder names).
	stamp := at.Format("2006-01-02T15-04-05")
	return fmt.Sprintf("%s-%s", stamp, agentID)
}
