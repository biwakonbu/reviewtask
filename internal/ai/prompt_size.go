package ai

import (
	"fmt"
	"sort"
	"strings"

	"reviewtask/internal/github"
)

// PromptSizeTracker tracks the size of different prompt components
type PromptSizeTracker struct {
	SystemPrompt     int
	LanguagePrompt   int
	PriorityPrompt   int
	NitpickPrompt    int
	ReviewsData      int
	ReviewBodies     int
	ReviewComments   int
	TotalSize        int
	Limit            int
	LargestComment   CommentSize
	ReviewBreakdown  []ReviewSize
	CommentBreakdown []CommentSize
}

// ReviewSize represents size information for a single review
type ReviewSize struct {
	ReviewID     int64
	ReviewerName string
	BodySize     int
	CommentsSize int
	TotalSize    int
}

// CommentSize represents size information for a single comment
type CommentSize struct {
	CommentID int64
	Author    string
	FileInfo  string
	Size      int
}

// NewPromptSizeTracker creates a new tracker with the default limit
func NewPromptSizeTracker() *PromptSizeTracker {
	return &PromptSizeTracker{
		Limit:            32 * 1024, // 32KB
		ReviewBreakdown:  make([]ReviewSize, 0),
		CommentBreakdown: make([]CommentSize, 0),
	}
}

// TrackSystemPrompt tracks the system prompt component
func (t *PromptSizeTracker) TrackSystemPrompt(prompt string) {
	t.SystemPrompt = len(prompt)
	t.updateTotal()
}

// TrackLanguagePrompt tracks the language instruction component
func (t *PromptSizeTracker) TrackLanguagePrompt(prompt string) {
	t.LanguagePrompt = len(prompt)
	t.updateTotal()
}

// TrackPriorityPrompt tracks the priority rules component
func (t *PromptSizeTracker) TrackPriorityPrompt(prompt string) {
	t.PriorityPrompt = len(prompt)
	t.updateTotal()
}

// TrackNitpickPrompt tracks the nitpick handling component
func (t *PromptSizeTracker) TrackNitpickPrompt(prompt string) {
	t.NitpickPrompt = len(prompt)
	t.updateTotal()
}

// TrackReviewsData tracks the reviews data and provides breakdown
func (t *PromptSizeTracker) TrackReviewsData(data string, reviews []github.Review) {
	t.ReviewsData = len(data)
	t.analyzeReviews(reviews)
	t.updateTotal()
}

// analyzeReviews provides detailed breakdown of review sizes
func (t *PromptSizeTracker) analyzeReviews(reviews []github.Review) {
	t.ReviewBreakdown = make([]ReviewSize, 0, len(reviews))
	t.CommentBreakdown = make([]CommentSize, 0)
	t.ReviewBodies = 0
	t.ReviewComments = 0

	for _, review := range reviews {
		reviewSize := ReviewSize{
			ReviewID:     review.ID,
			ReviewerName: review.Reviewer,
		}

		// Track review body size
		if review.Body != "" {
			bodySize := len(review.Body)
			reviewSize.BodySize = bodySize
			t.ReviewBodies += bodySize
		}

		// Track comments size
		for _, comment := range review.Comments {
			commentSize := len(comment.Body)

			// Add replies size
			for _, reply := range comment.Replies {
				commentSize += len(reply.Body) + len(reply.Author) + 10 // Account for formatting
			}

			reviewSize.CommentsSize += commentSize
			t.ReviewComments += commentSize

			// Track individual comment
			cs := CommentSize{
				CommentID: comment.ID,
				Author:    comment.Author,
				FileInfo:  fmt.Sprintf("%s:%d", comment.File, comment.Line),
				Size:      commentSize,
			}
			t.CommentBreakdown = append(t.CommentBreakdown, cs)

			// Track largest comment
			if cs.Size > t.LargestComment.Size {
				t.LargestComment = cs
			}
		}

		reviewSize.TotalSize = reviewSize.BodySize + reviewSize.CommentsSize
		t.ReviewBreakdown = append(t.ReviewBreakdown, reviewSize)
	}
}

// updateTotal updates the total size
func (t *PromptSizeTracker) updateTotal() {
	t.TotalSize = t.SystemPrompt + t.LanguagePrompt + t.PriorityPrompt +
		t.NitpickPrompt + t.ReviewsData
}

// IsExceeded returns true if the total size exceeds the limit
func (t *PromptSizeTracker) IsExceeded() bool {
	return t.TotalSize > t.Limit
}

// GetLargestComponent returns the name and size of the largest component
func (t *PromptSizeTracker) GetLargestComponent() (string, int) {
	components := map[string]int{
		"System prompt":     t.SystemPrompt,
		"Language settings": t.LanguagePrompt,
		"Priority rules":    t.PriorityPrompt,
		"Nitpick rules":     t.NitpickPrompt,
		"Review data":       t.ReviewsData,
	}

	var largest string = "System prompt" // Default to first component
	var largestSize int
	for name, size := range components {
		if size > largestSize {
			largest = name
			largestSize = size
		}
	}

	return largest, largestSize
}

