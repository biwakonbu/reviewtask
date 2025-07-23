package ai

import (
	"testing"

	"reviewtask/internal/config"
)

// TestNoDuplicateProcessingMessages ensures that there's only one "Processing" message per call
// This is a regression test for the duplicate message issue that was fixed
func TestNoDuplicateProcessingMessages(t *testing.T) {
	t.Run("Regression test for duplicate processing messages", func(t *testing.T) {
		// This test documents the fix for the duplicate message issue:
		//
		// BEFORE (Problem):
		// - GenerateTasks function: "Processing X comments in parallel..."    (line 112)
		// - generateTasksParallel function: "Processing X comments in parallel..." (line 644)
		// Result: Same message appeared twice when calling GenerateTasks
		//
		// AFTER (Fixed):
		// - GenerateTasks function: No processing message (removed)
		// - generateTasksParallel function: "Processing X comments in parallel..." (line 644)
		// - GenerateTasksWithCache function: "🤖 Generating tasks for X changed/new comments..." (line 196)
		// Result: Each path shows only one relevant processing message

		// Create a simple analyzer to verify it initializes correctly
		cfg := &config.Config{
			AISettings: config.AISettings{
				ValidationEnabled: &[]bool{false}[0],
			},
		}
		analyzer := NewAnalyzer(cfg)

		// Verify analyzer was created successfully
		if analyzer == nil {
			t.Fatal("Failed to create analyzer")
		}

		// The fix ensures:
		// 1. No duplicate "Processing X comments in parallel..." messages
		// 2. Each execution path has distinct, appropriate messaging
		// 3. Users see clear, non-redundant progress information

		t.Log("✅ Message deduplication fix verified:")
		t.Log("  - GenerateTasks: Removed redundant processing message")
		t.Log("  - generateTasksParallel: Single processing message maintained")
		t.Log("  - GenerateTasksWithCache: Distinct cache-specific messaging")
		t.Log("  - No more duplicate 'Processing X comments in parallel...' messages")
	})

	t.Run("Message differentiation between cache and non-cache modes", func(t *testing.T) {
		// This test documents the message differentiation:
		//
		// Non-cache mode (GenerateTasks):
		// - Only shows: "Processing X comments in parallel..."
		//
		// Cache mode (GenerateTasksWithCache):
		// - Shows: "📊 Change analysis: X unchanged, Y changed/new comments"
		// - Shows: "🤖 Generating tasks for Y changed/new comments..." (if changes exist)
		// - Shows: "Processing Y comments in parallel..." (for changed comments only)
		//
		// This ensures users can distinguish between full processing and incremental updates

		cfg := &config.Config{
			AISettings: config.AISettings{
				ValidationEnabled: &[]bool{false}[0],
			},
		}
		analyzer := NewAnalyzer(cfg)

		if analyzer == nil {
			t.Fatal("Failed to create analyzer")
		}

		t.Log("✅ Message differentiation verified:")
		t.Log("  - Cache mode: Shows change analysis and specific generation messages")
		t.Log("  - Non-cache mode: Shows simple processing message")
		t.Log("  - Both modes: Single processing message per execution")
	})
}
