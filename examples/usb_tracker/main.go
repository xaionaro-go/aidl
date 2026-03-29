// Query USB device state: ports, functions, speed, and HAL versions.
//
// Uses the generated IUsbManager proxy via the "usb" binder service.
// Enumerates USB ports with their capabilities, queries current USB
// function mode and link speed.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/usb_tracker ./examples/usb_tracker/
//	adb push build/usb_tracker /data/local/tmp/ && adb shell /data/local/tmp/usb_tracker
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AndroidGoLab/binder/android/hardware/usb"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// USB function bitmask constants (from android.hardware.usb.UsbManager).
const (
	usbFuncMTP         int64 = 1 << 0  // MTP (Media Transfer Protocol)
	usbFuncPTP         int64 = 1 << 2  // PTP (Picture Transfer Protocol)
	usbFuncMIDI        int64 = 1 << 3  // MIDI
	usbFuncAccessory   int64 = 1 << 5  // Android Accessory
	usbFuncAudioSource int64 = 1 << 6  // Audio Source
	usbFuncNCM         int64 = 1 << 9  // NCM (Network Control Model)
	usbFuncUVC         int64 = 1 << 12 // UVC (USB Video Class)
)

// USB port mode constants.
const (
	portModeNone   int32 = 0
	portModeSrc    int32 = 1 // USB-C host (source)
	portModeSink   int32 = 2 // USB-C device (sink)
	portModeDRP    int32 = 3 // Dual-role port
)

func usbFunctionNames(funcs int64) string {
	type funcEntry struct {
		mask int64
		name string
	}
	entries := []funcEntry{
		{usbFuncMTP, "MTP"},
		{usbFuncPTP, "PTP"},
		{usbFuncMIDI, "MIDI"},
		{usbFuncAccessory, "ACCESSORY"},
		{usbFuncAudioSource, "AUDIO_SOURCE"},
		{usbFuncNCM, "NCM"},
		{usbFuncUVC, "UVC"},
	}

	var names []string
	for _, e := range entries {
		if funcs&e.mask != 0 {
			names = append(names, e.name)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func portModeString(mode int32) string {
	switch mode {
	case portModeNone:
		return "none"
	case portModeSrc:
		return "source (host)"
	case portModeSink:
		return "sink (device)"
	case portModeDRP:
		return "dual-role"
	default:
		return fmt.Sprintf("unknown(%d)", mode)
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

	usbMgr, err := usb.GetUsbManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get usb service: %v\n", err)
		os.Exit(1)
	}

	// Current USB functions
	funcs, err := usbMgr.GetCurrentFunctions(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCurrentFunctions: %v\n", err)
	} else {
		fmt.Printf("USB functions:     0x%x (%s)\n", funcs, usbFunctionNames(funcs))
	}

	// USB link speed
	speed, err := usbMgr.GetCurrentUsbSpeed(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetCurrentUsbSpeed: %v\n", err)
	} else {
		fmt.Printf("USB speed:         %d\n", speed)
	}

	// Gadget HAL version
	gadgetVer, err := usbMgr.GetGadgetHalVersion(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetGadgetHalVersion: %v\n", err)
	} else {
		fmt.Printf("Gadget HAL:        v%d\n", gadgetVer)
	}

	// USB HAL version
	usbHalVer, err := usbMgr.GetUsbHalVersion(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetUsbHalVersion: %v\n", err)
	} else {
		fmt.Printf("USB HAL:           v%d\n", usbHalVer)
	}

	// USB ports
	ports, err := usbMgr.GetPorts(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetPorts: %v\n", err)
	} else {
		fmt.Printf("\nUSB ports: %d\n", len(ports))
		for _, p := range ports {
			fmt.Printf("  Port %q:\n", p.Id)
			fmt.Printf("    Supported modes:       %s\n", portModeString(p.SupportedModes))
			fmt.Printf("    Contamination protect: modes=0x%x\n", p.SupportedContaminantProtectionModes)
			fmt.Printf("    Presence protection:   %v\n", p.SupportsEnableContaminantPresenceProtection)
			fmt.Printf("    Presence detection:    %v\n", p.SupportsEnableContaminantPresenceDetection)
			fmt.Printf("    Compliance warnings:   %v\n", p.SupportsComplianceWarnings)
			fmt.Printf("    Alt modes mask:        0x%x\n", p.SupportedAltModesMask)
		}
	}

	// Screen-unlocked functions
	screenFuncs, err := usbMgr.GetScreenUnlockedFunctions(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetScreenUnlockedFunctions: %v\n", err)
	} else {
		fmt.Printf("\nScreen-unlocked:   0x%x (%s)\n", screenFuncs, usbFunctionNames(screenFuncs))
	}
}
