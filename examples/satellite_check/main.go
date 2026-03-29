// Check satellite telephony readiness by querying the telephony service.
//
// Queries the ITelephony proxy for radio state, data connectivity,
// network type, SIM presence, and satellite PLMNs. These are the
// prerequisite checks before satellite telephony can be used.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/satellite_check ./examples/satellite_check/
//	adb push satellite_check /data/local/tmp/ && adb shell /data/local/tmp/satellite_check
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/com/android/internal_/telephony"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func dataStateName(state int32) string {
	switch state {
	case 0:
		return "disconnected"
	case 1:
		return "connecting"
	case 2:
		return "connected"
	case 3:
		return "suspended"
	case 4:
		return "disconnecting"
	default:
		return fmt.Sprintf("unknown(%d)", state)
	}
}

func phoneTypeName(t int32) string {
	switch t {
	case 0:
		return "none"
	case 1:
		return "GSM"
	case 2:
		return "CDMA"
	case 3:
		return "SIP"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

func callStateName(state int32) string {
	switch state {
	case 0:
		return "idle"
	case 1:
		return "ringing"
	case 2:
		return "offhook"
	default:
		return fmt.Sprintf("unknown(%d)", state)
	}
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

	fmt.Println("=== Satellite Telephony Readiness ===")
	fmt.Println()

	// Get the telephony service.
	svc, err := sm.GetService(ctx, servicemanager.TelephonyService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get telephony service: %v\n", err)
		os.Exit(1)
	}

	tel := telephony.NewTelephonyProxy(svc)

	// Radio state — satellite requires an active radio.
	radioOn, err := tel.IsRadioOn(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsRadioOn: %v\n", err)
	} else {
		fmt.Printf("Radio on:       %v\n", radioOn)
	}

	// Call state
	callState, err := tel.GetCallState(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCallState: %v\n", err)
	} else {
		fmt.Printf("Call state:     %s\n", callStateName(callState))
	}

	// Data state
	dataState, err := tel.GetDataState(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetDataState: %v\n", err)
	} else {
		fmt.Printf("Data state:     %s\n", dataStateName(dataState))
	}

	// Active phone type
	phoneType, err := tel.GetActivePhoneType(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetActivePhoneType: %v\n", err)
	} else {
		fmt.Printf("Phone type:     %s\n", phoneTypeName(phoneType))
	}

	// SIM card presence
	hasIcc, err := tel.HasIccCard(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HasIccCard: %v\n", err)
	} else {
		fmt.Printf("SIM present:    %v\n", hasIcc)
	}

	// Network country
	country, err := tel.GetNetworkCountryIsoForPhone(ctx, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetNetworkCountryIsoForPhone: %v\n", err)
	} else {
		if country == "" {
			country = "(none)"
		}
		fmt.Printf("Network country: %s\n", country)
	}

	// Satellite PLMNs for default subscription
	fmt.Println()
	fmt.Println("Satellite PLMNs (sub 0):")
	plmns, err := tel.GetSatellitePlmnsForCarrier(ctx, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  GetSatellitePlmnsForCarrier: %v\n", err)
	} else if len(plmns) == 0 {
		fmt.Println("  (none — satellite not provisioned for this carrier)")
	} else {
		for _, p := range plmns {
			fmt.Printf("  %s\n", p)
		}
	}

	// Check satellite service registration
	fmt.Println()
	satSvc, err := sm.CheckService(ctx, servicemanager.SatelliteService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CheckService(satellite): %v\n", err)
	} else if satSvc == nil {
		fmt.Println("Satellite service: NOT REGISTERED (device may not support satellite telephony)")
	} else {
		fmt.Printf("Satellite service: FOUND (handle=%d, alive=%v)\n",
			satSvc.Handle(), satSvc.IsAlive(ctx))
	}
}
