// Check permissions for UIDs/PIDs via ActivityManager.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/permission_checker ./examples/permission_checker/
//	adb push permission_checker /data/local/tmp/ && adb shell /data/local/tmp/permission_checker
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/app"
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

	svc, err := sm.GetService(ctx, servicemanager.ActivityService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get activity service: %v\n", err)
		os.Exit(1)
	}

	am := app.NewActivityManagerProxy(svc)

	myPid := int32(os.Getpid())
	myUid := int32(os.Getuid())

	fmt.Printf("=== Permission Checker ===\n")
	fmt.Printf("PID: %d  UID: %d\n\n", myPid, myUid)

	permissions := []string{
		"android.permission.INTERNET",
		"android.permission.CAMERA",
		"android.permission.READ_PHONE_STATE",
		"android.permission.ACCESS_FINE_LOCATION",
		"android.permission.WRITE_EXTERNAL_STORAGE",
		"android.permission.RECORD_AUDIO",
		"android.permission.BLUETOOTH",
		"android.permission.NFC",
	}

	for _, perm := range permissions {
		result, err := am.CheckPermission(ctx, perm, myPid, myUid)
		if err != nil {
			fmt.Printf("  %-50s ERROR: %v\n", perm, err)
			continue
		}
		status := "DENIED"
		if result == 0 {
			status = "GRANTED"
		}
		fmt.Printf("  %-50s %s\n", perm, status)
	}

	// Also check with UID 0 (root) for comparison.
	fmt.Printf("\nPermission check for UID 0 (root):\n")
	for _, perm := range permissions[:3] {
		result, err := am.CheckPermission(ctx, perm, myPid, 0)
		if err != nil {
			fmt.Printf("  %-50s ERROR: %v\n", perm, err)
			continue
		}
		status := "DENIED"
		if result == 0 {
			status = "GRANTED"
		}
		fmt.Printf("  %-50s %s\n", perm, status)
	}

	// Query process limit.
	limit, err := am.GetProcessLimit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGetProcessLimit: %v\n", err)
	} else {
		fmt.Printf("\nProcess limit: %d\n", limit)
	}

	// Query monkey test state.
	monkey, err := am.IsUserAMonkey(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsUserAMonkey: %v\n", err)
	} else {
		fmt.Printf("Is monkey test: %v\n", monkey)
	}
}
