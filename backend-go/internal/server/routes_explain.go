package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/claudelauncher"
	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/promptassembly"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

var infographicJobs sync.Map

const (
	infographicStateFile  = "infographic-state.json"
	infographicPNG       = "infographic.png"
	infographicPromptFile = "image-prompt.txt"
)

// infographicState tracks the async infographic generation pipeline.
type infographicState struct {
	Status      string    `json:"status"` // pending, running, complete, failed, missing
	URL         string    `json:"url,omitempty"`
	Error       string    `json:"error,omitempty"`
	PromptRunDir string   `json:"promptRunDir,omitempty"`
	StartedAt   time.Time `json:"startedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// infographicStatePath returns the absolute path to the infographic state file.
func infographicStatePath(slug string) string {
	root, _ := workspace.ProjectRootForSlug(slug)
	return filepath.Join(root, "explain", infographicStateFile)
}

// infographicPNGPath returns the absolute path to the infographic PNG file.
func infographicPNGPath(slug string) string {
	root, _ := workspace.ProjectRootForSlug(slug)
	return filepath.Join(root, "explain", infographicPNG)
}

// readInfographicState reads the infographic state from disk.
func readInfographicState(slug string) (infographicState, bool, error) {
	path := infographicStatePath(slug)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return infographicState{Status: "missing"}, false, nil
		}
		return infographicState{Status: "missing"}, false, err
	}
	var state infographicState
	if err := json.Unmarshal(data, &state); err != nil {
		return infographicState{Status: "missing"}, false, err
	}
	return state, true, nil
}

// writeInfographicState writes the infographic state atomically.
func writeInfographicState(slug string, state infographicState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := infographicStatePath(slug)
	return workspace.AtomicWriteFile(path, data, 0o644)
}

// emitInfographicReady broadcasts an artifact-updated SSE event so the
// frontend can swap in the finished infographic without waiting for the
// next poll. Call only after the complete state is durably written.
func emitInfographicReady(slug string) {
	broadcaster.Emit("artifact-updated", map[string]any{
		"slug":     slug,
		"artifact": "explain/infographic.png",
	})
}

// handleGetExplainInfographic returns the current status of the infographic generation.
func (s *Server) handleGetExplainInfographic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}

	state, exists, err := readInfographicState(slug)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Missing state file
	if !exists {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "missing"})
		return
	}

	// If complete, verify PNG still exists
	if state.Status == "complete" {
		pngPath := infographicPNGPath(slug)
		if _, err := os.Stat(pngPath); err != nil {
			// PNG missing - treat as missing so user can retry
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "missing"})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status":    state.Status,
			"url":       state.URL,
			"updatedAt": state.UpdatedAt,
		})
		return
	}

	// For pending/running, check staleness (10 minutes)
	if state.Status == "pending" || state.Status == "running" {
		if time.Since(state.StartedAt) > 10*time.Minute {
			// Stale - treat as failed
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"status": "failed",
				"error":  "stale",
			})
			return
		}
	}

	// Return current state
	resp := map[string]any{"status": state.Status}
	if state.Error != "" {
		resp["error"] = state.Error
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// handleRequestExplainInfographic starts the infographic generation pipeline.
func (s *Server) handleRequestExplainInfographic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if !workspace.ValidateSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid slug")
		return
	}

	// force=1 = explicit user-requested regeneration: bypass the "already
	// complete" early-return so the pipeline overwrites the existing PNG.
	force := r.URL.Query().Get("force") == "1"

	// Check if already complete (skip when force-regenerating)
	if !force {
		state, exists, _ := readInfographicState(slug)
		if exists && state.Status == "complete" {
			pngPath := infographicPNGPath(slug)
			if _, err := os.Stat(pngPath); err == nil {
				httpx.WriteJSON(w, http.StatusOK, map[string]any{
					"status":    state.Status,
					"url":       state.URL,
					"updatedAt": state.UpdatedAt,
				})
				return
			}
		}
	}

	// Check availability
	if !s.ClaudeAvailable {
		httpx.Error(w, http.StatusServiceUnavailable, "claude binary not available")
		return
	}
	if s.ImageConfig == nil || s.ImageConfig.ImageAPIKey == "" || !s.ImageAvailable {
		httpx.Error(w, http.StatusServiceUnavailable, "infographic generation not configured")
		return
	}

	// Dedupe in-flight requests
	jobKey := slug
	if _, loaded := infographicJobs.LoadOrStore(jobKey, true); loaded {
		httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": "running"})
		return
	}

	// Write pending state
	now := time.Now().UTC()
	pendingState := infographicState{
		Status:    "pending",
		StartedAt: now,
		UpdatedAt: now,
	}
	if err := writeInfographicState(slug, pendingState); err != nil {
		infographicJobs.Delete(jobKey)
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Start pipeline in background
	go func() {
		defer infographicJobs.Delete(jobKey)
		s.runInfographicPipeline(context.Background(), slug)
	}()

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"status": "queued"})
}

// runInfographicPipeline executes the two-stage infographic generation.
func (s *Server) runInfographicPipeline(ctx context.Context, slug string) {
	cfg := s.ImageConfig
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		writeFailedState(slug, err.Error())
		return
	}

	// Write running state
	now := time.Now().UTC()
	runningState := infographicState{
		Status:    "running",
		StartedAt: now,
		UpdatedAt: now,
	}
	if err := writeInfographicState(slug, runningState); err != nil {
		writeFailedState(slug, err.Error())
		return
	}

	// STAGE A: Craft the image prompt
	runDirName := promptassembly.MakeRunDirName("infographic-crafter", time.Now().UTC())
	runDirRel := filepath.Join("runs", runDirName)
	prompt := buildCrafterPrompt(slug, runDirRel)
	pkg := &promptassembly.Package{
		PromptMd:   prompt,
		RunDirName: runDirName,
	}

	// Record prompt run dir in state
	runningState.PromptRunDir = runDirRel
	runningState.UpdatedAt = time.Now().UTC()
	writeInfographicState(slug, runningState)

	// Launch headless Claude to craft prompt
	err = claudelauncher.LaunchHeadless(ctx, slug, "infographic-crafter", s.ClaudeBin, cfg.ImagePromptModel, pkg)
	if err != nil {
		writeFailedState(slug, "stage A failed: "+err.Error())
		return
	}

	// Read the crafted prompt
	promptPath := filepath.Join(root, runDirRel, infographicPromptFile)
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		writeFailedState(slug, "crafter produced no prompt")
		return
	}

	crafted := strings.TrimSpace(string(promptBytes))
	// Strip markdown code fence if present
	if strings.HasPrefix(crafted, "```") {
		lines := strings.Split(crafted, "\n")
		if len(lines) >= 2 {
			// Find the end of the fence
			endFence := -1
			for i := 1; i < len(lines); i++ {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
					endFence = i
					break
				}
			}
			if endFence > 0 {
				crafted = strings.TrimSpace(strings.Join(lines[1:endFence], "\n"))
			} else {
				// No closing fence, strip first line only
				crafted = strings.TrimSpace(strings.Join(lines[1:], "\n"))
			}
		}
	}

	// Truncate to 4000 runes if needed
	runes := []rune(crafted)
	if len(runes) > 4000 {
		crafted = string(runes[:4000])
	}

	finalPrompt := "高质量、清晰明了的手绘信息图，" + crafted

	// STAGE B: Generate the image
	scriptAbs := filepath.Join(paths.WORKSPACE, "scripts", "gen_infographic.py")
	if paths.WORKSPACE == "" {
		scriptAbs = filepath.Join(paths.PROJECT_ROOT, "scripts", "gen_infographic.py")
	}
	pngAbs := infographicPNGPath(slug)

	ctx2, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx2, cfg.PythonBin, scriptAbs, finalPrompt, pngAbs)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"IMAGE_API_KEY="+cfg.ImageAPIKey,
		"IMAGE_BASE_URL="+cfg.ImageBaseURL,
		"IMAGE_MODEL="+cfg.ImageModel,
	)

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderr := stderrBuf.String()
		if len(stderr) > 500 {
			stderr = stderr[:500] + "..."
		}
		writeFailedState(slug, "stage B failed: "+err.Error()+": "+stderr)
		return
	}

	// Verify PNG exists
	if _, err := os.Stat(pngAbs); err != nil {
		writeFailedState(slug, "script exited but png missing")
		return
	}

	// Write complete state
	completeState := infographicState{
		Status:      "complete",
		URL:         "/files/projects/" + slug + "/explain/" + infographicPNG,
		PromptRunDir: runDirRel,
		StartedAt:   runningState.StartedAt,
		UpdatedAt:   time.Now().UTC(),
	}
	if err := writeInfographicState(slug, completeState); err != nil {
		fmt.Println("infographic: failed to write complete state:", err)
		return
	}
	emitInfographicReady(slug)
}

