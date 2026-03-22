package runner

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/electricbubble/gadb"
	"github.com/facebookincubator/go-belt/tool/logger"
)

// DeviceRunner executes commands and transfers files on a single Android device
// via the gadb (pure-Go ADB) client library.
type DeviceRunner struct {
	device gadb.Device
	serial string
}

// NewDeviceRunner connects to the ADB server and selects the device
// identified by serial.
func NewDeviceRunner(
	serial string,
) (*DeviceRunner, error) {
	client, err := gadb.NewClient()
	if err != nil {
		return nil, fmt.Errorf("connecting to ADB server: %w", err)
	}

	devices, err := client.DeviceList()
	if err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}

	for _, dev := range devices {
		if dev.Serial() == serial {
			return &DeviceRunner{
				device: dev,
				serial: serial,
			}, nil
		}
	}

	return nil, fmt.Errorf("device %q not found among %d connected device(s)", serial, len(devices))
}

// PushBinary transfers a local file to remotePath on the device and marks it
// executable (0755).
func (r *DeviceRunner) PushBinary(
	ctx context.Context,
	localPath string,
	remotePath string,
) (_err error) {
	logger.Tracef(ctx, "PushBinary")
	defer func() { logger.Tracef(ctx, "/PushBinary: %v", _err) }()

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening local file %q: %w", localPath, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat local file %q: %w", localPath, err)
	}

	logger.Debugf(ctx, "pushing %q (%d bytes) to %s:%s", localPath, stat.Size(), r.serial, remotePath)

	if err := r.device.Push(f, remotePath, stat.ModTime(), os.FileMode(0755)); err != nil {
		return fmt.Errorf("pushing to device: %w", err)
	}

	return nil
}

// Run executes a shell command on the device. It captures combined
// stdout/stderr and parses the exit code by appending "; echo $?" to the
// command. If timeout is positive, the command is wrapped with the shell
// timeout utility.
func (r *DeviceRunner) Run(
	ctx context.Context,
	command string,
	timeout time.Duration,
) (_ *RunResult, _err error) {
	logger.Tracef(ctx, "Run")
	defer func() { logger.Tracef(ctx, "/Run: %v", _err) }()

	shellCmd := command
	if timeout > 0 {
		seconds := int(timeout.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		shellCmd = fmt.Sprintf("timeout %d %s", seconds, shellCmd)
	}

	// Append exit-code probe so we can distinguish success from failure.
	shellCmd = fmt.Sprintf("%s; echo \"\\n$?\"", shellCmd)

	logger.Debugf(ctx, "running on %s: %s", r.serial, shellCmd)

	start := time.Now()
	output, err := r.device.RunShellCommand(shellCmd)
	elapsed := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("shell command failed: %w", err)
	}

	stdout, exitCode := parseExitCode(output)

	return &RunResult{
		Stdout:   stdout,
		ExitCode: exitCode,
		Duration: elapsed,
	}, nil
}

// Cleanup removes a file from the device.
func (r *DeviceRunner) Cleanup(
	ctx context.Context,
	remotePath string,
) (_err error) {
	logger.Tracef(ctx, "Cleanup")
	defer func() { logger.Tracef(ctx, "/Cleanup: %v", _err) }()

	_, err := r.device.RunShellCommand("rm", "-f", remotePath)
	if err != nil {
		return fmt.Errorf("removing %q: %w", remotePath, err)
	}

	return nil
}

// parseExitCode splits the combined output produced by "; echo $?" into the
// actual command output and an integer exit code.
func parseExitCode(raw string) (stdout string, exitCode int) {
	raw = strings.TrimRight(raw, "\r\n")
	idx := strings.LastIndex(raw, "\n")
	if idx < 0 {
		// The entire output is just the exit code (no preceding output).
		code, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return raw, -1
		}
		return "", code
	}

	codePart := strings.TrimSpace(raw[idx+1:])
	code, err := strconv.Atoi(codePart)
	if err != nil {
		// Could not parse exit code; return the full output.
		return raw, -1
	}

	return strings.TrimRight(raw[:idx], "\r\n"), code
}
