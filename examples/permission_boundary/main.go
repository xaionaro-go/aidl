// Test which binder calls succeed or fail from the current security context.
//
// Attempts various operations across services to map the permission
// boundary of the calling process (typically shell/root).
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/permission_boundary ./examples/permission_boundary/
//	adb push build/permission_boundary /data/local/tmp/ && adb shell /data/local/tmp/permission_boundary
package main

import (
	"context"
	"fmt"
	"os"

	genAdmin "github.com/AndroidGoLab/binder/android/app/admin"
	"github.com/AndroidGoLab/binder/android/content"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

type permTest struct {
	name string
	fn   func(ctx context.Context) error
}

func main() {
	ctx := context.Background()

	driver, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, driver, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)
	callerPkg := binder.DefaultCallerIdentity().PackageName

	// Build test list.
	var tests []permTest

	// PowerManager tests (read-only, usually succeed from shell).
	power, powerErr := genOs.GetPowerManager(ctx, sm)
	if powerErr == nil {
		tests = append(tests,
			permTest{"power.IsInteractive", func(ctx context.Context) error {
				_, err := power.IsInteractive(ctx)
				return err
			}},
			permTest{"power.IsPowerSaveMode", func(ctx context.Context) error {
				_, err := power.IsPowerSaveMode(ctx)
				return err
			}},
			permTest{"power.SetPowerSaveModeEnabled", func(ctx context.Context) error {
				// Attempt to toggle power save (write operation, may be denied).
				_, err := power.SetPowerSaveModeEnabled(ctx, false)
				return err
			}},
		)
	}

	// DevicePolicyManager tests.
	dpm, dpmErr := genAdmin.GetDevicePolicyManager(ctx, sm)
	if dpmErr == nil {
		tests = append(tests,
			permTest{"dpm.IsDeviceProvisioned", func(ctx context.Context) error {
				_, err := dpm.IsDeviceProvisioned(ctx)
				return err
			}},
			permTest{"dpm.GetStorageEncryptionStatus", func(ctx context.Context) error {
				_, err := dpm.GetStorageEncryptionStatus(ctx, callerPkg)
				return err
			}},
			permTest{"dpm.GetActiveAdmins", func(ctx context.Context) error {
				_, err := dpm.GetActiveAdmins(ctx)
				return err
			}},
			permTest{"dpm.GetCameraDisabled", func(ctx context.Context) error {
				_, err := dpm.GetCameraDisabled(ctx, content.ComponentName{}, callerPkg, false)
				return err
			}},
		)
	}

	// SecurityStateManager tests.
	sec, secErr := genOs.GetSecurityStateManager(ctx, sm)
	if secErr == nil {
		tests = append(tests,
			permTest{"security.GetGlobalSecurityState", func(ctx context.Context) error {
				_, err := sec.GetGlobalSecurityState(ctx)
				return err
			}},
		)
	}

	fmt.Println("=== Permission Boundary Test ===")
	fmt.Printf("PID: %d, UID: %d\n\n", os.Getpid(), os.Getuid())
	fmt.Printf("%-45s %s\n", "OPERATION", "RESULT")
	fmt.Println("--------------------------------------------------------------")

	passed := 0
	denied := 0
	for _, t := range tests {
		err := t.fn(ctx)
		if err == nil {
			fmt.Printf("%-45s ALLOWED\n", t.name)
			passed++
		} else {
			fmt.Printf("%-45s DENIED: %v\n", t.name, err)
			denied++
		}
	}

	fmt.Println()
	fmt.Printf("Summary: %d allowed, %d denied out of %d tests\n", passed, denied, len(tests))
}
