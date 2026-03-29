// Query telephony service for SIM state: radio, ICC card, data state.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/sim_status ./examples/sim_status/
//	adb push sim_status /data/local/tmp/ && adb shell /data/local/tmp/sim_status
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

	svc, err := sm.GetService(ctx, servicemanager.TelephonyService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get phone service: %v\n", err)
		os.Exit(1)
	}

	phone := telephony.NewTelephonyProxy(svc)

	fmt.Println("=== SIM Status ===")

	// Check if ICC card is present.
	hasIcc, err := phone.HasIccCard(ctx)
	if err != nil {
		fmt.Printf("HasIccCard: %v\n", err)
	} else {
		fmt.Printf("ICC card present: %v\n", hasIcc)
	}

	// Radio status.
	radioOn, err := phone.IsRadioOn(ctx)
	if err != nil {
		fmt.Printf("IsRadioOn: %v\n", err)
	} else {
		fmt.Printf("Radio on: %v\n", radioOn)
	}

	// Call state (0=IDLE, 1=RINGING, 2=OFFHOOK).
	callState, err := phone.GetCallState(ctx)
	if err != nil {
		fmt.Printf("GetCallState: %v\n", err)
	} else {
		states := map[int32]string{0: "IDLE", 1: "RINGING", 2: "OFFHOOK"}
		name := states[callState]
		if name == "" {
			name = fmt.Sprintf("UNKNOWN(%d)", callState)
		}
		fmt.Printf("Call state: %s\n", name)
	}

	// Data state (0=DISCONNECTED, 1=CONNECTING, 2=CONNECTED, 3=SUSPENDED).
	dataState, err := phone.GetDataState(ctx)
	if err != nil {
		fmt.Printf("GetDataState: %v\n", err)
	} else {
		states := map[int32]string{
			0: "DISCONNECTED", 1: "CONNECTING",
			2: "CONNECTED", 3: "SUSPENDED",
		}
		name := states[dataState]
		if name == "" {
			name = fmt.Sprintf("UNKNOWN(%d)", dataState)
		}
		fmt.Printf("Data state: %s\n", name)
	}

	// Active phone type (0=NONE, 1=GSM, 2=CDMA, 3=SIP).
	phoneType, err := phone.GetActivePhoneType(ctx)
	if err != nil {
		fmt.Printf("GetActivePhoneType: %v\n", err)
	} else {
		types := map[int32]string{0: "NONE", 1: "GSM", 2: "CDMA", 3: "SIP"}
		name := types[phoneType]
		if name == "" {
			name = fmt.Sprintf("UNKNOWN(%d)", phoneType)
		}
		fmt.Printf("Phone type: %s\n", name)
	}

	// Network country.
	country, err := phone.GetNetworkCountryIsoForPhone(ctx, 0)
	if err != nil {
		fmt.Printf("GetNetworkCountryIsoForPhone: %v\n", err)
	} else {
		fmt.Printf("Network country (slot 0): %q\n", country)
	}
}
