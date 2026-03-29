// Audit pending alarms via AlarmManager.
//
// Queries the "alarm" service for the next alarm clock, next wake-from-idle
// time, and alarm configuration version.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/alarm_auditor ./examples/alarm_auditor/
//	adb push build/alarm_auditor /data/local/tmp/ && adb shell /data/local/tmp/alarm_auditor
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AndroidGoLab/binder/android/app"
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

	svc, err := sm.GetService(ctx, servicemanager.AlarmService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get alarm service: %v\n", err)
		os.Exit(1)
	}

	am := app.NewAlarmManagerProxy(svc)

	// Query next wake-from-idle time.
	nextWake, err := am.GetNextWakeFromIdleTime(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetNextWakeFromIdleTime: %v\n", err)
	} else {
		if nextWake > 0 {
			wakeTime := time.UnixMilli(nextWake)
			fmt.Printf("Next wake-from-idle: %s (%d ms)\n", wakeTime.Format(time.RFC3339), nextWake)
		} else {
			fmt.Printf("Next wake-from-idle: none scheduled\n")
		}
	}

	// Query next alarm clock.
	alarmClock, err := am.GetNextAlarmClock(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetNextAlarmClock: %v\n", err)
	} else {
		_ = alarmClock
		fmt.Println("GetNextAlarmClock:   succeeded")
	}

	// Query alarm config version.
	configVersion, err := am.GetConfigVersion(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetConfigVersion: %v\n", err)
	} else {
		fmt.Printf("Alarm config version: %d\n", configVersion)
	}
}
