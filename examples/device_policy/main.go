// Query DevicePolicyManager for device administration state.
//
// Checks storage encryption status, active device admin list, and
// related device policy information.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/device_policy ./examples/device_policy/
//	adb push build/device_policy /data/local/tmp/ && adb shell /data/local/tmp/device_policy
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/app/admin"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// Encryption status constants from DevicePolicyManager.
const (
	encryptionStatusUnsupported       = 0
	encryptionStatusInactive          = 1
	encryptionStatusActivating        = 2
	encryptionStatusActiveDefaultKey  = 3
	encryptionStatusActivePerUser     = 4
	encryptionStatusActivatingPerUser = 5
)

func encryptionStatusString(status int32) string {
	switch status {
	case encryptionStatusUnsupported:
		return "UNSUPPORTED"
	case encryptionStatusInactive:
		return "INACTIVE"
	case encryptionStatusActivating:
		return "ACTIVATING"
	case encryptionStatusActiveDefaultKey:
		return "ACTIVE_DEFAULT_KEY"
	case encryptionStatusActivePerUser:
		return "ACTIVE_PER_USER"
	case encryptionStatusActivatingPerUser:
		return "ACTIVATING_PER_USER"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
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

	dpm, err := admin.GetDevicePolicyManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get device_policy service: %v\n", err)
		os.Exit(1)
	}

	// Query storage encryption status.
	encStatus, err := dpm.GetStorageEncryptionStatus(ctx, "com.android.shell")
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetStorageEncryptionStatus: %v\n", err)
	} else {
		fmt.Printf("Storage encryption: %s\n", encryptionStatusString(encStatus))
	}

	// Count active device admins.
	// ComponentName is an opaque parcelable in the generated code,
	// so we can only report the count.
	admins, err := dpm.GetActiveAdmins(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetActiveAdmins: %v\n", err)
	} else {
		fmt.Printf("\nActive device admins: %d\n", len(admins))
	}
}
