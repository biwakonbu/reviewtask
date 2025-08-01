package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/github"
)

func TestChunkerIntegration(t *testing.T) {
	t.Run("Chunk size calculation", func(t *testing.T) {
		// Test with 10KB chunk size
		chunker := NewCommentChunker(10000)
		
		// Create a 36KB comment
		largeText := strings.Repeat("This is a test sentence. ", 1500) // ~36KB
		comment := github.Comment{
			ID:   1,
			Body: largeText,
		}
		
		chunks := chunker.ChunkComment(comment)
		
		// Should be split into multiple chunks
		assert.Greater(t, len(chunks), 1)
		
		// Each chunk should have the part indicator
		for i, chunk := range chunks {
			assert.Contains(t, chunk.Body, "[Part")
			
			// Remove part indicator to check actual content size
			parts := strings.SplitN(chunk.Body, "\n", 2)
			if len(parts) > 1 {
				contentSize := len(parts[1])
				// Each chunk content should be under or around 10KB
				assert.LessOrEqual(t, contentSize, 11000) // Allow some buffer
			}
			
			// All chunks should have same metadata
			assert.Equal(t, comment.ID, chunk.ID)
			
			// Only first chunk should have replies
			if i == 0 && len(comment.Replies) > 0 {
				assert.Equal(t, comment.Replies, chunk.Replies)
			} else {
				assert.Nil(t, chunk.Replies)
			}
		}
	})
	
	t.Run("Exact chunk boundaries", func(t *testing.T) {
		chunker := NewCommentChunker(100)
		
		// Create text that's exactly at boundaries
		text := "First sentence. Second sentence. Third sentence. Fourth sentence. Fifth sentence."
		comment := github.Comment{
			ID:   2,
			Body: text,
		}
		
		chunks := chunker.ChunkComment(comment)
		
		// Verify sentence boundaries are respected
		for i, chunk := range chunks {
			// Each chunk should end with punctuation if possible
			content := chunk.Body
			if strings.Contains(content, "[Part") {
				parts := strings.SplitN(content, "\n", 2)
				if len(parts) > 1 {
					content = parts[1]
				}
			}
			
			// Check that we don't break mid-sentence
			trimmed := strings.TrimSpace(content)
			if len(trimmed) > 0 {
				lastChar := trimmed[len(trimmed)-1]
				// Should end with sentence ending or be the last chunk
				isLastChunk := i == len(chunks)-1
				assert.True(t, lastChar == '.' || lastChar == '!' || lastChar == '?' || isLastChunk)
			}
		}
	})
}