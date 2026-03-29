// Read sensor data from the SensorManager HAL.
//
// Lists all available hardware sensors (accelerometer, gyroscope, etc.)
// and queries the default accelerometer sensor info.
//
// The sensor HAL service is accessed via the HIDL service manager at
// "android.frameworks.sensorservice.ISensorManager/default", not the
// standard binder servicemanager "sensor" entry.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/sensor_reader ./examples/sensor_reader/
//	adb push build/sensor_reader /data/local/tmp/ && adb shell /data/local/tmp/sensor_reader
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/frameworks/sensorservice"
	"github.com/AndroidGoLab/binder/android/hardware/sensors"
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

	// The sensor HAL is registered under its HIDL fully-qualified name.
	svc, err := sm.GetService(ctx, "android.frameworks.sensorservice.ISensorManager/default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get sensor service: %v\n", err)
		os.Exit(1)
	}

	mgr := sensorservice.NewSensorManagerProxy(svc)

	// List all sensors.
	sensorList, err := mgr.GetSensorList(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSensorList: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d sensors:\n\n", len(sensorList))
	for i, s := range sensorList {
		fmt.Printf("  [%2d] %-40s type=%-3d vendor=%s\n",
			i+1, s.Name, s.Type, s.Vendor)
		fmt.Printf("       range=%.2f resolution=%.6f power=%.2f mA\n",
			s.MaxRange, s.Resolution, s.Power)
	}

	// Query the default accelerometer.
	accel, err := mgr.GetDefaultSensor(ctx, sensors.SensorTypeACCELEROMETER)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nGetDefaultSensor(ACCELEROMETER): %v\n", err)
	} else {
		fmt.Printf("\nDefault accelerometer:\n")
		fmt.Printf("  Name:       %s\n", accel.Name)
		fmt.Printf("  Vendor:     %s\n", accel.Vendor)
		fmt.Printf("  MaxRange:   %.2f\n", accel.MaxRange)
		fmt.Printf("  Resolution: %.6f\n", accel.Resolution)
		fmt.Printf("  Power:      %.2f mA\n", accel.Power)
	}
}
