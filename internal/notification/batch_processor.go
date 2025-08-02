package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"reviewtask/internal/config"
)

// BatchProcessor handles periodic batch processing of queued comments
type BatchProcessor struct {
	notifier *Notifier
	config   *config.Config
	ticker   *time.Ticker
	done     chan bool
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(notifier *Notifier, cfg *config.Config) *BatchProcessor {
	return &BatchProcessor{
		notifier: notifier,
		config:   cfg,
		done:     make(chan bool),
	}
}

// Start begins the batch processing loop
func (bp *BatchProcessor) Start() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.running {
		return fmt.Errorf("batch processor already running")
	}

	// Calculate interval from config
	interval := time.Duration(bp.config.CommentSettings.Throttling.BatchWindowMinutes) * time.Minute
	if interval < time.Minute {
		interval = time.Minute // Minimum 1 minute
	}

	bp.ticker = time.NewTicker(interval)
	bp.running = true

	bp.wg.Add(1)
	go bp.processLoop()

	return nil
}

// Stop halts the batch processor
func (bp *BatchProcessor) Stop() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if !bp.running {
		return
	}

	bp.running = false
	close(bp.done)
	if bp.ticker != nil {
		bp.ticker.Stop()
	}

	bp.wg.Wait()
}

// processLoop is the main processing loop
func (bp *BatchProcessor) processLoop() {
	defer bp.wg.Done()

	for {
		select {
		case <-bp.ticker.C:
			bp.processBatches()
		case <-bp.done:
			return
		}
	}
}

// processBatches processes all pending batches
func (bp *BatchProcessor) processBatches() {
	ctx := context.Background()

	// First optimize batches with AI if available
	if bp.config.AISettings.VerboseMode {
		err := bp.notifier.throttler.OptimizeBatches(ctx)
		if err != nil {
			// Log error but continue processing
			fmt.Printf("⚠️  Batch optimization error: %v\n", err)
		}
	}

	// Process batched comments
	err := bp.notifier.ProcessBatchedComments(ctx)
	if err != nil {
		fmt.Printf("⚠️  Batch processing error: %v\n", err)
	}
}

// ProcessNow triggers immediate batch processing
func (bp *BatchProcessor) ProcessNow() error {
	bp.mu.Lock()
	if !bp.running {
		bp.mu.Unlock()
		return fmt.Errorf("batch processor not running")
	}
	bp.mu.Unlock()

	bp.processBatches()
	return nil
}

// GetStatus returns the current status of the batch processor
func (bp *BatchProcessor) GetStatus() (bool, time.Duration) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if !bp.running || bp.ticker == nil {
		return false, 0
	}

	// Calculate time until next batch
	// This is approximate since we don't track the exact last tick time
	interval := time.Duration(bp.config.CommentSettings.Throttling.BatchWindowMinutes) * time.Minute
	
	return true, interval
}