// GenerateReport generates a detailed size breakdown report
func (t *PromptSizeTracker) GenerateReport() string {
	var report strings.Builder

	report.WriteString("Prompt size breakdown:\n")

	// Component breakdown
	components := []struct {
		name string
		size int
	}{
		{"System prompt", t.SystemPrompt},
		{"Language settings", t.LanguagePrompt},
		{"Priority rules", t.PriorityPrompt},
		{"Nitpick rules", t.NitpickPrompt},
		{"Review data", t.ReviewsData},
	}

	largestComponent, _ := t.GetLargestComponent()

	for _, comp := range components {
		percentage := float64(comp.size) / float64(t.TotalSize) * 100
		marker := ""
		if comp.name == largestComponent {
			marker = " ← LARGEST"
		}
		report.WriteString(fmt.Sprintf("  %-18s %7d bytes (%4.1f%%)%s\n",
			comp.name+":", comp.size, percentage, marker))
	}

	report.WriteString("  ─────────────────────────────────────────\n")
	report.WriteString(fmt.Sprintf("  Total:             %7d bytes (limit: %d bytes)\n",
		t.TotalSize, t.Limit))

	// Review data breakdown if it's the largest
	if largestComponent == "Review data" && len(t.ReviewBreakdown) > 0 {
		report.WriteString("\n📊 Review data breakdown:\n")
		report.WriteString(fmt.Sprintf("  - Review bodies: %d reviews, %d bytes\n",
			len(t.ReviewBreakdown), t.ReviewBodies))
		report.WriteString(fmt.Sprintf("  - Comments: %d comments, %d bytes\n",
			len(t.CommentBreakdown), t.ReviewComments))

		if t.LargestComment.Size > 0 {
			report.WriteString(fmt.Sprintf("  - Largest comment: ID %d (%d bytes)\n",
				t.LargestComment.CommentID, t.LargestComment.Size))
		}
	}

	return report.String()
}

// GenerateSuggestions generates optimization suggestions based on the size analysis
func (t *PromptSizeTracker) GenerateSuggestions() string {
	if !t.IsExceeded() {
		return ""
	}

	var suggestions strings.Builder
	suggestions.WriteString("\n💡 Suggestions:\n")

	largestComponent, _ := t.GetLargestComponent()
	excessSize := t.TotalSize - t.Limit

	switch largestComponent {
	case "Review data":
		suggestions.WriteString("  1. The review data is too large. Consider:\n")
		suggestions.WriteString("     - Processing reviews in smaller batches\n")
		suggestions.WriteString("     - Using incremental processing (already enabled)\n")

		if t.LargestComment.Size > 5000 {
			suggestions.WriteString("     - Truncating very long comments\n")
		}

		suggestions.WriteString(fmt.Sprintf("\n  2. This PR has %d comments. The system will automatically:\n",
			len(t.CommentBreakdown)))
		suggestions.WriteString("     - Use parallel processing for individual comments\n")
		suggestions.WriteString("     - Process in batches to avoid size limits\n")

	case "System prompt":
		suggestions.WriteString("  1. The system prompt is unusually large.\n")
		suggestions.WriteString("     - Review the task generation instructions\n")
		suggestions.WriteString("     - Consider simplifying the prompt template\n")

	default:
		suggestions.WriteString(fmt.Sprintf("  1. The %s component is too large.\n", largestComponent))
		suggestions.WriteString(fmt.Sprintf("     - Need to reduce by at least %d bytes\n", excessSize))
	}

	// Show top 3 largest reviews if relevant
	if largestComponent == "Review data" && len(t.ReviewBreakdown) > 3 {
		sort.Slice(t.ReviewBreakdown, func(i, j int) bool {
			return t.ReviewBreakdown[i].TotalSize > t.ReviewBreakdown[j].TotalSize
		})

		suggestions.WriteString("\n  3. Largest reviews:\n")
		for i := 0; i < 3 && i < len(t.ReviewBreakdown); i++ {
			r := t.ReviewBreakdown[i]
			suggestions.WriteString(fmt.Sprintf("     - Review %d by %s: %d bytes\n",
				r.ReviewID, r.ReviewerName, r.TotalSize))
		}
	}

	return suggestions.String()
}

// GenerateErrorMessage creates a detailed error message for prompt size exceeded
func (t *PromptSizeTracker) GenerateErrorMessage() string {
	var msg strings.Builder

	msg.WriteString("❌ Prompt size limit exceeded!\n\n")
	msg.WriteString(t.GenerateReport())
	msg.WriteString(t.GenerateSuggestions())

	return msg.String()
}
