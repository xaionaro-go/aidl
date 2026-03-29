// Monitor installed packages by polling the PackageManager.
//
// Uses IPackageManager via the "package" service to list all installed
// packages and detect changes by comparing against a previous snapshot.
// This demonstrates how to build a package monitoring tool using the
// binder interface, since real-time broadcast receivers require a
// full Android app context.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/package_monitor ./examples/package_monitor/
//	adb push build/package_monitor /data/local/tmp/ && adb shell /data/local/tmp/package_monitor
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/AndroidGoLab/binder/android/content/pm"
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

	svc, err := sm.GetService(ctx, servicemanager.PackageService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get package service: %v\n", err)
		os.Exit(1)
	}

	pkgMgr := pm.NewPackageManagerProxy(svc)

	// Take initial snapshot.
	baseline, err := pkgMgr.GetAllPackages(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAllPackages: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(baseline)

	fmt.Printf("Baseline: %d packages installed\n", len(baseline))
	fmt.Println("Monitoring for package changes (poll every 5s, 3 iterations)...")

	// Poll for changes.
	for i := 0; i < 3; i++ {
		time.Sleep(5 * time.Second)

		current, err := pkgMgr.GetAllPackages(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  poll %d: GetAllPackages: %v\n", i+1, err)
			continue
		}
		sort.Strings(current)

		baseSet := toSet(baseline)
		currSet := toSet(current)

		added := diff(currSet, baseSet)
		removed := diff(baseSet, currSet)

		switch {
		case len(added) > 0 || len(removed) > 0:
			fmt.Printf("  poll %d: %d packages (", i+1, len(current))
			if len(added) > 0 {
				fmt.Printf("+%d installed", len(added))
				for _, pkg := range added {
					fmt.Printf("\n    + %s", pkg)
				}
			}
			if len(removed) > 0 {
				if len(added) > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("-%d removed", len(removed))
				for _, pkg := range removed {
					fmt.Printf("\n    - %s", pkg)
				}
			}
			fmt.Println(")")
			baseline = current
		default:
			fmt.Printf("  poll %d: %d packages (no changes)\n", i+1, len(current))
		}
	}
}

func toSet(items []string) map[string]struct{} {
	s := make(map[string]struct{}, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

func diff(
	a map[string]struct{},
	b map[string]struct{},
) []string {
	var result []string
	for k := range a {
		if _, ok := b[k]; !ok {
			result = append(result, k)
		}
	}
	sort.Strings(result)
	return result
}
