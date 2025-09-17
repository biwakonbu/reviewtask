package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWriteWorker_QueueAndWrite(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create manager and write worker
	manager := &Manager{
		baseDir: tmpDir,
	}

	worker := NewWriteWorker(manager, 10, false)

	// Start worker
	if err := worker.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer worker.Stop()

	// Create a test task
	task := Task{
		ID:              uuid.New().String(),
		Description:     "Test task",
		Priority:        "high",
		OriginText:      "Original comment",
		PRNumber:        123,
		SourceCommentID: 456,
		Status:          "todo",
		CreatedAt:       time.Now().Format(time.RFC3339),
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}

	// Queue the task
	if err := worker.QueueTask(task); err != nil {
		t.Fatalf("Failed to queue task: %v", err)
	}

	// Wait for completion
	worker.WaitForCompletion()

	// Verify task was written to file
	tasksFilePath := filepath.Join(tmpDir, "PR-123", "tasks.json")
	if _, err := os.Stat(tasksFilePath); os.IsNotExist(err) {
		t.Errorf("Tasks file was not created at %s", tasksFilePath)
	}

	// Load and verify the task
	tasksData, err := os.ReadFile(tasksFilePath)
	if err != nil {
		t.Fatalf("Failed to read tasks file: %v", err)
	}

	var tasksFile TasksFile
	if err := json.Unmarshal(tasksData, &tasksFile); err != nil {
		t.Fatalf("Failed to unmarshal tasks: %v", err)
	}
	loadedTasks := tasksFile.Tasks

	if len(loadedTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(loadedTasks))
	}

	if loadedTasks[0].ID != task.ID {
		t.Errorf("Task ID mismatch: expected %s, got %s", task.ID, loadedTasks[0].ID)
	}
}

func TestWriteWorker_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()

	manager := &Manager{
		baseDir: tmpDir,
	}

	worker := NewWriteWorker(manager, 100, false)

	if err := worker.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer worker.Stop()

	// Queue multiple tasks concurrently
	var wg sync.WaitGroup
	numTasks := 20
	prNumber := 456

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			task := Task{
				ID:              uuid.New().String(),
				Description:     "Concurrent test task",
				Priority:        "medium",
				PRNumber:        prNumber,
				SourceCommentID: int64(1000 + index),
				Status:          "todo",
				TaskIndex:       index,
				CreatedAt:       time.Now().Format(time.RFC3339),
				UpdatedAt:       time.Now().Format(time.RFC3339),
			}

			if err := worker.QueueTask(task); err != nil {
				t.Errorf("Failed to queue task %d: %v", index, err)
			}
		}(i)
	}

	wg.Wait()
	worker.WaitForCompletion()

	// Verify all tasks were written
	tasksFilePath := filepath.Join(tmpDir, "PR-456", "tasks.json")
	tasksData, err := os.ReadFile(tasksFilePath)
	if err != nil {
		t.Fatalf("Failed to read tasks file: %v", err)
	}

	var tasksFile TasksFile
	if err := json.Unmarshal(tasksData, &tasksFile); err != nil {
		t.Fatalf("Failed to unmarshal tasks: %v", err)
	}
	loadedTasks := tasksFile.Tasks

	if len(loadedTasks) != numTasks {
		t.Errorf("Expected %d tasks, got %d", numTasks, len(loadedTasks))
	}

	// Check for write errors
	errors := worker.GetErrors()
	if len(errors) > 0 {
		t.Errorf("Got %d write errors during concurrent writes", len(errors))
		for _, e := range errors {
			t.Logf("Write error: %v", e.Error)
		}
	}
}

func TestWriteWorker_PRSpecificDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	manager := &Manager{
		baseDir: tmpDir,
	}

	worker := NewWriteWorker(manager, 10, false)

	if err := worker.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer worker.Stop()

	// Create tasks for different PRs
	pr1Task := Task{
		ID:       uuid.New().String(),
		PRNumber: 111,
		Status:   "todo",
	}

	pr2Task := Task{
		ID:       uuid.New().String(),
		PRNumber: 222,
		Status:   "todo",
	}

	// Queue tasks
	if err := worker.QueueTask(pr1Task); err != nil {
		t.Fatalf("Failed to queue PR1 task: %v", err)
	}
	if err := worker.QueueTask(pr2Task); err != nil {
		t.Fatalf("Failed to queue PR2 task: %v", err)
	}

	worker.WaitForCompletion()

	// Verify tasks were written to correct PR directories
	pr1File := filepath.Join(tmpDir, "PR-111", "tasks.json")
	pr2File := filepath.Join(tmpDir, "PR-222", "tasks.json")

	if _, err := os.Stat(pr1File); os.IsNotExist(err) {
		t.Errorf("PR-111 tasks file was not created")
	}
	if _, err := os.Stat(pr2File); os.IsNotExist(err) {
		t.Errorf("PR-222 tasks file was not created")
	}

	// Verify each PR has only its own task
	pr1Data, _ := os.ReadFile(pr1File)
	var pr1TasksFile TasksFile
	json.Unmarshal(pr1Data, &pr1TasksFile)
	if len(pr1TasksFile.Tasks) != 1 || pr1TasksFile.Tasks[0].ID != pr1Task.ID {
		t.Errorf("PR-111 has incorrect tasks")
	}

	pr2Data, _ := os.ReadFile(pr2File)
	var pr2TasksFile TasksFile
	json.Unmarshal(pr2Data, &pr2TasksFile)
	if len(pr2TasksFile.Tasks) != 1 || pr2TasksFile.Tasks[0].ID != pr2Task.ID {
		t.Errorf("PR-222 has incorrect tasks")
	}
}

func TestWriteWorker_QueueFull(t *testing.T) {
	tmpDir := t.TempDir()

	manager := &Manager{
		baseDir: tmpDir,
	}

	// Create worker with very small queue
	worker := NewWriteWorker(manager, 1, false)

	if err := worker.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}
	defer worker.Stop()

	// Fill the queue
	task1 := Task{
		ID:       uuid.New().String(),
		PRNumber: 333,
		Status:   "todo",
	}

	// First task should succeed
	if err := worker.QueueTask(task1); err != nil {
		t.Errorf("First task should queue successfully: %v", err)
	}

	// Try to add another task immediately (queue should be full)
	task2 := Task{
		ID:       uuid.New().String(),
		PRNumber: 333,
		Status:   "todo",
	}

	// This might fail if queue is full
	err := worker.QueueTask(task2)
	if err != nil && err.Error() != "task queue is full" {
		t.Errorf("Expected 'task queue is full' error, got: %v", err)
	}

	// Wait for worker to process
	worker.WaitForCompletion()
}

func TestWriteWorker_StopWhileProcessing(t *testing.T) {
	tmpDir := t.TempDir()

	manager := &Manager{
		baseDir: tmpDir,
	}

	worker := NewWriteWorker(manager, 100, false)

	if err := worker.Start(); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Queue some tasks
	for i := 0; i < 5; i++ {
		task := Task{
			ID:       uuid.New().String(),
			PRNumber: 444,
			Status:   "todo",
		}
		worker.QueueTask(task)
	}

	// Stop the worker (should process remaining tasks)
	if err := worker.Stop(); err != nil {
		t.Errorf("Failed to stop worker: %v", err)
	}

	// Try to queue after stop (should fail)
	task := Task{
		ID:       uuid.New().String(),
		PRNumber: 444,
		Status:   "todo",
	}

	err := worker.QueueTask(task)
	if err == nil || err.Error() != "write worker is not running" {
		t.Errorf("Expected 'write worker is not running' error, got: %v", err)
	}
}