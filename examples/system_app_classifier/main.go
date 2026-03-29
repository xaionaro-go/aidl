// Classify installed packages as system or user apps.
//
// Uses IPackageManager via the "package" service to get ApplicationInfo
// for each installed package, then checks the Flags field to distinguish
// system apps (FLAG_SYSTEM = 0x1) from user-installed apps.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/system_app_classifier ./examples/system_app_classifier/
//	adb push build/system_app_classifier /data/local/tmp/ && adb shell /data/local/tmp/system_app_classifier
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/content/pm"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	// ApplicationInfo.Flags bitmask for system apps.
	flagSystem = 0x1
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

	svc, err := sm.GetService(ctx, servicemanager.PackageService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get package service: %v\n", err)
		os.Exit(1)
	}

	pkgMgr := pm.NewPackageManagerProxy(svc)

	allPkgs, err := pkgMgr.GetAllPackages(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAllPackages: %v\n", err)
		os.Exit(1)
	}

	var systemApps, userApps []string
	var errors int

	for _, pkg := range allPkgs {
		appInfo, err := pkgMgr.GetApplicationInfo(ctx, pkg, 0)
		if err != nil {
			errors++
			continue
		}

		if appInfo.Flags&flagSystem != 0 {
			systemApps = append(systemApps, pkg)
		} else {
			userApps = append(userApps, pkg)
		}
	}

	fmt.Printf("Total packages: %d\n", len(allPkgs))
	fmt.Printf("System apps:    %d\n", len(systemApps))
	fmt.Printf("User apps:      %d\n", len(userApps))
	if errors > 0 {
		fmt.Printf("Errors:         %d\n", errors)
	}

	fmt.Println("\n--- System apps ---")
	for _, pkg := range systemApps {
		fmt.Printf("  %s\n", pkg)
	}

	fmt.Println("\n--- User apps ---")
	if len(userApps) == 0 {
		fmt.Println("  (none)")
	}
	for _, pkg := range userApps {
		fmt.Printf("  %s\n", pkg)
	}
}
