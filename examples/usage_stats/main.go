// Query app usage statistics via the UsageStatsManager.
//
// Uses IUsageStatsManager via the "usagestats" service to check
// app standby status, standby buckets, and app inactivity state.
//
// Note: Most usage stats queries require PACKAGE_USAGE_STATS
// permission. App standby enabled/bucket queries may work without it.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/usage_stats ./examples/usage_stats/
//	adb push build/usage_stats /data/local/tmp/ && adb shell /data/local/tmp/usage_stats
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/app/usage"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// Standby bucket constants from Android's UsageStatsManager.
const (
	standbyBucketExempted   = 5
	standbyBucketActive     = 10
	standbyBucketWorkingSet = 20
	standbyBucketFrequent   = 30
	standbyBucketRare       = 40
	standbyBucketRestricted = 45
	standbyBucketNever      = 50
)

func bucketName(bucket int32) string {
	switch {
	case bucket <= standbyBucketExempted:
		return "EXEMPTED"
	case bucket <= standbyBucketActive:
		return "ACTIVE"
	case bucket <= standbyBucketWorkingSet:
		return "WORKING_SET"
	case bucket <= standbyBucketFrequent:
		return "FREQUENT"
	case bucket <= standbyBucketRare:
		return "RARE"
	case bucket <= standbyBucketRestricted:
		return "RESTRICTED"
	default:
		return "NEVER"
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

	usageMgr, err := usage.GetUsageStatsManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get usagestats service: %v\n", err)
		os.Exit(1)
	}

	// Check if app standby is enabled.
	standbyEnabled, err := usageMgr.IsAppStandbyEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsAppStandbyEnabled: %v\n", err)
	} else {
		fmt.Printf("App standby enabled: %v\n", standbyEnabled)
	}

	// Check standby buckets and inactivity for well-known packages.
	packages := []string{
		"com.android.settings",
		"com.android.systemui",
		"com.android.chrome",
		"com.google.android.gms",
		"com.android.shell",
	}

	fmt.Println("\nApp standby status:")
	for _, pkg := range packages {
		bucket, err := usageMgr.GetAppStandbyBucket(ctx, pkg)
		if err != nil {
			fmt.Printf("  %-40s bucket: error (%v)\n", pkg, err)
			continue
		}

		inactive, err := usageMgr.IsAppInactive(ctx, pkg)
		if err != nil {
			fmt.Printf("  %-40s bucket=%d (%s), inactive: error\n",
				pkg, bucket, bucketName(bucket))
			continue
		}

		fmt.Printf("  %-40s bucket=%d (%s), inactive=%v\n",
			pkg, bucket, bucketName(bucket), inactive)
	}

	// Check usage source.
	usageSource, err := usageMgr.GetUsageSource(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGetUsageSource: %v\n", err)
	} else {
		fmt.Printf("\nUsage source: %d\n", usageSource)
	}
}
