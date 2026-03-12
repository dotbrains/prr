package spinner

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays an animated spinner with elapsed time.
type Spinner struct {
	w       io.Writer
	prefix  string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
	stopped bool
}

// New creates a spinner that writes to w with the given prefix.
// Example prefix: "→ Reviewing..."
func New(w io.Writer, prefix string) *Spinner {
	return &Spinner{
		w:      w,
		prefix: prefix,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Start begins the spinner animation in a background goroutine.
func (s *Spinner) Start() {
	go s.run()
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	close(s.stop)
	<-s.done
}

func (s *Spinner) run() {
	defer close(s.done)

	start := time.Now()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	frame := 0
	for {
		select {
		case <-s.stop:
			// Clear the spinner line
			fmt.Fprintf(s.w, "\r\033[K")
			return
		case <-tick.C:
			elapsed := time.Since(start).Truncate(time.Second)
			fmt.Fprintf(s.w, "\r%s %s (%s)", s.prefix, frames[frame%len(frames)], formatDuration(elapsed))
			frame++
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}
