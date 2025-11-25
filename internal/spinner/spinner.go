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
	message   string
	frames    []string
	interval  time.Duration
	writer    io.Writer
	stopChan  chan struct{}
	stopped   bool
	mu        sync.Mutex
	hideWhenDone bool
}

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// New creates a new spinner with the given message
func New(message string) *Spinner {
	return &Spinner{
		message:   message,
		frames:    defaultFrames,
		interval:  80 * time.Millisecond,
		writer:    os.Stdout,
		stopChan:  make(chan struct{}),
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

		// Wait a tiny bit for the goroutine to finish
		time.Sleep(10 * time.Millisecond)

		// Clear the line
		fmt.Fprint(s.writer, "\r\033[K")
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

