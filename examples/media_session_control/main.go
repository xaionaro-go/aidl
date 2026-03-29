// Enumerate active media sessions and query global priority.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/media_session_control ./examples/media_session_control/
//	adb push media_session_control /data/local/tmp/ && adb shell /data/local/tmp/media_session_control
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/content"
	"github.com/AndroidGoLab/binder/android/media/session"
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

	svc, err := sm.GetService(ctx, servicemanager.MediaSessionService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get media_session service: %v\n", err)
		os.Exit(1)
	}

	mgr := session.NewSessionManagerProxy(svc)

	// Check global priority state.
	active, err := mgr.IsGlobalPriorityActive(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsGlobalPriorityActive: %v\n", err)
	} else {
		fmt.Printf("Global priority active: %v\n", active)
	}

	// List active sessions (pass empty ComponentName for unfiltered list).
	sessions, err := mgr.GetSessions(ctx, content.ComponentName{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSessions: %v\n", err)
	} else {
		fmt.Printf("Active sessions: %d\n", len(sessions))
	}
}
