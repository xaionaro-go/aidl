package proxy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/electricbubble/gadb"
	"github.com/facebookincubator/go-belt/tool/logger"

	"github.com/AndroidGoLab/binder/interop/gadb/runner"
)

const (
	// defaultDaemonPort is the TCP port the daemon listens on.
	defaultDaemonPort = 7100

	// remoteDaemonPath is where the daemon binary is pushed on the device.
	remoteDaemonPath = "/data/local/tmp/binder-proxyd"

	// daemonStartupDelay gives the daemon time to open the binder driver
	// and start listening before the host connects.
	daemonStartupDelay = 2 * time.Second
)

// Session orchestrates the full remote-binder-proxy flow:
// cross-compile and push the daemon, start it, set up port forwarding,
// and create a RemoteTransport.
type Session struct {
	runner    *runner.DeviceRunner
	device    gadb.Device
	transport *RemoteTransport
	localPort int
}

// SessionOption configures a Session.
type SessionOption interface {
	applySession(*sessionConfig)
}

// SessionOptions is a slice of SessionOption.
type SessionOptions []SessionOption

func (opts SessionOptions) config() sessionConfig {
	cfg := defaultSessionConfig()
	for _, o := range opts {
		o.applySession(&cfg)
	}
	return cfg
}

type sessionConfig struct {
	LocalPort   int
	RemotePort  int
	DaemonBin   string // path to pre-built daemon binary; empty means cross-compile
}

func defaultSessionConfig() sessionConfig {
	return sessionConfig{
		LocalPort:  defaultDaemonPort,
		RemotePort: defaultDaemonPort,
	}
}

type sessionOptionLocalPort int

func (o sessionOptionLocalPort) applySession(c *sessionConfig) { c.LocalPort = int(o) }

// SessionOptionLocalPort sets the local TCP port for ADB forwarding.
func SessionOptionLocalPort(port int) SessionOption { return sessionOptionLocalPort(port) }

type sessionOptionRemotePort int

func (o sessionOptionRemotePort) applySession(c *sessionConfig) { c.RemotePort = int(o) }

// SessionOptionRemotePort sets the remote TCP port the daemon listens on.
func SessionOptionRemotePort(port int) SessionOption { return sessionOptionRemotePort(port) }

type sessionOptionDaemonBin string

func (o sessionOptionDaemonBin) applySession(c *sessionConfig) { c.DaemonBin = string(o) }

// SessionOptionDaemonBin sets the path to a pre-built daemon binary,
// skipping cross-compilation.
func SessionOptionDaemonBin(path string) SessionOption { return sessionOptionDaemonBin(path) }

// NewSession creates a Session: pushes the daemon to the device, starts it,
// sets up port forwarding, and connects.
func NewSession(
	ctx context.Context,
	deviceSerial string,
	opts ...SessionOption,
) (_ *Session, _err error) {
	logger.Tracef(ctx, "NewSession(%s)", deviceSerial)
	defer func() { logger.Tracef(ctx, "/NewSession: %v", _err) }()

	cfg := SessionOptions(opts).config()

	dr, err := runner.NewDeviceRunner(deviceSerial)
	if err != nil {
		return nil, fmt.Errorf("creating device runner: %w", err)
	}

	device, err := findDevice(deviceSerial)
	if err != nil {
		return nil, fmt.Errorf("finding device: %w", err)
	}

	s := &Session{
		runner:    dr,
		device:    device,
		localPort: cfg.LocalPort,
	}
	defer func() {
		if _err != nil {
			s.cleanup(ctx)
		}
	}()

	// Build or use pre-built daemon binary.
	daemonBin := cfg.DaemonBin
	if daemonBin == "" {
		var err error
		daemonBin, err = crossCompileDaemon(ctx)
		if err != nil {
			return nil, fmt.Errorf("cross-compiling daemon: %w", err)
		}
		// Remove both the binary and its parent temp directory.
		defer os.RemoveAll(filepath.Dir(daemonBin))
	}

	// Push daemon to device.
	if err := dr.PushBinary(ctx, daemonBin, remoteDaemonPath); err != nil {
		return nil, fmt.Errorf("pushing daemon: %w", err)
	}

	// Start daemon in background via a helper script. The script traps
	// SIGHUP so the daemon survives the gadb shell session closing
	// (Android's nohup is unreliable for this). Using a script avoids
	// shell syntax issues from DeviceRunner.Run appending "; echo $?"
	// after a trailing "&".
	listenAddr := fmt.Sprintf(":%d", cfg.RemotePort)
	scriptPath := remoteDaemonPath + ".sh"

	scriptContent := fmt.Sprintf("#!/system/bin/sh\ntrap '' HUP\n%s -listen %s </dev/null >/dev/null 2>&1 &\n",
		remoteDaemonPath, listenAddr,
	)
	localScript, err := os.CreateTemp("", "binder-proxyd-start-*.sh")
	if err != nil {
		return nil, fmt.Errorf("creating start script: %w", err)
	}
	if _, err := localScript.WriteString(scriptContent); err != nil {
		localScript.Close()
		os.Remove(localScript.Name())
		return nil, fmt.Errorf("writing start script: %w", err)
	}
	localScript.Close()
	defer os.Remove(localScript.Name())

	if err := dr.PushBinary(ctx, localScript.Name(), scriptPath); err != nil {
		return nil, fmt.Errorf("pushing start script: %w", err)
	}

	if _, err := dr.Run(ctx, scriptPath, 5*time.Second); err != nil {
		return nil, fmt.Errorf("starting daemon: %w", err)
	}

	logger.Debugf(ctx, "waiting %s for daemon startup", daemonStartupDelay)
	time.Sleep(daemonStartupDelay)

	// Set up ADB port forwarding.
	if err := device.Forward(cfg.LocalPort, cfg.RemotePort); err != nil {
		return nil, fmt.Errorf("port forwarding tcp:%d -> tcp:%d: %w", cfg.LocalPort, cfg.RemotePort, err)
	}

	// Connect to daemon.
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.LocalPort)
	transport, err := NewRemoteTransport(addr)
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon: %w", err)
	}
	s.transport = transport

	return s, nil
}

