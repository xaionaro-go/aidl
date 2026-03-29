// Poll thermal service for CPU/GPU temperatures, throttling status, and cooling devices.
//
// Uses the generated IThermalService proxy via the "thermalservice" binder
// service and IHardwarePropertiesManager via "hardware_properties" for
// CPU usage and device temperatures.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/thermal_monitor ./examples/thermal_monitor/
//	adb push build/thermal_monitor /data/local/tmp/ && adb shell /data/local/tmp/thermal_monitor
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// Android temperature type constants (from android.os.Temperature).
const (
	temperatureTypeCPU         int32 = 0
	temperatureTypeGPU         int32 = 1
	temperatureTypeBattery     int32 = 2
	temperatureTypeSkin        int32 = 3
	temperatureTypeUSBPort     int32 = 4
	temperatureTypePowerAmplif int32 = 5
	temperatureTypeBCL         int32 = 6
	temperatureTypeNPU         int32 = 7
)

// Android HardwarePropertiesManager temperature type constants.
const (
	deviceTempCPU     int32 = 0
	deviceTempGPU     int32 = 1
	deviceTempBattery int32 = 2
	deviceTempSkin    int32 = 3
)

// Android HardwarePropertiesManager temperature source constants.
const (
	tempSourceCurrent  int32 = 0
	tempSourceThrottle int32 = 1
	tempSourceShutdown int32 = 2
)

var temperatureTypeNames = map[int32]string{
	temperatureTypeCPU:         "CPU",
	temperatureTypeGPU:         "GPU",
	temperatureTypeBattery:     "Battery",
	temperatureTypeSkin:        "Skin",
	temperatureTypeUSBPort:     "USB Port",
	temperatureTypePowerAmplif: "Power Amp",
	temperatureTypeBCL:         "BCL",
	temperatureTypeNPU:         "NPU",
}

var thermalStatusNames = []string{
	"none", "light", "moderate", "severe", "critical", "emergency", "shutdown",
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

	// --- Thermal Service ---
	thermalSvc, err := sm.GetService(ctx, servicemanager.ThermalService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get thermal service: %v\n", err)
	} else {
		thermal := genOs.NewThermalServiceProxy(thermalSvc)

		// Current thermal status
		status, err := thermal.GetCurrentThermalStatus(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetCurrentThermalStatus: %v\n", err)
		} else {
			name := "unknown"
			if int(status) < len(thermalStatusNames) {
				name = thermalStatusNames[status]
			}
			fmt.Printf("Thermal status:     %s (%d)\n", name, status)
		}

		// Thermal headroom forecast
		headroom, err := thermal.GetThermalHeadroom(ctx, 10)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetThermalHeadroom: %v\n", err)
		} else {
			fmt.Printf("Thermal headroom:   %.2f (10s forecast)\n", headroom)
		}

		// All current temperatures
		temps, err := thermal.GetCurrentTemperatures(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetCurrentTemperatures: %v\n", err)
		} else {
			fmt.Printf("\nTemperature sensors: %d\n", len(temps))
			for _, t := range temps {
				typeName := temperatureTypeNames[t.Type]
				if typeName == "" {
					typeName = fmt.Sprintf("type(%d)", t.Type)
				}
				statusName := "ok"
				if int(t.Status) < len(thermalStatusNames) {
					statusName = thermalStatusNames[t.Status]
				}
				fmt.Printf("  %-20s %-10s %.1f C  status=%s\n", t.Name, typeName, t.Value, statusName)
			}
		}

		// Cooling devices
		coolers, err := thermal.GetCurrentCoolingDevices(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetCurrentCoolingDevices: %v\n", err)
		} else {
			fmt.Printf("\nCooling devices: %d\n", len(coolers))
			for _, c := range coolers {
				fmt.Printf("  %-20s type=%d  value=%d\n", c.Name, c.Type, c.Value)
			}
		}
	}

	// --- Hardware Properties Manager ---
	hwProps, err := genOs.GetHardwarePropertiesManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get hardware_properties service: %v\n", err)
		return
	}

	// CPU temperatures (current)
	cpuTemps, err := hwProps.GetDeviceTemperatures(ctx, deviceTempCPU, tempSourceCurrent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDeviceTemperatures(CPU): %v\n", err)
	} else {
		fmt.Printf("\nCPU temperatures (current): %v C\n", cpuTemps)
	}

	// CPU throttling thresholds
	cpuThrottle, err := hwProps.GetDeviceTemperatures(ctx, deviceTempCPU, tempSourceThrottle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDeviceTemperatures(CPU,throttle): %v\n", err)
	} else {
		fmt.Printf("CPU throttle thresholds:    %v C\n", cpuThrottle)
	}

	// GPU temperatures
	gpuTemps, err := hwProps.GetDeviceTemperatures(ctx, deviceTempGPU, tempSourceCurrent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDeviceTemperatures(GPU): %v\n", err)
	} else {
		fmt.Printf("GPU temperatures (current): %v C\n", gpuTemps)
	}

	// CPU usage
	cpuUsages, err := hwProps.GetCpuUsages(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCpuUsages: %v\n", err)
	} else {
		fmt.Printf("\nCPU usage: %d cores\n", len(cpuUsages))
		for i, u := range cpuUsages {
			pct := float64(0)
			if u.Total > 0 {
				pct = float64(u.Active) / float64(u.Total) * 100
			}
			fmt.Printf("  Core %d: %.1f%% (active=%d total=%d)\n", i, pct, u.Active, u.Total)
		}
	}

	// Fan speeds
	fans, err := hwProps.GetFanSpeeds(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetFanSpeeds: %v\n", err)
	} else if len(fans) > 0 {
		fmt.Printf("\nFan speeds: %v RPM\n", fans)
	}
}
