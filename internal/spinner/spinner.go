package spinner

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Enabled controls whether spinners are displayed (disabled in verbose mode)
var Enabled = true

// Spinner represents a loading spinner
type Spinner struct {
	message      string
	frames       []string
	interval     time.Duration
	writer       io.Writer
	stopChan     chan struct{}
	stopped      bool
	mu           sync.Mutex
	hideWhenDone bool
	wg           sync.WaitGroup
}

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// New creates a new spinner with the given message
func New(message string) *Spinner {
	return &Spinner{
		message:      message,
		frames:       defaultFrames,
		interval:     80 * time.Millisecond,
		writer:       os.Stdout,
		stopChan:     make(chan struct{}),
		hideWhenDone: false,
	}
}

// HideWhenDone sets whether to hide the spinner line when done
func (s *Spinner) HideWhenDone() *Spinner {
	s.hideWhenDone = true
	return s
}

// Start starts the spinner
func (s *Spinner) Start() *Spinner {
	if Enabled {
		s.wg.Add(1)
		go s.run()
	}
	return s
}

// Stop stops the spinner and optionally shows a final message
func (s *Spinner) Stop(finalMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.stopped = true

	if Enabled {
		close(s.stopChan)
		s.mu.Unlock() // Unlock so the goroutine can finish

		// Wait for the goroutine to fully finish
		s.wg.Wait()

		s.mu.Lock() // Re-lock for the rest of the function

		// Clear the line and move cursor to beginning
		fmt.Fprint(s.writer, "\r\033[K")

		// Flush to ensure clearing is processed immediately
		if f, ok := s.writer.(interface{ Sync() error }); ok {
			_ = f.Sync()
		}
	}

	// Print final message if not hiding
	if !s.hideWhenDone && finalMessage != "" {
		fmt.Fprintln(s.writer, finalMessage)
	}
}

// UpdateMessage updates the spinner message while it's running
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

func (s *Spinner) run() {
	defer s.wg.Done()
	frameIdx := 0
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.Lock()
			frame := s.frames[frameIdx%len(s.frames)]
			fmt.Fprintf(s.writer, "\r%s %s", frame, s.message)
			s.mu.Unlock()
			frameIdx++
		}
	}
}

// Wrap runs a function with a spinner
func Wrap(message string, fn func() error) error {
	if !Enabled {
		// When disabled (verbose mode), just run the function
		return fn()
	}
	sp := New(message).HideWhenDone().Start()
	err := fn()
	sp.Stop("")
	return err
}

// WrapWithSuccess runs a function with a spinner and shows success/error message
func WrapWithSuccess(message, successMessage string, fn func() error) error {
	if !Enabled {
		// When disabled (verbose mode), print message and run
		fmt.Println(message)
		err := fn()
		if err != nil {
			fmt.Printf("✗ Error: %v\n", err)
		}
		return err
	}
	sp := New(message).Start()
	err := fn()
	if err != nil {
		sp.Stop(fmt.Sprintf("✗ %s: %v", message, err))
		return err
	}
	sp.Stop(fmt.Sprintf("✓ %s", successMessage))
	return nil
}

// ProgressFunc is a function that can be called to update spinner progress
type ProgressFunc func(message string)

// WrapWithAutoDelay runs a function and only shows a spinner if it takes longer than the delay
func WrapWithAutoDelay(message string, delay time.Duration, fn func() error) error {
	if !Enabled {
		// When disabled (verbose mode), just run the function
		return fn()
	}

	// Create a timer and a flag to track if we should show the spinner
	var sp *Spinner
	var mu sync.Mutex
	spinnerStarted := false

	// Start a goroutine that will show the spinner after the delay
	timer := time.AfterFunc(delay, func() {
		mu.Lock()
		defer mu.Unlock()
		sp = New(message).HideWhenDone().Start()
		spinnerStarted = true
	})

	// Run the function (synchronously)
	err := fn()

	// Stop the timer to prevent spinner from starting if function completed quickly
	timer.Stop()

	// Small delay to ensure timer goroutine has finished
	time.Sleep(15 * time.Millisecond)

	// Now safely stop the spinner if it was started
	mu.Lock()
	if spinnerStarted && sp != nil {
		sp.Stop("")
	}
	mu.Unlock()

	return err
}

// WrapWithAutoDelayAndProgress runs a function with auto-delay spinner that supports progress updates
func WrapWithAutoDelayAndProgress(message string, delay time.Duration, fn func(progress ProgressFunc) error) error {
	if !Enabled {
		// When disabled (verbose mode), just run the function with a no-op progress callback
		return fn(func(msg string) {})
	}

	// Create a timer and a flag to track if we should show the spinner
	var sp *Spinner
	var mu sync.Mutex
	spinnerStarted := false

	// Start a goroutine that will show the spinner after the delay
	timer := time.AfterFunc(delay, func() {
		mu.Lock()
		defer mu.Unlock()
		sp = New(message).HideWhenDone().Start()
		spinnerStarted = true
	})

	// Create progress callback that updates spinner if it's running
	progress := func(msg string) {
		mu.Lock()
		defer mu.Unlock()
		if spinnerStarted && sp != nil {
			sp.UpdateMessage(msg)
		}
		// If spinner hasn't started yet, progress updates are no-ops
	}

	// Run the function (synchronously) with progress callback
	err := fn(progress)

	// Stop the timer to prevent spinner from starting if function completed quickly
	timer.Stop()

	// Small delay to ensure timer goroutine has finished
	time.Sleep(15 * time.Millisecond)

	// Now safely stop the spinner if it was started
	mu.Lock()
	if spinnerStarted && sp != nil {
		sp.Stop("")
	}
	mu.Unlock()

	return err
}
