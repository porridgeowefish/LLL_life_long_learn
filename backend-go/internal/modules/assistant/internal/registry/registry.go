// Package agentregistry loads agent identity + charter definitions from disk.
package agentregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

// OutputTarget declares where an agent's output should be promoted.
type OutputTarget struct {
	ZoneName workspace.ZoneName `json:"zone"`
	Filename string             `json:"filename"`
}

// Primitives declares which reasoning primitive .md files the agent requires
// or optionally uses. Required primitives are always expanded into the prompt
// by promptassembly; optional primitives are activated by charter rules.
// See docs/00-product-and-architecture/AGENT_PRIMITIVES.md.
type Primitives struct {
	Required []string `json:"required,omitempty"`
	Optional []string `json:"optional,omitempty"`
}

// Agent is the runtime shape of one registry entry.
type Agent struct {
	ID                   string                  `json:"id"`
	Name                 string                  `json:"name"`
	Icon                 string                  `json:"icon"`
	Description          string                  `json:"description,omitempty"`
	UserStory            string                  `json:"userStory,omitempty"`
	Primitives           Primitives              `json:"primitives,omitempty"`
	AllowedZones         []workspace.ZoneName    `json:"allowedZones"`
	AllowedProjectTypes  []workspace.ProjectType `json:"allowedProjectTypes,omitempty"`
	CharterPath          string                  `json:"charterPath"`
	DefaultOutputTargets []OutputTarget          `json:"defaultOutputTargets"`

	// CharterText is loaded lazily and cached in-memory.
	CharterText string `json:"-"`
}

// MaxRequiredPrimitives caps the size of an agent's required-primitive list.
// The limit exists to prevent recreating a single "god skill" that absorbs
// every mechanism — see AGENT_PRIMITIVES.md anti-patterns.
const MaxRequiredPrimitives = 6

// validPrimitiveName matches the file-name shape agents/primitives/<name>.md.
var validPrimitiveName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Registry is the in-memory collection of agents, keyed by ID.
type Registry struct {
	mu   sync.RWMutex
	byID map[string]*Agent
}

// New returns an empty registry. Call Load() to populate from disk.
func New() *Registry {
	return &Registry{byID: make(map[string]*Agent)}
}

// Load scans AGENTS_ROOT/registry/*.json and loads every agent definition.
// Returns an error if any agent is malformed or references an unknown zone.
func (r *Registry) Load() error {
	regDir := filepath.Join(activeAgentsRoot(), "registry")
	entries, err := os.ReadDir(regDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil // no agents yet — fine
	}
	if err != nil {
		return fmt.Errorf("agent registry: read dir: %w", err)
	}
	loaded := make(map[string]*Agent)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(regDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var a Agent
		if err := json.Unmarshal(data, &a); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		if a.ID == "" {
			return fmt.Errorf("agent in %s missing id", path)
		}
		if err := validateAgent(&a); err != nil {
			return fmt.Errorf("agent %s: %w", a.ID, err)
		}
		// Resolve charter path. Registry entries commonly use a path
		// relative to the workspace root (e.g. "agents/charters/explain.md"),
		// so we anchor relative paths to the workspace root — the parent
		// of AGENTS_ROOT — to make loading robust against the process
		// working directory.
		if a.CharterPath != "" {
			chPath := a.CharterPath
			if !filepath.IsAbs(chPath) {
				chPath = filepath.Join(filepath.Dir(activeAgentsRoot()), chPath)
			}
			charterBytes, err := os.ReadFile(chPath)
			if err != nil {
				return fmt.Errorf("agent %s: read charter %s: %w", a.ID, a.CharterPath, err)
			}
			a.CharterText = string(charterBytes)
		}
		loaded[a.ID] = &a
	}
	r.mu.Lock()
	r.byID = loaded
	r.mu.Unlock()
	return nil
}

func validateAgent(a *Agent) error {
	if len(a.AllowedZones) == 0 && len(a.AllowedProjectTypes) == 0 {
		return errors.New("agent must allow at least one zone or project type")
	}
	for _, z := range a.AllowedZones {
		if !workspace.ValidateZoneName(string(z)) {
			return fmt.Errorf("unknown zone in allowedZones: %s", z)
		}
	}
	for _, projectType := range a.AllowedProjectTypes {
		if projectType == "" || !workspace.ValidateProjectType(projectType) {
			return fmt.Errorf("unknown project type in allowedProjectTypes: %s", projectType)
		}
	}
	if strings.TrimSpace(a.UserStory) == "" {
		return errors.New("userStory is empty (every agent must declare the user story it serves)")
	}
	if len(a.Primitives.Required) > MaxRequiredPrimitives {
		return fmt.Errorf("too many required primitives: %d > %d", len(a.Primitives.Required), MaxRequiredPrimitives)
	}
	seen := map[string]bool{}
	for _, name := range a.Primitives.Required {
		if !validPrimitiveName.MatchString(name) {
			return fmt.Errorf("invalid required primitive name: %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate required primitive: %s", name)
		}
		seen[name] = true
	}
	for _, name := range a.Primitives.Optional {
		if !validPrimitiveName.MatchString(name) {
			return fmt.Errorf("invalid optional primitive name: %q", name)
		}
		if seen[name] {
			return fmt.Errorf("primitive %s appears in both required and optional", name)
		}
		seen[name] = true
	}
	return nil
}

// PrimitivePath returns the on-disk path of a primitive .md file given its id.
// Pure function — does not check existence. Callers (registry loader,
// promptassembly) use this to read or validate primitives.
func PrimitivePath(name string) string {
	return filepath.Join(activeAgentsRoot(), "primitives", name+".md")
}

// PrimitiveExists reports whether the named primitive file is on disk.
func PrimitiveExists(name string) bool {
	if !validPrimitiveName.MatchString(name) {
		return false
	}
	_, err := os.Stat(PrimitivePath(name))
	return err == nil
}

// List returns every loaded agent, sorted by ID.
func (r *Registry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Agent, 0, len(r.byID))
	for _, a := range r.byID {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns the agent with the given ID, or nil/false if not found.
func (r *Registry) Get(id string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.byID[id]
	return a, ok
}

// agentsRootOverride is set by tests to redirect AGENTS_ROOT to a temp dir.
// Production code never sets it.
var agentsRootOverride string

// AgentsRootForTest returns the current override (test helper).
func AgentsRootForTest() string { return agentsRootOverride }

// SetAgentsRootForTest sets the override (test helper).
func SetAgentsRootForTest(dir string) { agentsRootOverride = dir }

// activeAgentsRoot returns the override or the real AGENTS_ROOT.
func activeAgentsRoot() string {
	if agentsRootOverride != "" {
		return agentsRootOverride
	}
	return paths.AGENTS_ROOT
}
