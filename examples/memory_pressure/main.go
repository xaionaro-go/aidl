// Read memory pressure info from ActivityManager.
//
// Uses the generated IActivityManager proxy via the "activity" binder
// service. Queries system memory info, memory trim level, and per-process
// memory state.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/memory_pressure ./examples/memory_pressure/
//	adb push build/memory_pressure /data/local/tmp/ && adb shell /data/local/tmp/memory_pressure
package main

import (
	"context"
	"fmt"
	"os"

	genApp "github.com/AndroidGoLab/binder/android/app"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// Android memory trim level constants.
const (
	trimLevelRunningModerate int32 = 5
	trimLevelRunningLow      int32 = 10
	trimLevelRunningCritical int32 = 15
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

	am := genApp.NewActivityManagerProxy(svc)

	// System-wide memory info via GetMemoryInfo.
	// The generated ActivityManagerMemoryInfo parcelable has no fields
	// exposed (Java-only Parcelable), but the call itself succeeds and
	// confirms the service is responsive.
	memInfo := genApp.ActivityManagerMemoryInfo{}
	err = am.GetMemoryInfo(ctx, memInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMemoryInfo: %v\n", err)
	} else {
		fmt.Println("GetMemoryInfo: call succeeded (parcelable fields not exposed in AIDL)")
	}

	// Memory trim level
	trimLevel, err := am.GetMemoryTrimLevel(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMemoryTrimLevel: %v\n", err)
	} else {
		trimName := "unknown"
		switch {
		case trimLevel == 0:
			trimName = "none"
		case trimLevel <= trimLevelRunningModerate:
			trimName = "running_moderate"
		case trimLevel <= trimLevelRunningLow:
			trimName = "running_low"
		case trimLevel <= trimLevelRunningCritical:
			trimName = "running_critical"
		default:
			trimName = fmt.Sprintf("level(%d)", trimLevel)
		}
		fmt.Printf("Memory trim level: %s (%d)\n", trimName, trimLevel)
	}

	// Process limit (related to memory management)
	limit, err := am.GetProcessLimit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetProcessLimit: %v\n", err)
	} else {
		fmt.Printf("Process limit:     %d", limit)
		if limit < 0 {
			fmt.Print(" (unlimited)")
		}
		fmt.Println()
	}

	// Our own memory state
	myState := genApp.ActivityManagerRunningAppProcessInfo{}
	err = am.GetMyMemoryState(ctx, myState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMyMemoryState: %v\n", err)
	} else {
		fmt.Println("GetMyMemoryState:  call succeeded (parcelable fields not exposed in AIDL)")
	}

	// Process memory info for our PID
	pid := int32(os.Getpid())
	memInfos, err := am.GetProcessMemoryInfo(ctx, []int32{pid})
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetProcessMemoryInfo(%d): %v\n", pid, err)
	} else {
		fmt.Printf("GetProcessMemoryInfo: %d entries for PID %d\n", len(memInfos), pid)
	}

	// App freezer (relates to memory pressure response)
	freezer, err := am.IsAppFreezerSupported(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsAppFreezerSupported: %v\n", err)
	} else {
		fmt.Printf("App freezer:       %v\n", freezer)
	}

	// Running processes count
	procs, err := am.GetRunningAppProcesses(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetRunningAppProcesses: %v\n", err)
	} else {
		fmt.Printf("Running processes: %d\n", len(procs))
	}
}
