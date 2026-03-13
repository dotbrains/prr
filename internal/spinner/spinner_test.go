package spinner

import (
	"bytes"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf, "loading")
	if s == nil {
		t.Fatal("expected non-nil spinner")
	}
	if s.prefix != "loading" {
		t.Errorf("expected prefix 'loading', got %q", s.prefix)
	}
	if s.stopped {
		t.Error("expected stopped to be false")
	}
}

func TestStartStop(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf, "→ Reviewing...")
	s.Start()
	// Let it tick at least once.
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected spinner to produce output")
	}
}

func TestDoubleStop(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf, "test")
	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()
	// Second stop should not panic or block.
	s.Stop()
}

func TestFormatDuration_Seconds(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{59 * time.Second, "59s"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{60 * time.Second, "1m00s"},
		{90 * time.Second, "1m30s"},
		{125 * time.Second, "2m05s"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
