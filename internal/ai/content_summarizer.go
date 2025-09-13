package ai

import (
	"fmt"
	"regexp"
	"strings"

	"reviewtask/internal/github"
)

// ContentSummarizer handles automatic summarization of large comments
type ContentSummarizer struct {
	maxSize     int
	verboseMode bool
}

// NewContentSummarizer creates a new content summarizer
func NewContentSummarizer(maxSize int, verboseMode bool) *ContentSummarizer {
	if maxSize <= 0 {
		maxSize = 20000 // Default to 20KB before summarization
	}
	return &ContentSummarizer{
		maxSize:     maxSize,
		verboseMode: verboseMode,
	}
}

// ShouldSummarize checks if a comment needs summarization
func (cs *ContentSummarizer) ShouldSummarize(comment github.Comment) bool {
	return len(comment.Body) > cs.maxSize
}

// SummarizeComment creates a summarized version of a large comment
func (cs *ContentSummarizer) SummarizeComment(comment github.Comment) github.Comment {
	if !cs.ShouldSummarize(comment) {
		return comment
	}

	if cs.verboseMode {
		fmt.Printf("  📝 Summarizing comment %d (size: %d -> target: %d bytes)\n",
			comment.ID, len(comment.Body), cs.maxSize)
	}

	summarizedBody := cs.createSummary(comment.Body)

	// Create summarized comment
	summarized := comment
	summarized.Body = summarizedBody

	if cs.verboseMode {
		fmt.Printf("  ✅ Summary created (size: %d bytes, reduction: %.1f%%)\n",
			len(summarizedBody),
			100.0*(1.0-float64(len(summarizedBody))/float64(len(comment.Body))))
	}

	return summarized
}

// createSummary generates an intelligent summary of the comment content
func (cs *ContentSummarizer) createSummary(content string) string {
	// Start with the original content
	lines := strings.Split(content, "\n")

	// Extract the most important information
	summary := cs.extractKeyInformation(lines)

	// If still too large, apply aggressive summarization
	if len(summary) > cs.maxSize {
		summary = cs.applyAggressiveSummarization(summary)
	}

	// Add summary indicator
	header := fmt.Sprintf("[SUMMARIZED: Original %d bytes, Summary %d bytes]\n\n",
		len(content), len(summary))

	return header + summary
}

// extractKeyInformation extracts the most relevant parts of the comment
func (cs *ContentSummarizer) extractKeyInformation(lines []string) string {
	var keyLines []string
	var codeBlocks []string
	var issues []string
	var suggestions []string

	inCodeBlock := false
	var currentCodeBlock []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				// End of code block
				if len(currentCodeBlock) > 0 {
					codeBlocks = append(codeBlocks, cs.summarizeCodeBlock(currentCodeBlock))
				}
				currentCodeBlock = nil
				inCodeBlock = false
			} else {
				// Start of code block
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			currentCodeBlock = append(currentCodeBlock, line)
			continue
		}

		// Skip empty lines and very short lines
		if len(line) < 3 {
			continue
		}

		// Categorize important content
		lowLine := strings.ToLower(line)

		// High priority patterns (always include)
		if cs.isHighPriorityLine(lowLine) {
			keyLines = append(keyLines, line)
			continue
		}

		// Issues and problems
		if cs.isIssueLine(lowLine) {
			issues = append(issues, line)
			continue
		}

		// Suggestions and recommendations
		if cs.isSuggestionLine(lowLine) {
			suggestions = append(suggestions, line)
			continue
		}

		// Include other important-looking lines but limit them
		if cs.isImportantLine(lowLine) && len(keyLines) < 10 {
			keyLines = append(keyLines, line)
		}
	}

	// Compile summary sections
	var summaryParts []string

	if len(keyLines) > 0 {
		summaryParts = append(summaryParts, "**Key Points:**\n"+strings.Join(keyLines, "\n"))
	}

	if len(issues) > 0 {
		summaryParts = append(summaryParts, "**Issues:**\n"+strings.Join(issues, "\n"))
	}

	if len(suggestions) > 0 {
		summaryParts = append(summaryParts, "**Suggestions:**\n"+strings.Join(suggestions, "\n"))
	}

	if len(codeBlocks) > 0 {
		summaryParts = append(summaryParts, "**Code Examples:**\n"+strings.Join(codeBlocks, "\n\n"))
	}

	return strings.Join(summaryParts, "\n\n")
}

