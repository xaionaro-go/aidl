// Sensor data collection relay: list available sensors and query defaults.
//
// Uses the frameworks SensorManager (android.frameworks.sensorservice)
// to enumerate device sensors and query the default accelerometer.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/sensor_gateway ./examples/sensor_gateway/
//	adb push build/sensor_gateway /data/local/tmp/ && adb shell /data/local/tmp/sensor_gateway
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

	// The frameworks sensor manager is an AIDL service, not the SDK
	// Context.SENSOR_SERVICE ("sensor"). Use the full AIDL instance name.
	sensorSvc, err := sm.GetService(ctx, "android.frameworks.sensorservice.ISensorManager/default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get sensor service: %v\n", err)
		os.Exit(1)
	}

	sensorMgr := sensorservice.NewSensorManagerProxy(sensorSvc)

	fmt.Println("=== Sensor Gateway ===")
	fmt.Println()

	// List all sensors.
	sensorList, err := sensorMgr.GetSensorList(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSensorList: %v\n", err)
	} else {
		fmt.Printf("Available sensors: %d\n\n", len(sensorList))
		for i, s := range sensorList {
			typeName := sensorTypeName(s.Type)
			fmt.Printf("  [%2d] %-30s type=%-25s handle=%d\n",
				i, s.Name, typeName, s.SensorHandle)
		}
	}
	fmt.Println()

	// Query default accelerometer.
	accel, err := sensorMgr.GetDefaultSensor(ctx, sensors.SensorTypeACCELEROMETER)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDefaultSensor(ACCELEROMETER): %v\n", err)
	} else {
		fmt.Printf("Default accelerometer:\n")
		fmt.Printf("  Name:       %s\n", accel.Name)
		fmt.Printf("  Vendor:     %s\n", accel.Vendor)
		fmt.Printf("  Resolution: %f\n", accel.Resolution)
		fmt.Printf("  Max range:  %f\n", accel.MaxRange)
		fmt.Printf("  Power:      %f mA\n", accel.Power)
	}

	// Query default light sensor.
	light, err := sensorMgr.GetDefaultSensor(ctx, sensors.SensorTypeLIGHT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDefaultSensor(LIGHT): %v\n", err)
	} else {
		fmt.Printf("\nDefault light sensor:\n")
		fmt.Printf("  Name:       %s\n", light.Name)
		fmt.Printf("  Max range:  %f lux\n", light.MaxRange)
	}
}

func sensorTypeName(t sensors.SensorType) string {
	names := map[sensors.SensorType]string{
		sensors.SensorTypeACCELEROMETER:   "ACCELEROMETER",
		sensors.SensorTypeMagneticField:   "MAGNETIC_FIELD",
		sensors.SensorTypeGYROSCOPE:       "GYROSCOPE",
		sensors.SensorTypeLIGHT:           "LIGHT",
		sensors.SensorTypePRESSURE:        "PRESSURE",
		sensors.SensorTypePROXIMITY:       "PROXIMITY",
		sensors.SensorTypeGRAVITY:         "GRAVITY",
		sensors.SensorTypeRotationVector:   "ROTATION_VECTOR",
		sensors.SensorTypeAmbientTemperature: "AMBIENT_TEMPERATURE",
		sensors.SensorTypeStepCounter:      "STEP_COUNTER",
		sensors.SensorTypeStepDetector:     "STEP_DETECTOR",
		sensors.SensorTypeHeartRate:        "HEART_RATE",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return fmt.Sprintf("TYPE_%d", t)
}
