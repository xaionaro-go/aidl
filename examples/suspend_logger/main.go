// Acquire and release a wake lock via the PowerManager binder service.
//
// Uses the framework PowerManager service (accessible from shell context)
// rather than the SystemSuspend HAL (which requires privileged SELinux context).
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/suspend_logger ./examples/suspend_logger/
//	adb push build/suspend_logger /data/local/tmp/ && adb shell /data/local/tmp/suspend_logger
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	partialWakeLock = 1 // PowerManager.PARTIAL_WAKE_LOCK
	packageName     = "com.android.shell"
)

func main() {
	ctx := context.Background()

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer drv.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)

	power, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get PowerManager: %v\n", err)
		os.Exit(1)
	}

	// Create a binder token for wake lock ownership tracking.
	lockToken := binder.NewStubBinder(&wakeLockToken{})
	lockToken.RegisterWithTransport(ctx, transport)

	wlCallback := genOs.NewWakeLockCallbackStub(&noopWakeLockCallback{})

	fmt.Println("Acquiring partial wake lock...")
	err = power.AcquireWakeLock(
		ctx,
		lockToken,
		partialWakeLock,
		"suspend_logger_example",
		packageName,
		genOs.WorkSource{},
		"",  // historyTag
		0,   // displayId
		wlCallback,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AcquireWakeLock: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wake lock acquired. Device will not suspend.")

	fmt.Println("Holding for 3 seconds...")
	time.Sleep(3 * time.Second)

	fmt.Println("Releasing wake lock...")
	if err := power.ReleaseWakeLock(ctx, lockToken, 0); err != nil {
		fmt.Fprintf(os.Stderr, "ReleaseWakeLock: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wake lock released. Device may suspend normally.")
}

// wakeLockToken is a minimal TransactionReceiver used as the binder token
// for PowerManager wake lock acquire/release.
type wakeLockToken struct{}

func (w *wakeLockToken) Descriptor() string { return "wakelock.token" }

func (w *wakeLockToken) OnTransaction(
	_ context.Context,
	_ binder.TransactionCode,
	_ *parcel.Parcel,
) (*parcel.Parcel, error) {
	return parcel.New(), nil
}

// noopWakeLockCallback implements IWakeLockCallbackServer.
type noopWakeLockCallback struct{}

func (n *noopWakeLockCallback) OnStateChanged(_ context.Context, _ bool) error {
	return nil
}

var _ genOs.IWakeLockCallbackServer = (*noopWakeLockCallback)(nil)
