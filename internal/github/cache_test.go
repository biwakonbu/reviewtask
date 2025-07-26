package github

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPICache(t *testing.T) {
	// Create temporary cache directory
	tempDir, err := os.MkdirTemp("", "reviewtask-cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create cache with custom directory
	cache := &APICache{
		cacheDir: tempDir,
		ttl:      2 * time.Second, // Short TTL for testing
	}

	t.Run("SetAndGet", func(t *testing.T) {
		// Test data
		owner := "testowner"
		repo := "testrepo"
		prNumber := 123
		testData := map[string]interface{}{
			"number": prNumber,
			"title":  "Test PR",
			"state":  "open",
		}

		// Set cache
		err := cache.Set("GetPRInfo", owner, repo, testData, prNumber)
		assert.NoError(t, err)

		// Get from cache
		cached, found := cache.Get("GetPRInfo", owner, repo, prNumber)
		assert.True(t, found)
		assert.NotNil(t, cached)

		// Verify data
		cachedMap, ok := cached.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, float64(prNumber), cachedMap["number"]) // JSON unmarshal converts to float64
		assert.Equal(t, "Test PR", cachedMap["title"])
		assert.Equal(t, "open", cachedMap["state"])
	})

	t.Run("CacheMiss", func(t *testing.T) {
		// Try to get non-existent cache entry
		cached, found := cache.Get("NonExistent", "owner", "repo", 999)
		assert.False(t, found)
		assert.Nil(t, cached)
	})

	t.Run("CacheExpiration", func(t *testing.T) {
		// Set cache with short TTL
		testData := "test data"
		err := cache.Set("ExpireTest", "owner", "repo", testData)
		assert.NoError(t, err)

		// Verify it's cached
		cached, found := cache.Get("ExpireTest", "owner", "repo")
		assert.True(t, found)
		assert.Equal(t, testData, cached)

		// Wait for expiration
		time.Sleep(3 * time.Second)

		// Should be expired
		cached, found = cache.Get("ExpireTest", "owner", "repo")
		assert.False(t, found)
		assert.Nil(t, cached)
	})

	t.Run("DifferentParameters", func(t *testing.T) {
		// Set cache for different parameters
		err := cache.Set("Method", "owner1", "repo1", "data1", 1)
		assert.NoError(t, err)

		err = cache.Set("Method", "owner1", "repo1", "data2", 2)
		assert.NoError(t, err)

		err = cache.Set("Method", "owner2", "repo1", "data3", 1)
		assert.NoError(t, err)

		// Verify each cache entry is separate
		cached, found := cache.Get("Method", "owner1", "repo1", 1)
		assert.True(t, found)
		assert.Equal(t, "data1", cached)

		cached, found = cache.Get("Method", "owner1", "repo1", 2)
		assert.True(t, found)
		assert.Equal(t, "data2", cached)

		cached, found = cache.Get("Method", "owner2", "repo1", 1)
		assert.True(t, found)
		assert.Equal(t, "data3", cached)
	})

	t.Run("ComplexData", func(t *testing.T) {
		// Test with complex nested data structure
		reviews := []Review{
			{
				ID:       100,
				Reviewer: "user1",
				State:    "APPROVED",
				Body:     "LGTM",
				Comments: []Comment{
					{
						ID:   200,
						File: "main.go",
						Line: 42,
						Body: "Consider adding error handling",
					},
				},
			},
		}

		err := cache.Set("GetPRReviews", "owner", "repo", reviews, 123)
		assert.NoError(t, err)

		cached, found := cache.Get("GetPRReviews", "owner", "repo", 123)
		assert.True(t, found)

		// The cached data will be unmarshaled as []interface{} due to JSON
		cachedSlice, ok := cached.([]interface{})
		assert.True(t, ok)
		assert.Len(t, cachedSlice, 1)
	})

	t.Run("Clear", func(t *testing.T) {
		// Add multiple cache entries
		cache.Set("Method1", "owner", "repo", "data1")
		cache.Set("Method2", "owner", "repo", "data2")
		cache.Set("Method3", "owner", "repo", "data3")

		// Verify they exist
		_, found := cache.Get("Method1", "owner", "repo")
		assert.True(t, found)

		// Clear all cache
		err := cache.Clear()
		assert.NoError(t, err)

		// Verify all are gone
		_, found = cache.Get("Method1", "owner", "repo")
		assert.False(t, found)
		_, found = cache.Get("Method2", "owner", "repo")
		assert.False(t, found)
		_, found = cache.Get("Method3", "owner", "repo")
		assert.False(t, found)
	})

	t.Run("ClearExpired", func(t *testing.T) {
		// Create cache with different TTLs
		shortCache := &APICache{
			cacheDir: tempDir,
			ttl:      1 * time.Second,
		}
		longCache := &APICache{
			cacheDir: tempDir,
			ttl:      10 * time.Second,
		}

		// Set entries with different TTLs
		shortCache.Set("ShortLived", "owner", "repo", "data1")
		longCache.Set("LongLived", "owner", "repo", "data2")

		// Wait for short-lived to expire
		time.Sleep(2 * time.Second)

		// Clear expired entries
		err := cache.ClearExpired()
		assert.NoError(t, err)

		// Short-lived should be gone
		_, found := cache.Get("ShortLived", "owner", "repo")
		assert.False(t, found)

		// Long-lived should still exist
		_, found = longCache.Get("LongLived", "owner", "repo")
		assert.True(t, found)
	})

	t.Run("CacheKeyGeneration", func(t *testing.T) {
		// Test that different parameters generate different keys
		key1 := cache.getCacheKey("Method", "owner", "repo", 1, 2, 3)
		key2 := cache.getCacheKey("Method", "owner", "repo", 1, 2, 4)
		key3 := cache.getCacheKey("Method", "owner", "repo2", 1, 2, 3)
		key4 := cache.getCacheKey("Method2", "owner", "repo", 1, 2, 3)

		// All keys should be different
		assert.NotEqual(t, key1, key2)
		assert.NotEqual(t, key1, key3)
		assert.NotEqual(t, key1, key4)
		assert.NotEqual(t, key2, key3)
		assert.NotEqual(t, key2, key4)
		assert.NotEqual(t, key3, key4)

		// Same parameters should generate same key
		key5 := cache.getCacheKey("Method", "owner", "repo", 1, 2, 3)
		assert.Equal(t, key1, key5)
	})
}

func TestNewAPICache(t *testing.T) {
	// Test default cache creation
	cache := NewAPICache(5 * time.Minute)
	assert.NotNil(t, cache)
	assert.Equal(t, 5*time.Minute, cache.ttl)

	// Verify cache directory exists
	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, ".cache", "reviewtask", "github-api")
	assert.DirExists(t, expectedDir)

	// Clean up
	os.RemoveAll(filepath.Join(homeDir, ".cache", "reviewtask"))
}
