package ai

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reviewtask/internal/config"
	"reviewtask/internal/storage"
	"strings"
)

// SemanticAnalyzer handles semantic analysis of comment changes
type SemanticAnalyzer struct {
	config *config.Config
}

// NewSemanticAnalyzer creates a new semantic analyzer
func NewSemanticAnalyzer(config *config.Config) *SemanticAnalyzer {
	return &SemanticAnalyzer{
		config: config,
	}
}

// SemanticChangeRequest represents a request to check if two texts have semantic differences
type SemanticChangeRequest struct {
	OriginalText string `json:"original_text"`
	NewText      string `json:"new_text"`
}

// SemanticChangeResponse represents the AI's response about semantic changes
type SemanticChangeResponse struct {
	HasSemanticChange bool   `json:"has_semantic_change"`
	Explanation       string `json:"explanation"`
	ChangeType        string `json:"change_type"` // "cosmetic", "clarification", "semantic", "major"
}

// AnalyzeSemanticChange uses AI to determine if a comment change is semantically significant
func (s *SemanticAnalyzer) AnalyzeSemanticChange(originalText, newText string) (*SemanticChangeResponse, error) {
	// If texts are identical, no semantic change
	if originalText == newText {
		return &SemanticChangeResponse{
			HasSemanticChange: false,
			Explanation:      "Texts are identical",
			ChangeType:       "none",
		}, nil
	}

	prompt := fmt.Sprintf(`Analyze if these two review comments have semantic differences that would require regenerating tasks.

Original comment:
%s

New comment:
%s

Determine if the change is:
- "cosmetic": Only formatting, typos, or minor wording changes that don't affect meaning
- "clarification": Better explanation but same core request
- "semantic": Changes that affect what tasks should be generated
- "major": Completely different request or significant scope change

Respond in JSON format:
{
  "has_semantic_change": boolean (true if tasks need regeneration),
  "explanation": "Brief explanation of the change",
  "change_type": "cosmetic|clarification|semantic|major"
}`, originalText, newText)

	claudeCmd, err := FindClaudeCommand(s.config.AISettings.ClaudePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Claude: %w", err)
	}

	cmd := exec.Command(claudeCmd,
		"--output-format", "json",
		prompt)

	// Set environment for Claude command
	cmd.Env = append(cmd.Environ(), "TERM=dumb", "NO_COLOR=1")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run claude: %w", err)
	}

	// Parse JSON response
	var response SemanticChangeResponse
	responseStr := strings.TrimSpace(string(output))
	
	if err := json.Unmarshal([]byte(responseStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w\nResponse: %s", err, responseStr)
	}

	return &response, nil
}

// GenerateSemanticHash generates a semantic hash for a comment
func (s *SemanticAnalyzer) GenerateSemanticHash(text string) (string, error) {
	prompt := fmt.Sprintf(`Extract the core semantic meaning of this review comment and generate a stable hash.
Focus on:
1. What specific changes are requested
2. Which files/components are affected
3. The priority or severity of the issue

Ignore:
- Formatting and whitespace
- Minor wording differences
- Examples that don't change the core request

Comment:
%s

Generate a semantic hash that captures the essence of what tasks should be created.
Respond with ONLY the hash string (20-30 characters, alphanumeric).`, text)

	claudeCmd, err := FindClaudeCommand(s.config.AISettings.ClaudePath)
	if err != nil {
		return "", fmt.Errorf("failed to find Claude: %w", err)
	}

	cmd := exec.Command(claudeCmd, prompt)

	// Set environment for Claude command
	cmd.Env = append(cmd.Environ(), "TERM=dumb", "NO_COLOR=1")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to run claude: %w", err)
	}

	hash := strings.TrimSpace(string(output))
	// Ensure hash is valid
	if len(hash) < 10 || len(hash) > 40 {
		// Fallback to text hash if AI response is invalid
		return storage.CalculateTextHash(text)[:20], nil
	}

	return hash, nil
}

// BatchAnalyzeChanges analyzes multiple comment changes for efficiency
func (s *SemanticAnalyzer) BatchAnalyzeChanges(changes []storage.CommentChange) (map[int64]bool, error) {
	semanticChanges := make(map[int64]bool)

	for _, change := range changes {
		if change.Type == "modified" {
			result, err := s.AnalyzeSemanticChange(change.PreviousText, change.CurrentText)
			if err != nil {
				// On error, assume semantic change to be safe
				fmt.Printf("⚠️  Failed to analyze semantic change for comment %d: %v\n", change.CommentID, err)
				semanticChanges[change.CommentID] = true
				continue
			}

			semanticChanges[change.CommentID] = result.HasSemanticChange
			
			if result.HasSemanticChange {
				fmt.Printf("🔄 Comment %d has semantic changes (%s): %s\n", 
					change.CommentID, result.ChangeType, result.Explanation)
			} else {
				fmt.Printf("✓ Comment %d has only cosmetic changes: %s\n", 
					change.CommentID, result.Explanation)
			}
		}
	}

	return semanticChanges, nil
}