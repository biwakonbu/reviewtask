package ai

import (
	"fmt"
	"strings"
	"reviewtask/internal/github"
)

// CommentChunker handles splitting large comments into manageable chunks
type CommentChunker struct {
	maxChunkSize int
}

// NewCommentChunker creates a new comment chunker
func NewCommentChunker(maxSize int) *CommentChunker {
	if maxSize <= 0 {
		maxSize = 10000 // Default to 10KB chunks
	}
	return &CommentChunker{
		maxChunkSize: maxSize,
	}
}

// ChunkComment splits a large comment into smaller chunks if needed
func (c *CommentChunker) ChunkComment(comment github.Comment) []github.Comment {
	if len(comment.Body) <= c.maxChunkSize {
		return []github.Comment{comment}
	}

	// Split the comment body into chunks
	chunks := c.splitIntoChunks(comment.Body)
	result := make([]github.Comment, len(chunks))

	for i, chunk := range chunks {
		result[i] = github.Comment{
			ID:        comment.ID,
			File:      comment.File,
			Line:      comment.Line,
			Body:      chunk,
			Author:    comment.Author,
			CreatedAt: comment.CreatedAt,
			Replies:   comment.Replies, // Include replies only in first chunk
		}
		
		// Add chunk indicator
		if len(chunks) > 1 {
			result[i].Body = fmt.Sprintf("[Part %d/%d]\n%s", i+1, len(chunks), chunk)
		}
		
		// Only include replies in the first chunk
		if i > 0 {
			result[i].Replies = nil
		}
	}

	return result
}

// splitIntoChunks splits text into chunks at sentence boundaries when possible
func (c *CommentChunker) splitIntoChunks(text string) []string {
	if len(text) <= c.maxChunkSize {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= c.maxChunkSize {
			chunks = append(chunks, remaining)
			break
		}

		// Try to find a good break point (sentence end, paragraph, etc.)
		chunkEnd := c.findBreakPoint(remaining, c.maxChunkSize)
		
		chunk := strings.TrimSpace(remaining[:chunkEnd])
		chunks = append(chunks, chunk)
		remaining = strings.TrimSpace(remaining[chunkEnd:])
	}

	return chunks
}

// findBreakPoint finds a good point to break the text
func (c *CommentChunker) findBreakPoint(text string, maxPos int) int {
	if maxPos >= len(text) {
		return len(text)
	}

	// Look for sentence endings first
	sentenceEnds := []string{". ", ".\n", "! ", "!\n", "? ", "?\n"}
	bestPos := c.findLastOccurrence(text[:maxPos], sentenceEnds)
	if bestPos > maxPos/2 { // If we found a sentence end in the second half
		return bestPos + 1
	}

	// Look for paragraph breaks
	paragraphBreaks := []string{"\n\n", "\n-", "\n*", "\n1.", "\n2.", "\n3."}
	bestPos = c.findLastOccurrence(text[:maxPos], paragraphBreaks)
	if bestPos > maxPos/2 {
		return bestPos + 1
	}

	// Look for any newline
	newlinePos := strings.LastIndex(text[:maxPos], "\n")
	if newlinePos > maxPos/2 {
		return newlinePos + 1
	}

	// Look for word boundaries
	spacePos := strings.LastIndex(text[:maxPos], " ")
	if spacePos > 0 {
		return spacePos + 1
	}

	// Fallback: just cut at maxPos
	return maxPos
}

// findLastOccurrence finds the last occurrence of any delimiter
func (c *CommentChunker) findLastOccurrence(text string, delimiters []string) int {
	bestPos := -1
	for _, delim := range delimiters {
		pos := strings.LastIndex(text, delim)
		if pos > bestPos {
			bestPos = pos
		}
	}
	return bestPos
}

// ShouldChunkComment determines if a comment needs chunking
func (c *CommentChunker) ShouldChunkComment(comment github.Comment) bool {
	// Check if the comment body alone exceeds the chunk size
	return len(comment.Body) > c.maxChunkSize
}