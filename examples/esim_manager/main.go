// Query eSIM/eUICC profile management: OTA status, supported countries,
// available memory.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/esim_manager ./examples/esim_manager/
//	adb push esim_manager /data/local/tmp/ && adb shell /data/local/tmp/esim_manager
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/com/android/internal_/telephony"
	"github.com/AndroidGoLab/binder/com/android/internal_/telephony/euicc"
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

	fmt.Println("=== eSIM/eUICC Manager ===")

	// First get the default eUICC card ID from telephony.
	phoneSvc, err := sm.GetService(ctx, servicemanager.TelephonyService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get phone service: %v\n", err)
		os.Exit(1)
	}

	phone := telephony.NewTelephonyProxy(phoneSvc)
	cardId, err := phone.GetCardIdForDefaultEuicc(ctx, 0)
	if err != nil {
		fmt.Printf("GetCardIdForDefaultEuicc: %v\n", err)
		cardId = 0
	} else {
		fmt.Printf("Default eUICC card ID: %d\n", cardId)
	}

	// Query the euicc controller service.
	euiccSvc, err := sm.CheckService(ctx, servicemanager.EuiccService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CheckService(euicc): %v\n", err)
		os.Exit(1)
	}
	if euiccSvc == nil {
		fmt.Println("euicc service: NOT REGISTERED (device may not support eSIM)")
		return
	}
	fmt.Printf("euicc service: FOUND (handle=%d)\n", euiccSvc.Handle())

	euiccProxy := euicc.NewEuiccControllerProxy(euiccSvc)

	// Query OTA status.
	otaStatus, err := euiccProxy.GetOtaStatus(ctx, cardId)
	if err != nil {
		fmt.Printf("GetOtaStatus: %v\n", err)
	} else {
		// 0=UNKNOWN, 1=READY, 2=IN_PROGRESS, 3=SUCCEEDED, 4=FAILED, 5=NOT_NEEDED
		statuses := map[int32]string{
			0: "UNKNOWN", 1: "READY", 2: "IN_PROGRESS",
			3: "SUCCEEDED", 4: "FAILED", 5: "NOT_NEEDED",
		}
		name := statuses[otaStatus]
		if name == "" {
			name = fmt.Sprintf("UNKNOWN(%d)", otaStatus)
		}
		fmt.Printf("OTA status: %s\n", name)
	}

	// Query supported countries.
	countries, err := euiccProxy.GetSupportedCountries(ctx, true)
	if err != nil {
		fmt.Printf("GetSupportedCountries: %v\n", err)
	} else {
		fmt.Printf("Supported countries: %d\n", len(countries))
		for i, c := range countries {
			if i >= 20 {
				fmt.Printf("  ... and %d more\n", len(countries)-20)
				break
			}
			fmt.Printf("  %s", c)
			if (i+1)%10 == 0 {
				fmt.Println()
			}
		}
		if len(countries)%10 != 0 && len(countries) <= 20 {
			fmt.Println()
		}
	}

	// Query eUICC info.
	info, err := euiccProxy.GetEuiccInfo(ctx, cardId)
	if err != nil {
		fmt.Printf("GetEuiccInfo: %v\n", err)
	} else {
		fmt.Printf("eUICC info: %+v\n", info)
	}
}
