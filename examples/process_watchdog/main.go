// List running processes via ActivityManager and check resource usage.
//
// Uses the generated IActivityManager proxy via the "activity" binder
// service. Queries process list, running services, process limits,
// per-process memory (PSS), system configuration, and various system
// state indicators.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/process_watchdog ./examples/process_watchdog/
//	adb push build/process_watchdog /data/local/tmp/ && adb shell /data/local/tmp/process_watchdog
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

	fmt.Println("=== Process Watchdog ===")
	fmt.Println()

	// System configuration
	config, err := am.GetConfiguration(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetConfiguration: %v\n", err)
	} else {
		fmt.Println("-- System Configuration --")
		fmt.Printf("  Font scale:    %.1f\n", config.FontScale)
		fmt.Printf("  Orientation:   %s\n", orientationName(config.Orientation))
		fmt.Printf("  Touchscreen:   %s\n", touchscreenName(config.Touchscreen))
		fmt.Printf("  Keyboard:      %s\n", keyboardName(config.Keyboard))
		if config.Mcc != 0 || config.Mnc != 0 {
			fmt.Printf("  MCC/MNC:       %d/%d\n", config.Mcc, config.Mnc)
		}
		fmt.Println()
	}

	// Process limit
	fmt.Println("-- Process State --")
	limit, err := am.GetProcessLimit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetProcessLimit: %v\n", err)
	} else {
		fmt.Printf("Process limit:   %d", limit)
		if limit < 0 {
			fmt.Print(" (unlimited)")
		}
		fmt.Println()
	}

	// Current user
	userId, err := am.GetCurrentUserId(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCurrentUserId: %v\n", err)
	} else {
		fmt.Printf("Current user:    %d\n", userId)
	}

	// Running processes — count items
	procs, err := am.GetRunningAppProcesses(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetRunningAppProcesses: %v\n", err)
	} else {
		fmt.Printf("Running procs:   %d\n", len(procs))
	}

	// Running services — count items
	services, err := am.GetServices(ctx, 100, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetServices: %v\n", err)
	} else {
		fmt.Printf("Running services: %d\n", len(services))
	}

	// UID activity check for our own process
	uid := int32(os.Getuid())
	pid := int32(os.Getpid())
	fmt.Println()
	fmt.Println("-- Own Process --")
	fmt.Printf("UID: %d  PID: %d\n", uid, pid)

	active, err := am.IsUidActive(ctx, uid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsUidActive: %v\n", err)
	} else {
		fmt.Printf("UID active:      %v\n", active)
	}

	procState, err := am.GetUidProcessState(ctx, uid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetUidProcessState: %v\n", err)
	} else {
		fmt.Printf("Process state:   %d (%s)\n", procState, processStateName(procState))
	}

	// Memory usage for our own PID
	pss, err := am.GetProcessPss(ctx, []int32{pid})
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetProcessPss: %v\n", err)
	} else if len(pss) > 0 {
		fmt.Printf("PSS memory:      %d KB\n", pss[0])
	}

	// System indicators
	fmt.Println()
	fmt.Println("-- System Indicators --")

	monkey, err := am.IsUserAMonkey(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsUserAMonkey: %v\n", err)
	} else {
		fmt.Printf("Monkey mode:     %v\n", monkey)
	}

	freezer, err := am.IsAppFreezerSupported(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsAppFreezerSupported: %v\n", err)
	} else {
		fmt.Printf("App freezer:     %v\n", freezer)
	}

	lockMode, err := am.GetLockTaskModeState(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetLockTaskModeState: %v\n", err)
	} else {
		modeNames := map[int32]string{
			0: "none",
			1: "locked",
			2: "pinned",
		}
		name := modeNames[lockMode]
		if name == "" {
			name = fmt.Sprintf("unknown(%d)", lockMode)
		}
		fmt.Printf("Lock task:       %s\n", name)
	}
}

func orientationName(o int32) string {
	switch o {
	case 0:
		return "undefined"
	case 1:
		return "portrait"
	case 2:
		return "landscape"
	default:
		return fmt.Sprintf("%d", o)
	}
}

func touchscreenName(t int32) string {
	switch t {
	case 0:
		return "undefined"
	case 1:
		return "notouch"
	case 2:
		return "stylus"
	case 3:
		return "finger"
	default:
		return fmt.Sprintf("%d", t)
	}
}

func keyboardName(k int32) string {
	switch k {
	case 0:
		return "undefined"
	case 1:
		return "nokeys"
	case 2:
		return "qwerty"
	case 3:
		return "12key"
	default:
		return fmt.Sprintf("%d", k)
	}
}

func processStateName(state int32) string {
	// From android.app.ActivityManager.PROCESS_STATE_*
	switch {
	case state <= 2:
		return "top/foreground"
	case state <= 5:
		return "important/visible"
	case state <= 8:
		return "service"
	case state <= 13:
		return "cached"
	default:
		return "background"
	}
}
