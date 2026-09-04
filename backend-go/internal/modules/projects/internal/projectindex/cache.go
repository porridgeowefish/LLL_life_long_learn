// Package projectindex maintains an in-memory cache of project metadata
// rebuilt from the filesystem on demand. The cache is a runtime accelerator
// only; workspace state.json files are the durable truth.
package projectindex

import (
	"sync"
	"time"

	"github.com/xmz14/lll/backend-go/internal/modules/projects/internal/workspace"
)

// Cache holds flat project metadata keyed by globally unique slug.
type Cache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
}

type cacheEntry struct {
	Meta   workspace.ProjectMeta
	Loaded time.Time
}

// New returns an empty cache. Call Rebuild() to populate.
func New() *Cache {
	return &Cache{items: make(map[string]cacheEntry)}
}

// Rebuild walks the workspace and refreshes every entry.
func (c *Cache) Rebuild() error {
	all, err := workspace.IndexAll()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheEntry, len(all))
	now := time.Now()
	for _, m := range all {
		c.items[m.Slug] = cacheEntry{Meta: m, Loaded: now}
	}
	return nil
}

// All returns a flat list of cached metadata (sorted by slug).
func (c *Cache) All() []workspace.ProjectMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]workspace.ProjectMeta, 0, len(c.items))
	for _, e := range c.items {
		out = append(out, e.Meta)
	}
	return out
}

// Get returns metadata for a slug. Returns ok=false if not cached.
func (c *Cache) Get(slug string) (workspace.ProjectMeta, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.items[slug]; ok {
		return e.Meta, true
	}
	return workspace.ProjectMeta{}, false
}

// Invalidate drops one entry. Called by writers that modify a project.
func (c *Cache) Invalidate(slug string) {
	c.mu.Lock()
	delete(c.items, slug)
	c.mu.Unlock()
}

// InvalidateAll clears the cache entirely.
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	c.items = make(map[string]cacheEntry)
	c.mu.Unlock()
}
