// Package promptassembly primitive loader.
//
// Reads reasoning-primitive .md files from agents/primitives/ and expands
// them into a single Markdown block ready for inclusion in the agent prompt.
// Caches file contents in-process; the cache is process-wide and assumes
// primitive files do not change at runtime. Tests reset the cache via
// ClearPrimitiveCacheForTest.
package promptassembly

import (
	"fmt"
	"os"
	"strings"
	"sync"

	agentregistry "github.com/xmz14/lll/backend-go/internal/modules/assistant/internal/registry"
)

// primitiveCache caches primitive bodies in-memory keyed by primitive id.
// Empty string is a sentinel meaning "known missing" — it prevents a
// repeated os.Stat on a missing file from causing repeated disk hits.
var primitiveCache = struct {
	sync.RWMutex
	m map[string]string
}{
	m: map[string]string{},
}

// LoadPrimitive reads agents/primitives/<name>.md and returns its body.
// On success the body is cached; on failure the miss is cached as an empty
// sentinel so the same primitive won't be re-read repeatedly.
func LoadPrimitive(name string) (string, error) {
	primitiveCache.RLock()
	if v, ok := primitiveCache.m[name]; ok {
		primitiveCache.RUnlock()
		if v == "" {
			return "", fmt.Errorf("primitive %q not found (cached miss)", name)
		}
		return v, nil
	}
	primitiveCache.RUnlock()

	path := agentregistry.PrimitivePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		// Cache the miss so we don't keep hammering the filesystem.
		primitiveCache.Lock()
		primitiveCache.m[name] = ""
		primitiveCache.Unlock()
		return "", fmt.Errorf("load primitive %q: %w", name, err)
	}
	body := string(data)
	primitiveCache.Lock()
	primitiveCache.m[name] = body
	primitiveCache.Unlock()
	return body, nil
}

// ClearPrimitiveCacheForTest resets the in-memory primitive cache.
// Production code never calls this; primitive files are immutable at runtime.
func ClearPrimitiveCacheForTest() {
	primitiveCache.Lock()
	defer primitiveCache.Unlock()
	primitiveCache.m = map[string]string{}
}

// ExpandPrimitives returns the body of the # Reasoning Primitives prompt
// section, given an agent's required and optional primitive lists.
//
// Behavior:
//   - Every name in `required` is loaded and concatenated, in order. A
//     missing required primitive returns an error — charter validation
//     upstream should normally have caught this, but the loader treats it
//     as fatal so a half-broken primitive set never silently ships.
//   - Every name in `optional` is loaded and concatenated, in order. A
//     missing optional primitive produces a warning on stderr and is
//     skipped — optional primitives may legitimately not exist on disk
//     (e.g. an agent's optional list mentions a future primitive).
//
// Optional primitive bodies are included as reference material, not as output
// requirements. The charter decides whether a mechanism is activated for the
// current topic. The prompt preface below makes that distinction explicit so
// merely listing an optional primitive does not create a matching section.
func ExpandPrimitives(required, optional []string) (string, error) {
	var b strings.Builder
	if len(required) > 0 {
		b.WriteString("### Required primitives\n\n")
		for _, name := range required {
			body, err := LoadPrimitive(name)
			if err != nil {
				return "", fmt.Errorf("required primitive %s: %w", name, err)
			}
			b.WriteString(body)
			if !strings.HasSuffix(body, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}
	if len(optional) > 0 {
		b.WriteString("### Optional primitive references\n\n")
		b.WriteString("The following mechanisms are reference material only. Do not create a page or section merely because a mechanism is listed here. Activate one only when the charter's conditions fit the current topic.\n\n")
		for _, name := range optional {
			body, err := LoadPrimitive(name)
			if err != nil {
				// Optional missing: warn and skip.
				fmt.Fprintln(os.Stderr, "promptassembly: optional primitive", name, "missing:", err)
				continue
			}
			b.WriteString(body)
			if !strings.HasSuffix(body, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}