// buildCrafterPrompt constructs the prompt for the infographic-crafter subagent.
func buildCrafterPrompt(slug, runDirRel string) string {
	return fmt.Sprintf(`你是一个图像提示词提炼助手。当前工作目录是学习项目根目录。

任务：
1. 读取 explain/manifest.json，得到主题标题与全部页面列表（按 order 排序）。
2. 以"最后一页"（order 最大的页面，通常是批判性思维总结页）为主要信息源；如需补充上下文，再读 1-2 页正文。explain/output.md 只是兼容占位，真正内容在 pages/ 里。
3. 把主题的"批判性思维精华"蒸馏成一份简洁的信息图内容说明，供图像模型生成一张清晰简洁的手绘信息图：
   - 只挑选最承重的概念、关系与结论；详略得当，绝不把所有细节都塞进一张图。
   - 写清"这张图要让读者一眼抓住什么主线、关键节点之间是什么关系"（因果、层级、对比、流程先后、反馈闭环、整体—部分等）。
   - 用简洁的自然语言陈述要点本身，不堆装饰性细节。
   - 不要规定画面布局或构图——不要写"上图/下图""左侧/右侧""用箭头从X指向Y""分成几栏"等空间或排版指令；画面由图像模型自行设计。
   - 不要写风格或媒介描述（如"手绘""马克笔""白板"）；整体风格由系统统一指定。

输出契约（严格遵守）：
- 只把"最终提示词文本"写入文件：%s/image-prompt.txt
- 文件内容只能是提示词本身：不要前言、解释、markdown 代码围栏或多余空行。
- 提示词不超过 400 字。
- 不要修改其它任何文件。
`, runDirRel)
}

// writeFailedState writes a failed state atomically.
func writeFailedState(slug, errorMsg string) {
	state := infographicState{
		Status:    "failed",
		Error:     errorMsg,
		UpdatedAt: time.Now().UTC(),
	}
	// Preserve startedAt if we had it
	if existing, exists, _ := readInfographicState(slug); exists {
		state.StartedAt = existing.StartedAt
		state.PromptRunDir = existing.PromptRunDir
	}
	if err := writeInfographicState(slug, state); err != nil {
		fmt.Println("infographic: failed to write failed state:", err)
	}
}
