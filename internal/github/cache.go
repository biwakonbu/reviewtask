package github

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry represents a cached API response
type CacheEntry struct {
	Data      interface{} `json:"data"`
	CachedAt  time.Time   `json:"cached_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// APICache handles caching of GitHub API responses
type APICache struct {
	cacheDir string
	ttl      time.Duration
}

// NewAPICache creates a new API cache
func NewAPICache(ttl time.Duration) *APICache {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cache", "reviewtask", "github-api")
	
	// Ensure cache directory exists
	os.MkdirAll(cacheDir, 0755)
	
	return &APICache{
		cacheDir: cacheDir,
		ttl:      ttl,
	}
}

// getCacheKey generates a cache key for the given parameters
func (c *APICache) getCacheKey(method, owner, repo string, params ...interface{}) string {
	key := fmt.Sprintf("%s-%s-%s", method, owner, repo)
	for _, p := range params {
		key += fmt.Sprintf("-%v", p)
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}

// Get retrieves a cached response if available and not expired
func (c *APICache) Get(method, owner, repo string, params ...interface{}) (interface{}, bool) {
	key := c.getCacheKey(method, owner, repo, params...)
	cachePath := filepath.Join(c.cacheDir, key+".json")
	
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}
	
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	
	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Delete expired cache
		os.Remove(cachePath)
		return nil, false
	}
	
	return entry.Data, true
}

// Set stores a response in the cache
func (c *APICache) Set(method, owner, repo string, data interface{}, params ...interface{}) error {
	key := c.getCacheKey(method, owner, repo, params...)
	cachePath := filepath.Join(c.cacheDir, key+".json")
	
	entry := CacheEntry{
		Data:      data,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(c.ttl),
	}
	
	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(cachePath, jsonData, 0644)
}

// Clear removes all cached entries
func (c *APICache) Clear() error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			os.Remove(filepath.Join(c.cacheDir, entry.Name()))
		}
	}
	
	return nil
}

// ClearExpired removes only expired cache entries
func (c *APICache) ClearExpired() error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		cachePath := filepath.Join(c.cacheDir, entry.Name())
		data, err := os.ReadFile(cachePath)
		if err != nil {
			continue
		}
		
		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			continue
		}
		
		if time.Now().After(cacheEntry.ExpiresAt) {
			os.Remove(cachePath)
		}
	}
	
	return nil
}