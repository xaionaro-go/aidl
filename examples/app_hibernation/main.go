// Query app hibernation status via the AppHibernationService.
//
// Uses IAppHibernationService via the "app_hibernation" service to check
// which packages are hibernating globally and for the current user, and
// whether OAT artifact deletion is enabled.
//
// Note: AppHibernationService was introduced in Android 12 (API 31).
// Requires system permissions to query hibernation state.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/app_hibernation ./examples/app_hibernation/
//	adb push build/app_hibernation /data/local/tmp/ && adb shell /data/local/tmp/app_hibernation
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/apphibernation"
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

	svc, err := sm.GetService(ctx, servicemanager.AppHibernationService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get app_hibernation service: %v\n", err)
		fmt.Fprintf(os.Stderr, "(AppHibernationService not available, requires Android 12+)\n")
		os.Exit(1)
	}

	hibMgr := apphibernation.NewAppHibernationServiceProxy(svc)

	// Check OAT artifact deletion setting.
	oatDeletion, err := hibMgr.IsOatArtifactDeletionEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsOatArtifactDeletionEnabled: %v\n", err)
	} else {
		fmt.Printf("OAT artifact deletion enabled: %v\n", oatDeletion)
	}

	// List packages hibernating for the current user.
	hibPkgs, err := hibMgr.GetHibernatingPackagesForUser(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetHibernatingPackagesForUser: %v\n", err)
	} else if len(hibPkgs) == 0 {
		fmt.Println("\nHibernating packages (current user): (none)")
	} else {
		fmt.Printf("\nHibernating packages (current user): %d\n", len(hibPkgs))
		for _, pkg := range hibPkgs {
			fmt.Printf("  %s\n", pkg)
		}
	}

	// Check hibernation for well-known packages.
	checkPkgs := []string{
		"com.android.settings",
		"com.android.chrome",
		"com.google.android.gms",
	}

	fmt.Println("\nPer-package hibernation status:")
	for _, pkg := range checkPkgs {
		globallyHibernating, err := hibMgr.IsHibernatingGlobally(ctx, pkg)
		if err != nil {
			fmt.Printf("  %-40s global: error (%v)\n", pkg, err)
			continue
		}

		userHibernating, err := hibMgr.IsHibernatingForUser(ctx, pkg)
		if err != nil {
			fmt.Printf("  %-40s global=%v, user: error (%v)\n", pkg, globallyHibernating, err)
			continue
		}

		fmt.Printf("  %-40s global=%v, user=%v\n", pkg, globallyHibernating, userHibernating)
	}
}
