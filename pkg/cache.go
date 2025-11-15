package pkg

import (
	"html/template"
	"sync"
	"time"
)

type Manager struct {
	Cache      []Cache
	mu         sync.RWMutex
	stopCh     chan struct{}
	cleanupWG  sync.WaitGroup
}

// CleanupInterval defines how often to clean up expired entries (e.g., every 5 minutes)
const CleanupInterval = 5 * time.Minute

type Cache struct {
	ID          string
	Title       string
	Favicon     string
	Description string
	Path        string
	HTML        *template.Template
	Body        string
	CSS         string
	JS          string
	CSSLinks    []template.HTML
	Expiration  int64 // Unix timestamp for expiration
}

// NewManager initializes a new Manager instance
func NewManager() *Manager {
	manager := &Manager{
		Cache:  make([]Cache, 0),
		stopCh: make(chan struct{}),
	}

	// Start the background cleanup goroutine
	manager.cleanupWG.Add(1)
	go manager.startCleanup()

	return manager
}

// AddCache adds a new cache entry with a default TTL of 10 minutes
func (m *Manager) AddCache(cache Cache) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Cache = append(m.Cache, cache)
}

// GetCache retrieves a cache entry by ID if it hasn’t expired
func (m *Manager) GetCache(path string) (Cache, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cache := range m.Cache {
		if cache.Path == path && cache.Expiration > time.Now().Unix() {
			return cache, true
		}
	}
	return Cache{}, false
}

// DeleteExpired removes expired entries from the Cache
func (m *Manager) DeleteExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	m.Cache = filterExpired(m.Cache, now)
}

// startCleanup runs in the background to periodically remove expired cache entries
func (m *Manager) startCleanup() {
	defer m.cleanupWG.Done()

	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.DeleteExpired()
		case <-m.stopCh:
			return
		}
	}
}

// Close stops the background cleanup goroutine and waits for it to finish
func (m *Manager) Close() {
	close(m.stopCh)
	m.cleanupWG.Wait()
}

// filterExpired is a helper function that filters out expired cache entries
func filterExpired(caches []Cache, currentTime int64) []Cache {
	filtered := make([]Cache, 0)
	for _, cache := range caches {
		if cache.Expiration > currentTime {
			filtered = append(filtered, cache)
		}
	}
	return filtered
}
