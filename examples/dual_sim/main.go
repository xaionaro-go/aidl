// Monitor SIM slots: query active subscription count, slot info,
// and per-slot ICC card presence.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/dual_sim ./examples/dual_sim/
//	adb push dual_sim /data/local/tmp/ && adb shell /data/local/tmp/dual_sim
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

	fmt.Println("=== Dual SIM Monitor ===")

	// Query subscription service for slot count.
	subSvc, err := sm.GetService(ctx, servicemanager.ServiceName("isub"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get isub service: %v\n", err)
		os.Exit(1)
	}

	sub := telephony.NewSubProxy(subSvc)

	maxSlots, err := sub.GetActiveSubInfoCountMax(ctx)
	if err != nil {
		fmt.Printf("GetActiveSubInfoCountMax: %v\n", err)
	} else {
		fmt.Printf("Max SIM slots: %d\n", maxSlots)
	}

	activeCount, err := sub.GetActiveSubInfoCount(ctx, true)
	if err != nil {
		fmt.Printf("GetActiveSubInfoCount: %v\n", err)
	} else {
		fmt.Printf("Active subscriptions: %d\n", activeCount)
	}

	defaultSubId, err := sub.GetDefaultSubId(ctx)
	if err != nil {
		fmt.Printf("GetDefaultSubId: %v\n", err)
	} else {
		fmt.Printf("Default subscription ID: %d\n", defaultSubId)
	}

	activeSubIds, err := sub.GetActiveSubIdList(ctx, false)
	if err != nil {
		fmt.Printf("GetActiveSubIdList: %v\n", err)
	} else {
		fmt.Printf("Active subscription IDs: %v\n", activeSubIds)
	}

	// Query telephony service for per-slot info.
	phoneSvc, err := sm.GetService(ctx, servicemanager.TelephonyService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get phone service: %v\n", err)
		os.Exit(1)
	}

	phone := telephony.NewTelephonyProxy(phoneSvc)

	// Check ICC card presence per slot.
	slotCount := int32(2) // default to 2 slots
	if maxSlots > 0 {
		slotCount = maxSlots
	}
	fmt.Printf("\nPer-slot status:\n")
	for slot := int32(0); slot < slotCount; slot++ {
		hasIcc, err := phone.HasIccCardUsingSlotIndex(ctx, slot)
		if err != nil {
			fmt.Printf("  Slot %d: HasIccCard error: %v\n", slot, err)
			continue
		}
		phoneType, err := phone.GetActivePhoneTypeForSlot(ctx, slot)
		typeStr := "N/A"
		if err == nil {
			types := map[int32]string{0: "NONE", 1: "GSM", 2: "CDMA", 3: "SIP"}
			typeStr = types[phoneType]
			if typeStr == "" {
				typeStr = fmt.Sprintf("UNKNOWN(%d)", phoneType)
			}
		}
		fmt.Printf("  Slot %d: ICC=%v  PhoneType=%s\n", slot, hasIcc, typeStr)
	}

	// Multi-SIM support.
	multiSimResult, err := phone.IsMultiSimSupported(ctx)
	if err != nil {
		fmt.Printf("\nIsMultiSimSupported: %v\n", err)
	} else {
		// 0 = MULTISIM_ALLOWED, 1 = MULTISIM_NOT_SUPPORTED_BY_HARDWARE, ...
		statuses := map[int32]string{
			0: "ALLOWED",
			1: "NOT_SUPPORTED_BY_HARDWARE",
			2: "NOT_SUPPORTED_BY_CARRIER",
		}
		name := statuses[multiSimResult]
		if name == "" {
			name = fmt.Sprintf("UNKNOWN(%d)", multiSimResult)
		}
		fmt.Printf("\nMulti-SIM support: %s\n", name)
	}
}
