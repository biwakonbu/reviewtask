package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WriteWorker manages concurrent writes to the tasks file
type WriteWorker struct {
	manager      *Manager
	taskQueue    chan Task
	errorQueue   chan WriteError
	wg           sync.WaitGroup
	mu           sync.Mutex
	isRunning    bool
	verboseMode  bool
	shutdownChan chan struct{}
}

// WriteError represents a failed write attempt
type WriteError struct {
	Task      Task
	Error     error
	Timestamp time.Time
}

// NewWriteWorker creates a new write worker instance
func NewWriteWorker(manager *Manager, queueSize int, verbose bool) *WriteWorker {
	return &WriteWorker{
		manager:      manager,
		taskQueue:    make(chan Task, queueSize),
		errorQueue:   make(chan WriteError, queueSize),
		verboseMode:  verbose,
		shutdownChan: make(chan struct{}),
	}
}

// Start begins the write worker goroutine
func (w *WriteWorker) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return fmt.Errorf("write worker is already running")
	}

	w.isRunning = true
	w.wg.Add(1)

	go w.processQueue()

	if w.verboseMode {
		fmt.Println("✅ Write worker started")
	}

	return nil
}

// Stop gracefully shuts down the write worker
func (w *WriteWorker) Stop() error {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return fmt.Errorf("write worker is not running")
	}
	w.isRunning = false
	w.mu.Unlock()

	// Signal shutdown
	close(w.shutdownChan)

	// Close the queue to prevent new tasks
	close(w.taskQueue)

	// Wait for worker to finish
	w.wg.Wait()

	if w.verboseMode {
		fmt.Println("✅ Write worker stopped")
	}

	return nil
}

// QueueTask adds a task to the write queue
func (w *WriteWorker) QueueTask(task Task) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return fmt.Errorf("write worker is not running")
	}

	select {
	case w.taskQueue <- task:
		if w.verboseMode {
			fmt.Printf("📝 Queued task %s for writing\n", task.ID)
		}
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// QueueTasks adds multiple tasks to the write queue
func (w *WriteWorker) QueueTasks(tasks []Task) error {
	for _, task := range tasks {
		if err := w.QueueTask(task); err != nil {
			return fmt.Errorf("failed to queue task %s: %w", task.ID, err)
		}
	}
	return nil
}

// GetErrors returns failed write attempts
func (w *WriteWorker) GetErrors() []WriteError {
	var errors []WriteError

	// Non-blocking read of all available errors
	for {
		select {
		case err := <-w.errorQueue:
			errors = append(errors, err)
		default:
			return errors
		}
	}
}

// WaitForCompletion waits for all queued tasks to be processed
func (w *WriteWorker) WaitForCompletion() {
	// Wait for queue to be empty
	for len(w.taskQueue) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Small additional delay to ensure last write completes
	time.Sleep(50 * time.Millisecond)
}

// processQueue is the main worker loop
func (w *WriteWorker) processQueue() {
	defer w.wg.Done()

	for {
		select {
		case task, ok := <-w.taskQueue:
			if !ok {
				// Queue closed, exit
				return
			}

			// Write the task
			if err := w.writeTask(task); err != nil {
				// Record the error
				writeErr := WriteError{
					Task:      task,
					Error:     err,
					Timestamp: time.Now(),
				}

				select {
				case w.errorQueue <- writeErr:
					if w.verboseMode {
						fmt.Printf("❌ Failed to write task %s: %v\n", task.ID, err)
					}
				default:
					// Error queue is full, log but continue
					if w.verboseMode {
						fmt.Printf("⚠️  Error queue full, dropping error for task %s\n", task.ID)
					}
				}
			} else {
				if w.verboseMode {
					fmt.Printf("✅ Successfully wrote task %s\n", task.ID)
				}
			}

		case <-w.shutdownChan:
			// Drain remaining tasks before shutdown
			for task := range w.taskQueue {
				_ = w.writeTask(task)
			}
			return
		}
	}
}

// writeTask performs the actual file write operation
func (w *WriteWorker) writeTask(task Task) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Load existing tasks from PR-specific directory
	prDir := filepath.Join(w.manager.baseDir, fmt.Sprintf("PR-%d", task.PRNumber))
	tasksFile := filepath.Join(prDir, "tasks.json")

	// Ensure PR directory exists
	if err := os.MkdirAll(prDir, 0755); err != nil {
		return fmt.Errorf("failed to create PR directory: %w", err)
	}

	var existingTasks TasksFile
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read tasks file: %w", err)
		}
		// File doesn't exist, create new
		existingTasks = TasksFile{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Tasks:       []Task{},
		}
	} else {
		if err := json.Unmarshal(data, &existingTasks); err != nil {
			return fmt.Errorf("failed to unmarshal existing tasks: %w", err)
		}
	}

	// Check if task already exists (by ID)
	taskExists := false
	for i, existing := range existingTasks.Tasks {
		if existing.ID == task.ID {
			// Update existing task
			existingTasks.Tasks[i] = task
			taskExists = true
			break
		}
	}

	// Add new task if it doesn't exist
	if !taskExists {
		existingTasks.Tasks = append(existingTasks.Tasks, task)
	}

	// Update timestamp
	existingTasks.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

	// Marshal back to JSON
	updatedData, err := json.MarshalIndent(existingTasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Write to file
	if err := os.WriteFile(tasksFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	return nil
}

// Stats returns current queue statistics
func (w *WriteWorker) Stats() (queueSize int, errorCount int, isRunning bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return len(w.taskQueue), len(w.errorQueue), w.isRunning
}
