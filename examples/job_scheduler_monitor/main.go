// Query JobScheduler state from the "jobscheduler" service.
//
// Lists all pending jobs across namespaces and reports summary counts.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/job_scheduler_monitor ./examples/job_scheduler_monitor/
//	adb push build/job_scheduler_monitor /data/local/tmp/ && adb shell /data/local/tmp/job_scheduler_monitor
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/app/job"
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

	svc, err := sm.GetService(ctx, servicemanager.JobSchedulerService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get jobscheduler service: %v\n", err)
		os.Exit(1)
	}

	js := job.NewJobSchedulerProxy(svc)

	// Get all pending jobs (grouped by namespace).
	pendingJobs, err := js.GetAllPendingJobs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAllPendingJobs: %v\n", err)
		fmt.Println("Note: JobScheduler queries may require system permissions.")
		os.Exit(1)
	}

	totalJobs := 0
	for ns := range pendingJobs {
		totalJobs++
		_ = ns
	}
	fmt.Printf("Pending job namespaces: %d\n", totalJobs)

	for ns := range pendingJobs {
		label := ns
		if label == "" {
			label = "(default)"
		}
		fmt.Printf("  namespace: %s\n", label)
	}

	// Get all job snapshots.
	snapshots, err := js.GetAllJobSnapshots(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAllJobSnapshots: %v\n", err)
	} else {
		_ = snapshots
		fmt.Println("\nGetAllJobSnapshots: succeeded")
	}
}
