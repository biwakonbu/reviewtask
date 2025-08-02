package notification

// ExclusionReason represents why a review comment was not converted to a task
type ExclusionReason struct {
	Type        string   // "policy", "duplicate", "out_of_scope", "deprecated", "already_implemented"
	References  []string // Related documents, PRs, or Issue numbers
	Explanation string   // Detailed explanation
	Confidence  float64  // AI confidence score (0.0-1.0)
}

// Common exclusion types
const (
	ExclusionTypePolicy             = "Project Policy Violation"
	ExclusionTypeDuplicate          = "Duplicate Suggestion"
	ExclusionTypeOutOfScope         = "Out of Scope"
	ExclusionTypeDeprecated         = "Deprecated Approach"
	ExclusionTypeAlreadyImplemented = "Already Implemented"
	ExclusionTypeLowPriority        = "Low Priority"
	ExclusionTypeInvalid            = "Invalid Suggestion"
)