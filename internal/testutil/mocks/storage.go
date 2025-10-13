package mocks

import "reviewtask/internal/storage"

// MockStorageManager implements a mock storage manager for testing
// This consolidates the duplicate implementations from test/integration_test.go
// and internal/ai/statistics_test.go to ensure consistency and reduce maintenance overhead.
type MockStorageManager struct {
	tasks          map[int][]storage.Task
	prBranches     map[string][]int
	currentBranch  string
	allPRNumbers   []int
	failedComments []storage.FailedComment
}

// NewMockStorageManager creates a new MockStorageManager with default values
func NewMockStorageManager() *MockStorageManager {
	return &MockStorageManager{
		tasks:          make(map[int][]storage.Task),
		prBranches:     make(map[string][]int),
		currentBranch:  "main",
		allPRNumbers:   []int{},
		failedComments: []storage.FailedComment{},
	}
}

// GetTasksByPR returns tasks for a specific PR number
func (m *MockStorageManager) GetTasksByPR(prNumber int) ([]storage.Task, error) {
	if tasks, exists := m.tasks[prNumber]; exists {
		return tasks, nil
	}
	return []storage.Task{}, nil
}

// GetCurrentBranch returns the current branch name
func (m *MockStorageManager) GetCurrentBranch() (string, error) {
	return m.currentBranch, nil
}

// GetPRsForBranch returns PR numbers for a specific branch
func (m *MockStorageManager) GetPRsForBranch(branchName string) ([]int, error) {
	if prs, exists := m.prBranches[branchName]; exists {
		return prs, nil
	}
	return []int{}, nil
}

// GetAllPRNumbers returns all PR numbers
// This implementation supports both modes: explicit allPRNumbers list and aggregation from prBranches
func (m *MockStorageManager) GetAllPRNumbers() ([]int, error) {
	// If allPRNumbers is explicitly set, use it
	if len(m.allPRNumbers) > 0 {
		return m.allPRNumbers, nil
	}

	// Otherwise, aggregate from prBranches (backward compatibility)
	prSet := make(map[int]bool)
	for _, prs := range m.prBranches {
		for _, pr := range prs {
			prSet[pr] = true
		}
	}

	var allPRs []int
	for pr := range prSet {
		allPRs = append(allPRs, pr)
	}
	return allPRs, nil
}

// Helper methods for setting up test data

// SetTasks sets tasks for a specific PR number
func (m *MockStorageManager) SetTasks(prNumber int, tasks []storage.Task) {
	m.tasks[prNumber] = tasks
}

// SetPRsForBranch sets PR numbers for a specific branch
func (m *MockStorageManager) SetPRsForBranch(branchName string, prNumbers []int) {
	m.prBranches[branchName] = prNumbers
}

// SetCurrentBranch sets the current branch name
func (m *MockStorageManager) SetCurrentBranch(branch string) {
	m.currentBranch = branch
}

// SetAllPRNumbers sets the explicit list of all PR numbers
func (m *MockStorageManager) SetAllPRNumbers(prNumbers []int) {
	m.allPRNumbers = prNumbers
}

// SaveFailedComment saves a failed comment
func (m *MockStorageManager) SaveFailedComment(comment storage.FailedComment) error {
	m.failedComments = append(m.failedComments, comment)
	return nil
}

// GetFailedComments returns all failed comments
func (m *MockStorageManager) GetFailedComments() []storage.FailedComment {
	return m.failedComments
}
