// Audit permissions for installed apps via the ActivityManager.
//
// Uses IActivityManager via the "activity" service to check specific
// permissions for well-known packages, and IPackageManager to enumerate
// installed packages.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/permission_audit ./examples/permission_audit/
//	adb push build/permission_audit /data/local/tmp/ && adb shell /data/local/tmp/permission_audit
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/app"
	"github.com/AndroidGoLab/binder/android/content/pm"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	permissionGranted = 0
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

	// Get ActivityManager for permission checks.
	amSvc, err := sm.GetService(ctx, servicemanager.ActivityService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get activity service: %v\n", err)
		os.Exit(1)
	}
	am := app.NewActivityManagerProxy(amSvc)

	// Get PackageManager for package enumeration.
	pkgSvc, err := sm.GetService(ctx, servicemanager.PackageService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get package service: %v\n", err)
		os.Exit(1)
	}
	pkgMgr := pm.NewPackageManagerProxy(pkgSvc)

	// Audit permissions for the current process.
	myPid := int32(os.Getpid())
	myUid := int32(os.Getuid())

	sensitivePermissions := []string{
		"android.permission.CAMERA",
		"android.permission.RECORD_AUDIO",
		"android.permission.ACCESS_FINE_LOCATION",
		"android.permission.READ_CONTACTS",
		"android.permission.READ_PHONE_STATE",
		"android.permission.SEND_SMS",
		"android.permission.INTERNET",
		"android.permission.READ_EXTERNAL_STORAGE",
		"android.permission.WRITE_EXTERNAL_STORAGE",
	}

	fmt.Printf("Permission audit for current process (pid=%d, uid=%d):\n", myPid, myUid)
	for _, perm := range sensitivePermissions {
		result, err := am.CheckPermission(ctx, perm, myPid, myUid)
		if err != nil {
			fmt.Printf("  %-50s ERROR: %v\n", perm, err)
			continue
		}
		status := "DENIED"
		if result == permissionGranted {
			status = "GRANTED"
		}
		fmt.Printf("  %-50s %s\n", perm, status)
	}

	// Check permissions for well-known packages (using uid=1000 for system).
	fmt.Println("\nPermission audit for system (uid=1000):")
	systemPermissions := []string{
		"android.permission.INTERNET",
		"android.permission.ACCESS_WIFI_STATE",
		"android.permission.BLUETOOTH",
	}
	for _, perm := range systemPermissions {
		result, err := am.CheckPermission(ctx, perm, myPid, 1000)
		if err != nil {
			fmt.Printf("  %-50s ERROR: %v\n", perm, err)
			continue
		}
		status := "DENIED"
		if result == permissionGranted {
			status = "GRANTED"
		}
		fmt.Printf("  %-50s %s\n", perm, status)
	}

	// Show a few installed packages with their UIDs.
	allPkgs, err := pkgMgr.GetAllPackages(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGetAllPackages: %v\n", err)
		return
	}

	fmt.Printf("\nInstalled packages (showing first 5 of %d):\n", len(allPkgs))
	limit := 5
	if len(allPkgs) < limit {
		limit = len(allPkgs)
	}
	for _, pkg := range allPkgs[:limit] {
		available, err := pkgMgr.IsPackageAvailable(ctx, pkg)
		if err != nil {
			fmt.Printf("  %-40s error: %v\n", pkg, err)
			continue
		}
		fmt.Printf("  %-40s available=%v\n", pkg, available)
	}
}
