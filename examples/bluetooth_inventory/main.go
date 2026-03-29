// Enumerate paired/bonded Bluetooth devices and query adapter info.
//
// Demonstrates:
//   - IBluetoothManager: GetState
//   - RegisterAdapter -> IBluetooth: GetName, GetBondedDevices,
//     GetSupportedProfiles, GetRemoteName, GetRemoteAlias, GetBondState
//   - AttributionSource for authenticated calls
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/bluetooth_inventory ./examples/bluetooth_inventory/
//	adb push build/bluetooth_inventory /data/local/tmp/ && adb shell /data/local/tmp/bluetooth_inventory
package main

import (
	"context"
	"fmt"
	"os"

	genBluetooth "github.com/AndroidGoLab/binder/android/bluetooth"
	"github.com/AndroidGoLab/binder/android/content"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

type noopManagerCallback struct{}

func (noopManagerCallback) OnBluetoothServiceUp(context.Context, binder.IBinder) error { return nil }
func (noopManagerCallback) OnBluetoothServiceDown(context.Context) error               { return nil }
func (noopManagerCallback) OnBluetoothOn(context.Context) error                        { return nil }
func (noopManagerCallback) OnBluetoothOff(context.Context) error                       { return nil }

func shellAttribution() content.AttributionSource {
	return content.AttributionSource{
		AttributionSourceState: content.AttributionSourceState{
			Pid:         int32(os.Getpid()),
			Uid:         int32(os.Getuid()),
			PackageName: "com.android.shell",
		},
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
		fmt.Fprintf(os.Stderr, "transport: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close(ctx)

	sm := servicemanager.New(transport)

	btMgrSvc, err := sm.GetService(ctx, servicemanager.ServiceName("bluetooth_manager"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "bluetooth_manager: %v\n", err)
		os.Exit(1)
	}
	mgr := genBluetooth.NewBluetoothManagerProxy(btMgrSvc)

	state, _ := mgr.GetState(ctx)
	fmt.Printf("Bluetooth state: %d\n", state)

	callback := genBluetooth.NewBluetoothManagerCallbackStub(noopManagerCallback{})
	btAdapterBinder, err := mgr.RegisterAdapter(ctx, callback)
	if err != nil || btAdapterBinder == nil {
		fmt.Println("IBluetooth: not available (adapter may be off)")
		os.Exit(0)
	}
	btProxy := genBluetooth.NewBluetoothProxy(btAdapterBinder)

	attr := shellAttribution()

	// Adapter name.
	name, err := btProxy.GetName(ctx, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetName: %v\n", err)
	} else {
		fmt.Printf("Adapter name: %s\n", name)
	}

	// Supported profiles.
	profiles, err := btProxy.GetSupportedProfiles(ctx, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSupportedProfiles: %v\n", err)
	} else {
		fmt.Printf("Supported profiles: %v\n", profiles)
	}

	// Connection state.
	connState, err := btProxy.GetAdapterConnectionState(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetAdapterConnectionState: %v\n", err)
	} else {
		fmt.Printf("Adapter connection state: %d\n", connState)
	}

	// Bonded (paired) devices.
	devices, err := btProxy.GetBondedDevices(ctx, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetBondedDevices: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bonded devices: %d\n", len(devices))
	for i, dev := range devices {
		fmt.Printf("  [%d] AddressType=%d\n", i, dev.AddressType)

		// Query remote device info.
		remoteName, err := btProxy.GetRemoteName(ctx, dev, attr)
		if err == nil {
			fmt.Printf("       Name: %s\n", remoteName)
		}

		alias, err := btProxy.GetRemoteAlias(ctx, dev, attr)
		if err == nil {
			fmt.Printf("       Alias: %s\n", alias)
		}

		bondState, err := btProxy.GetBondState(ctx, dev, attr)
		if err == nil {
			fmt.Printf("       Bond state: %d\n", bondState)
		}
	}
}
