package projectctx

import "sync"

// cacheEntry wraps a ProjectMetadata with storage metadata.
type cacheEntry struct {
	meta *ProjectMetadata
}

// ContextCache stores ProjectMetadata in memory to avoid redundant scans.
// It is safe for concurrent use.
type ContextCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

// NewContextCache creates a new empty ContextCache.
func NewContextCache() *ContextCache {
	return &ContextCache{
		entries: make(map[string]*cacheEntry),
	}
}

// Get returns cached ProjectMetadata for the given workspace dir, or nil if not cached.
func (cc *ContextCache) Get(workspaceDir string) *ProjectMetadata {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	e, ok := cc.entries[workspaceDir]
	if !ok {
		return nil
	}
	return e.meta
}

// Put stores ProjectMetadata for the given workspace dir.
// The ProjectMetadata.ScannedAt field serves as the cache timestamp.
func (cc *ContextCache) Put(workspaceDir string, meta *ProjectMetadata) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.entries[workspaceDir] = &cacheEntry{meta: meta}
}

// Invalidate removes the cached entry for the given workspace dir.
func (cc *ContextCache) Invalidate(workspaceDir string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	delete(cc.entries, workspaceDir)
}
