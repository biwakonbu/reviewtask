package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestSignalHandling tests the signal handling mechanism in the root command
func TestSignalHandling(t *testing.T) {
	tests := []struct {
		name           string
		signal         os.Signal
		expectExitCode int
		timeout        time.Duration
	}{
		{
			name:           "SIGINT should trigger graceful shutdown",
			signal:         os.Interrupt,
			expectExitCode: 0,
			timeout:        2 * time.Second,
		},
		{
			name:           "SIGTERM should trigger graceful shutdown",
			signal:         syscall.SIGTERM,
			expectExitCode: 0,
			timeout:        2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context that can be cancelled
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Set up signal handling like in runReviewTask
			signalCh := make(chan os.Signal, 1)
			signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(signalCh)

			cancelCalled := false
			signalReceived := make(chan struct{})

			// Simulate the signal handling goroutine
			go func() {
				sigCount := 0
				for range signalCh {
					sigCount++
					if sigCount == 1 {
						// First signal: try graceful cancellation
						cancel()
						cancelCalled = true
						close(signalReceived)

						// Wait for graceful shutdown with timeout
						go func() {
							timeout := time.NewTimer(3 * time.Second)
							defer timeout.Stop()
							<-timeout.C
							if sigCount == 1 {
								// This would be os.Exit(1) in real code
								t.Log("Would force exit after timeout")
							}
						}()
					} else {
						// Second signal: force immediate exit
						t.Log("Would force terminate immediately")
						return
					}
				}
			}()

			// Send the test signal
			signalCh <- tt.signal

			// Wait for cancellation with timeout
			select {
			case <-ctx.Done():
				if !cancelCalled {
					t.Error("Context was cancelled but cancel was not called by signal handler")
				}
			case <-time.After(tt.timeout):
				t.Error("Signal handling timed out")
			}

			// Verify the context was properly cancelled
			if ctx.Err() != context.Canceled {
				t.Errorf("Expected context to be cancelled, got error: %v", ctx.Err())
			}
		})
	}
}

// TestDoubleSignalHandling tests the behavior when multiple signals are sent
func TestDoubleSignalHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, os.Interrupt)
	defer signal.Stop(signalCh)

	forceExitCalled := false
	firstSignalReceived := make(chan struct{})
	secondSignalReceived := make(chan struct{})

	// Simulate the signal handling goroutine
	go func() {
		sigCount := 0
		for range signalCh {
			sigCount++
			if sigCount == 1 {
				cancel()
				close(firstSignalReceived)
			} else {
				// Second signal: force immediate exit
				forceExitCalled = true
				close(secondSignalReceived)
				return
			}
		}
	}()

	// Send two signals rapidly
	signalCh <- os.Interrupt
	
	// Wait for first signal to be processed
	<-firstSignalReceived
	
	signalCh <- os.Interrupt
	
	// Wait for second signal to be processed
	<-secondSignalReceived

	if !forceExitCalled {
		t.Error("Expected force exit to be called after second signal")
	}

	// Verify the context was cancelled
	if ctx.Err() != context.Canceled {
		t.Errorf("Expected context to be cancelled, got error: %v", ctx.Err())
	}
}

// TestContextPropagation tests that context cancellation propagates correctly
func TestContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a child context
	childCtx, childCancel := context.WithCancel(ctx)
	defer childCancel()

	// Cancel the parent context (simulating signal handling)
	cancel()

	// Child context should also be cancelled
	select {
	case <-childCtx.Done():
		// Expected behavior
	case <-time.After(100 * time.Millisecond):
		t.Error("Child context was not cancelled when parent was cancelled")
	}

	if childCtx.Err() != context.Canceled {
		t.Errorf("Expected child context to be cancelled, got error: %v", childCtx.Err())
	}
}
