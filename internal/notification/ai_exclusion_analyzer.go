package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// AIExclusionAnalyzer uses AI to determine why comments were excluded
type AIExclusionAnalyzer struct {
	config       *config.Config
	claudeClient ai.ClaudeClient
}

// NewAIExclusionAnalyzer creates a new AI-powered exclusion analyzer
func NewAIExclusionAnalyzer(cfg *config.Config) (*AIExclusionAnalyzer, error) {
	client, err := ai.NewRealClaudeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude client: %w", err)
	}

	return &AIExclusionAnalyzer{
		config:       cfg,
		claudeClient: client,
	}, nil
}

// AnalyzeExclusionWithAI uses AI to determine why a comment was excluded
func (aea *AIExclusionAnalyzer) AnalyzeExclusionWithAI(ctx context.Context, comment string, review github.Review) (*ExclusionReason, error) {
	prompt := aea.buildExclusionAnalysisPrompt(comment, review)
	
	response, err := aea.claudeClient.SendMessage(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Parse AI response
	return aea.parseExclusionResponse(response)
}

// buildExclusionAnalysisPrompt creates the prompt for exclusion analysis
func (aea *AIExclusionAnalyzer) buildExclusionAnalysisPrompt(comment string, review github.Review) string {
	// Load project context
	projectContext := aea.loadProjectContext()

	prompt := fmt.Sprintf(`You are analyzing why a GitHub PR review comment was not converted to a task.

Project Context:
%s

Review Comment:
%s

Reviewer: %s
Review State: %s

Analyze why this comment was not converted to an actionable task. Consider:
1. Is it already resolved or addressed?
2. Is it non-actionable (praise, acknowledgment, question)?
3. Is it low priority (nitpick, style suggestion)?
4. Does it violate project policies?
5. Is it out of scope for the current PR?
6. Is it a duplicate of another comment?

Respond with JSON in this exact format:
{
  "type": "string", // One of: "Already Implemented", "Invalid Suggestion", "Low Priority", "Project Policy Violation", "Out of Scope", "Duplicate Suggestion", "Unknown"
  "explanation": "string", // Clear explanation in 1-2 sentences
  "references": ["string"], // Optional: relevant files or policies
  "confidence": 0.0 // Confidence score 0.0-1.0
}

Focus on being accurate and helpful. If unsure, use "Unknown" with lower confidence.`,
		projectContext,
		comment,
		review.Reviewer,
		review.State)

	return prompt
}

// loadProjectContext loads relevant project files for context
func (aea *AIExclusionAnalyzer) loadProjectContext() string {
	var context strings.Builder
	context.WriteString("Project files found:\n")

	// Check for common project files
	projectFiles := []string{
		"CONTRIBUTING.md",
		"CODE_OF_CONDUCT.md", 
		"ARCHITECTURE.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"README.md",
	}

	for _, file := range projectFiles {
		if _, err := os.Stat(file); err == nil {
			context.WriteString(fmt.Sprintf("- %s\n", file))
			
			// Read first few lines for context
			content, err := aea.readFileHead(file, 20)
			if err == nil && content != "" {
				context.WriteString(fmt.Sprintf("  Preview: %s\n", content))
			}
		}
	}

	// Check for project-specific patterns
	if aea.hasGoMod() {
		context.WriteString("- Go project detected\n")
	}
	if aea.hasPackageJSON() {
		context.WriteString("- Node.js project detected\n")
	}

	return context.String()
}

// readFileHead reads the first n lines of a file
func (aea *AIExclusionAnalyzer) readFileHead(filename string, lines int) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	fileLines := strings.Split(string(content), "\n")
	if len(fileLines) > lines {
		fileLines = fileLines[:lines]
	}

	// Truncate long lines and join
	var result []string
	for _, line := range fileLines {
		if len(line) > 100 {
			line = line[:100] + "..."
		}
		result = append(result, line)
	}

	return strings.Join(result, " "), nil
}

// hasGoMod checks if this is a Go project
func (aea *AIExclusionAnalyzer) hasGoMod() bool {
	_, err := os.Stat("go.mod")
	return err == nil
}

// hasPackageJSON checks if this is a Node.js project
func (aea *AIExclusionAnalyzer) hasPackageJSON() bool {
	_, err := os.Stat("package.json")
	return err == nil
}

// parseExclusionResponse parses the AI response into an ExclusionReason
func (aea *AIExclusionAnalyzer) parseExclusionResponse(response string) (*ExclusionReason, error) {
	// Extract JSON from response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	var result struct {
		Type        string   `json:"type"`
		Explanation string   `json:"explanation"`
		References  []string `json:"references"`
		Confidence  float64  `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &ExclusionReason{
		Type:        result.Type,
		Explanation: result.Explanation,
		References:  result.References,
		Confidence:  result.Confidence,
	}, nil
}

// extractJSON extracts JSON content from AI response
func extractJSON(response string) string {
	// Try to find JSON between code blocks
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json")
		if start != -1 {
			start += 7
			end := strings.Index(response[start:], "```")
			if end != -1 {
				return strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// Try to find raw JSON
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start != -1 && end != -1 && start < end {
		return response[start : end+1]
	}

	return ""
}

// EnhanceExclusionReason uses AI to enhance a basic exclusion reason
func (aea *AIExclusionAnalyzer) EnhanceExclusionReason(ctx context.Context, basicReason *ExclusionReason, comment string, review github.Review) (*ExclusionReason, error) {
	// If confidence is already high, no need to enhance
	if basicReason.Confidence >= 0.8 {
		return basicReason, nil
	}

	// Use AI to get better analysis
	aiReason, err := aea.AnalyzeExclusionWithAI(ctx, comment, review)
	if err != nil {
		// Return basic reason if AI fails
		return basicReason, nil
	}

	// Use AI reason if it has higher confidence
	if aiReason.Confidence > basicReason.Confidence {
		return aiReason, nil
	}

	return basicReason, nil
}