package runner

import "time"

// RunResult holds the output and metadata of a command execution on a device.
type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}
