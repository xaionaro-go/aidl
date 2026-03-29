// Query status bar state: navigation bar mode, tracing, last system key.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/statusbar_control ./examples/statusbar_control/
//	adb push statusbar_control /data/local/tmp/ && adb shell /data/local/tmp/statusbar_control
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/com/android/internal_/statusbar"
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

	svc, err := sm.GetService(ctx, servicemanager.StatusBarService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get statusbar service: %v\n", err)
		os.Exit(1)
	}

	sb := statusbar.NewStatusBarServiceProxy(svc)

	// Query navigation bar mode.
	navMode, err := sb.GetNavBarMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetNavBarMode: %v\n", err)
	} else {
		modeNames := map[int32]string{
			0: "3-button",
			1: "2-button",
			2: "gesture",
		}
		name := modeNames[navMode]
		if name == "" {
			name = fmt.Sprintf("unknown(%d)", navMode)
		}
		fmt.Printf("Nav bar mode: %s\n", name)
	}

	// Check if system UI tracing is active.
	tracing, err := sb.IsTracing(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsTracing: %v\n", err)
	} else {
		fmt.Printf("Status bar tracing: %v\n", tracing)
	}

	// Query last system key.
	lastKey, err := sb.GetLastSystemKey(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetLastSystemKey: %v\n", err)
	} else {
		fmt.Printf("Last system key: %d\n", lastKey)
	}
}