// applyAggressiveSummarization reduces content further if needed
func (cs *ContentSummarizer) applyAggressiveSummarization(content string) string {
	lines := strings.Split(content, "\n")

	// Keep only the most critical information
	var criticalLines []string
	for _, line := range lines {
		if len(criticalLines)*50 > cs.maxSize { // Rough estimate
			break
		}

		line = strings.TrimSpace(line)
		lowLine := strings.ToLower(line)

		// Only keep critical information
		if cs.isCriticalLine(lowLine) {
			criticalLines = append(criticalLines, line)
		}
	}

	if len(criticalLines) == 0 {
		// Last resort: take first meaningful lines
		for i, line := range lines {
			if i >= 20 { // Limit to first 20 lines
				break
			}
			line = strings.TrimSpace(line)
			if len(line) > 5 && !strings.HasPrefix(line, "#") {
				criticalLines = append(criticalLines, line)
			}
		}
	}

	result := strings.Join(criticalLines, "\n")
	if len(result) > cs.maxSize {
		// Ultimate truncation
		result = result[:cs.maxSize-20] + "\n\n[TRUNCATED...]"
	}

	return result
}

// Helper methods for content classification

func (cs *ContentSummarizer) isHighPriorityLine(line string) bool {
	highPriorityPatterns := []string{
		"security", "vulnerability", "critical", "urgent", "error", "bug",
		"failure", "broken", "crash", "memory leak", "performance",
		"todo:", "fixme:", "hack:", "warning:",
	}

	for _, pattern := range highPriorityPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (cs *ContentSummarizer) isIssueLine(line string) bool {
	issuePatterns := []string{
		"issue:", "problem:", "incorrect", "wrong", "missing", "fail",
		"doesn't work", "not working", "broken", "error in",
	}

	for _, pattern := range issuePatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (cs *ContentSummarizer) isSuggestionLine(line string) bool {
	suggestionPatterns := []string{
		"suggest", "recommend", "should", "could", "might want to",
		"consider", "perhaps", "maybe", "improvement", "better",
	}

	for _, pattern := range suggestionPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (cs *ContentSummarizer) isImportantLine(line string) bool {
	// Lines that are likely to be important
	if len(line) < 10 || len(line) > 200 {
		return false
	}

	// Contains keywords that suggest importance
	importantKeywords := []string{
		"important", "note", "remember", "key", "main", "primary",
		"essential", "required", "necessary", "must", "need",
	}

	for _, keyword := range importantKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}

	// Bullet points or numbered lists
	if regexp.MustCompile(`^\s*[*\-+•]\s`).MatchString(line) ||
		regexp.MustCompile(`^\s*\d+[.)]\s`).MatchString(line) {
		return true
	}

	// Questions
	if strings.HasSuffix(line, "?") {
		return true
	}

	return false
}

func (cs *ContentSummarizer) isCriticalLine(line string) bool {
	return cs.isHighPriorityLine(line) || cs.isIssueLine(line)
}

func (cs *ContentSummarizer) summarizeCodeBlock(lines []string) string {
	if len(lines) <= 5 {
		// Short code block, include as-is
		return "```\n" + strings.Join(lines, "\n") + "\n```"
	}

	// For longer blocks, show first few lines and summary
	preview := lines[:3]
	summary := fmt.Sprintf("[Code block with %d lines - showing first 3]\n```\n%s\n...\n```",
		len(lines), strings.Join(preview, "\n"))

	return summary
}