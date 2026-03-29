// Monitor charging status and battery health via the Health HAL.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/charge_monitor ./examples/charge_monitor/
//	adb push build/charge_monitor /data/local/tmp/ && adb shell /data/local/tmp/charge_monitor
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/hardware/health"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func batteryStatusString(s health.BatteryStatus) string {
	switch s {
	case health.BatteryStatusUNKNOWN:
		return "UNKNOWN"
	case health.BatteryStatusCHARGING:
		return "CHARGING"
	case health.BatteryStatusDISCHARGING:
		return "DISCHARGING"
	case health.BatteryStatusNotCharging:
		return "NOT_CHARGING"
	case health.BatteryStatusFULL:
		return "FULL"
	default:
		return fmt.Sprintf("(%d)", s)
	}
}

func main() {
	ctx := context.Background()

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer drv.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)

	svc, err := sm.GetService(ctx, servicemanager.ServiceName(health.DescriptorIHealth+"/default"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get health service: %v\n", err)
		os.Exit(1)
	}

	h := health.NewHealthProxy(svc)

	status, err := h.GetChargeStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetChargeStatus: %v\n", err)
	} else {
		fmt.Printf("Charge status:    %s\n", batteryStatusString(status))
	}

	capacity, err := h.GetCapacity(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCapacity: %v\n", err)
	} else {
		fmt.Printf("Battery level:    %d%%\n", capacity)
	}

	info, err := h.GetHealthInfo(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetHealthInfo: %v\n", err)
	} else {
		fmt.Printf("Charger AC:       %v\n", info.ChargerAcOnline)
		fmt.Printf("Charger USB:      %v\n", info.ChargerUsbOnline)
		fmt.Printf("Charger wireless: %v\n", info.ChargerWirelessOnline)
		fmt.Printf("Battery present:  %v\n", info.BatteryPresent)
		fmt.Printf("Battery voltage:  %d mV\n", info.BatteryVoltageMillivolts)
		fmt.Printf("Battery temp:     %.1f C\n", float64(info.BatteryTemperatureTenthsCelsius)/10.0)
		fmt.Printf("Battery current:  %d uA\n", info.BatteryCurrentMicroamps)
		fmt.Printf("Cycle count:      %d\n", info.BatteryCycleCount)
	}

	policy, err := h.GetChargingPolicy(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetChargingPolicy: %v\n", err)
	} else {
		fmt.Printf("Charging policy:  %d\n", policy)
	}
}
