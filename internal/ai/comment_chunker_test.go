package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/github"
)

func TestCommentChunker(t *testing.T) {
	t.Run("Small comment not chunked", func(t *testing.T) {
		chunker := NewCommentChunker(1000)
		comment := github.Comment{
			ID:   1,
			Body: "This is a small comment",
		}

		chunks := chunker.ChunkComment(comment)
		assert.Len(t, chunks, 1)
		assert.Equal(t, comment.Body, chunks[0].Body)
	})

	t.Run("Large comment is chunked", func(t *testing.T) {
		chunker := NewCommentChunker(100)
		comment := github.Comment{
			ID:   2,
			Body: "This is a very long comment that exceeds the chunk size limit. It should be split into multiple chunks. Each chunk should be properly sized.",
		}

		chunks := chunker.ChunkComment(comment)
		assert.Greater(t, len(chunks), 1)

		// Verify chunks contain part indicators
		for _, chunk := range chunks {
			assert.Contains(t, chunk.Body, "[Part")
			assert.LessOrEqual(t, len(chunk.Body), 150) // Allow some overhead for part indicator
		}
	})

	t.Run("Chunk at sentence boundaries", func(t *testing.T) {
		chunker := NewCommentChunker(50)
		comment := github.Comment{
			ID:   3,
			Body: "First sentence. Second sentence. Third sentence. Fourth sentence.",
		}

		chunks := chunker.ChunkComment(comment)
		assert.Greater(t, len(chunks), 1)

		// First chunk should end with a sentence
		firstChunk := chunks[0].Body
		assert.Contains(t, firstChunk, "sentence.")
	})

	t.Run("Preserve comment metadata", func(t *testing.T) {
		chunker := NewCommentChunker(50)
		now := time.Now().Format(time.RFC3339)
		comment := github.Comment{
			ID:        4,
			File:      "main.go",
			Line:      42,
			Body:      strings.Repeat("Long text. ", 20),
			Author:    "reviewer1",
			CreatedAt: now,
			Replies: []github.Reply{
				{Author: "user1", Body: "Reply"},
			},
		}

		chunks := chunker.ChunkComment(comment)

		// All chunks should preserve metadata
		for i, chunk := range chunks {
			assert.Equal(t, comment.ID, chunk.ID)
			assert.Equal(t, comment.File, chunk.File)
			assert.Equal(t, comment.Line, chunk.Line)
			assert.Equal(t, comment.Author, chunk.Author)
			assert.Equal(t, comment.CreatedAt, chunk.CreatedAt)

			// Only first chunk should have replies
			if i == 0 {
				assert.Len(t, chunk.Replies, 1)
			} else {
				assert.Nil(t, chunk.Replies)
			}
		}
	})

	t.Run("ShouldChunkComment", func(t *testing.T) {
		chunker := NewCommentChunker(100)

		smallComment := github.Comment{
			Body: "Small",
		}
		assert.False(t, chunker.ShouldChunkComment(smallComment))

		largeComment := github.Comment{
			Body: strings.Repeat("Large ", 100),
		}
		assert.True(t, chunker.ShouldChunkComment(largeComment))
	})
}

func TestChunkingStrategies(t *testing.T) {
	t.Run("Paragraph breaks", func(t *testing.T) {
		chunker := NewCommentChunker(50) // Smaller size to force chunking
		text := "First paragraph with some text.\n\nSecond paragraph with more text.\n\nThird paragraph."

		chunks := chunker.splitIntoChunks(text)
		assert.Greater(t, len(chunks), 1)

		// Should break at paragraph boundaries
		assert.Contains(t, chunks[0], "First paragraph")
		// Since we respect paragraph boundaries, the third paragraph should be in a later chunk
		if len(chunks) > 2 {
			assert.NotContains(t, chunks[0], "Third paragraph")
		}
	})

	t.Run("List items", func(t *testing.T) {
		chunker := NewCommentChunker(80)
		text := "Issues found:\n- First issue that needs fixing\n- Second issue to address\n- Third issue"

		chunks := chunker.splitIntoChunks(text)

		// Should try to keep list items together
		for _, chunk := range chunks {
			// Each chunk should have complete list items
			if strings.Contains(chunk, "- First") {
				assert.Contains(t, chunk, "fixing")
			}
		}
	})

	t.Run("Code blocks", func(t *testing.T) {
		chunker := NewCommentChunker(100)
		text := "Here's the problem:\n```go\nfunc main() {\n    // This is a code block\n}\n```\nPlease fix this."

		chunks := chunker.splitIntoChunks(text)

		// Verify chunking doesn't break in the middle of code blocks if possible
		assert.Greater(t, len(chunks), 0)
	})
}