// Transport returns the RemoteTransport connected to the device daemon.
func (s *Session) Transport() *RemoteTransport {
	return s.transport
}

// Close kills the daemon, removes the binary, tears down port forwarding,
// and closes the transport.
func (s *Session) Close(
	ctx context.Context,
) error {
	logger.Tracef(ctx, "Session.Close")
	defer func() { logger.Tracef(ctx, "/Session.Close") }()

	var errs []error

	if s.transport != nil {
		if err := s.transport.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing transport: %w", err))
		}
	}

	s.cleanup(ctx)

	return errors.Join(errs...)
}

// cleanup kills the daemon process, removes the binary, and tears down
// port forwarding. Safe to call multiple times.
func (s *Session) cleanup(ctx context.Context) {
	// Kill daemon process.
	if _, err := s.runner.Run(ctx, "pkill -f binder-proxyd", 5*time.Second); err != nil {
		logger.Debugf(ctx, "killing daemon: %v", err)
	}

	// Remove daemon binary and start script.
	if err := s.runner.Cleanup(ctx, remoteDaemonPath); err != nil {
		logger.Debugf(ctx, "removing daemon binary: %v", err)
	}
	if err := s.runner.Cleanup(ctx, remoteDaemonPath+".sh"); err != nil {
		logger.Debugf(ctx, "removing daemon start script: %v", err)
	}

	// Remove port forwarding.
	if err := s.device.ForwardKill(s.localPort); err != nil {
		logger.Debugf(ctx, "removing port forward: %v", err)
	}
}

// findDevice connects to the ADB server and finds the device by serial.
func findDevice(serial string) (gadb.Device, error) {
	client, err := gadb.NewClient()
	if err != nil {
		return gadb.Device{}, fmt.Errorf("connecting to ADB server: %w", err)
	}

	devices, err := client.DeviceList()
	if err != nil {
		return gadb.Device{}, fmt.Errorf("listing devices: %w", err)
	}

	for _, dev := range devices {
		if dev.Serial() == serial {
			return dev, nil
		}
	}

	return gadb.Device{}, fmt.Errorf("device %q not found among %d connected device(s)", serial, len(devices))
}

// crossCompileDaemon builds the daemon binary for arm64 in a temp directory.
func crossCompileDaemon(ctx context.Context) (_ string, _err error) {
	logger.Tracef(ctx, "crossCompileDaemon")
	defer func() { logger.Tracef(ctx, "/crossCompileDaemon: %v", _err) }()

	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", fmt.Errorf("finding repo root: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "binder-proxyd-build-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	outPath := filepath.Join(tmpDir, "binder-proxyd")

	cmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "./cmd/binder-proxyd/")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("go build: %w\n%s", err, output)
	}

	logger.Debugf(ctx, "built daemon binary: %s", outPath)
	return outPath, nil
}

// findRepoRoot walks up from the current directory to locate go.mod.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fall back to git.
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("cannot find repo root: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
