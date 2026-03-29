// Lightweight MDM agent querying device policies via DevicePolicyManager.
//
// Queries password quality, encryption status, camera policy, provisioning
// state, and device owner information.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/mdm_agent ./examples/mdm_agent/
//	adb push build/mdm_agent /data/local/tmp/ && adb shell /data/local/tmp/mdm_agent
package main

import (
	"context"
	"fmt"
	"os"

	genAdmin "github.com/AndroidGoLab/binder/android/app/admin"
	"github.com/AndroidGoLab/binder/android/content"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

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

	dpm, err := genAdmin.GetDevicePolicyManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get device_policy service: %v\n", err)
		os.Exit(1)
	}

	// An empty ComponentName queries the global (aggregated) policy.
	emptyAdmin := content.ComponentName{}
	callerPkg := binder.DefaultCallerIdentity().PackageName

	// Password quality
	quality, err := dpm.GetPasswordQuality(ctx, emptyAdmin, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetPasswordQuality: %v\n", err)
	} else {
		fmt.Printf("Password quality:          0x%x\n", quality)
	}

	// Storage encryption status
	encStatus, err := dpm.GetStorageEncryptionStatus(ctx, callerPkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetStorageEncryptionStatus: %v\n", err)
	} else {
		names := map[int32]string{
			0: "unsupported",
			1: "inactive",
			2: "activating",
			3: "active_default_key",
			4: "active_per_user",
			5: "active",
		}
		name := names[encStatus]
		if name == "" {
			name = fmt.Sprintf("unknown(%d)", encStatus)
		}
		fmt.Printf("Encryption status:         %s (%d)\n", name, encStatus)
	}

	// Camera disabled
	camDisabled, err := dpm.GetCameraDisabled(ctx, emptyAdmin, callerPkg, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCameraDisabled: %v\n", err)
	} else {
		fmt.Printf("Camera disabled:           %v\n", camDisabled)
	}

	// Screen capture disabled
	scDisabled, err := dpm.GetScreenCaptureDisabled(ctx, emptyAdmin, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetScreenCaptureDisabled: %v\n", err)
	} else {
		fmt.Printf("Screen capture disabled:   %v\n", scDisabled)
	}

	// Device provisioned
	provisioned, err := dpm.IsDeviceProvisioned(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsDeviceProvisioned: %v\n", err)
	} else {
		fmt.Printf("Device provisioned:        %v\n", provisioned)
	}

	// Auto time required
	autoTime, err := dpm.GetAutoTimeRequired(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAutoTimeRequired: %v\n", err)
	} else {
		fmt.Printf("Auto time required:        %v\n", autoTime)
	}

	// Device owner name
	ownerName, err := dpm.GetDeviceOwnerName(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDeviceOwnerName: %v\n", err)
	} else {
		if ownerName == "" {
			ownerName = "(none)"
		}
		fmt.Printf("Device owner:              %s\n", ownerName)
	}

	// Active admins
	admins, err := dpm.GetActiveAdmins(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetActiveAdmins: %v\n", err)
	} else {
		fmt.Printf("Active admins:             %d\n", len(admins))
	}

	// Max failed passwords for wipe
	maxFailed, err := dpm.GetMaximumFailedPasswordsForWipe(ctx, emptyAdmin, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMaximumFailedPasswordsForWipe: %v\n", err)
	} else {
		fmt.Printf("Max failed passwords/wipe: %d\n", maxFailed)
	}

	// Maximum time to lock (ms)
	maxTimeLock, err := dpm.GetMaximumTimeToLock(ctx, emptyAdmin, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMaximumTimeToLock: %v\n", err)
	} else {
		if maxTimeLock == 0 {
			fmt.Printf("Max time to lock:          unlimited\n")
		} else {
			fmt.Printf("Max time to lock:          %d ms\n", maxTimeLock)
		}
	}
}
