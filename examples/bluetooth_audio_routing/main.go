// Manage Bluetooth A2DP audio connections via binder.
//
// Demonstrates:
//   - IBluetoothManager: GetState, RegisterAdapter
//   - IBluetooth: GetProfile (A2DP profile ID = 2)
//   - IBluetoothA2dp: GetConnectedDevices, GetActiveDevice,
//     GetSupportedCodecTypes, GetConnectionState
//   - AttributionSource for authenticated calls
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/bluetooth_audio_routing ./examples/bluetooth_audio_routing/
//	adb push build/bluetooth_audio_routing /data/local/tmp/ && adb shell /data/local/tmp/bluetooth_audio_routing
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

// A2DP profile ID in Android BluetoothProfile constants.
const profileA2DP = 2

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
	fmt.Printf("IBluetooth: handle %d\n", btAdapterBinder.Handle())

	attr := shellAttribution()

	// Check A2DP profile connection state at the adapter level.
	a2dpState, err := btProxy.GetProfileConnectionState(ctx, profileA2DP, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetProfileConnectionState(A2DP): %v\n", err)
	} else {
		fmt.Printf("A2DP profile connection state: %d\n", a2dpState)
	}

	// Get A2DP profile binder.
	a2dpBinder, err := btProxy.GetProfile(ctx, profileA2DP)
	if err != nil || a2dpBinder == nil {
		fmt.Println("IBluetoothA2dp: not available")
		os.Exit(0)
	}
	a2dp := genBluetooth.NewBluetoothA2dpProxy(a2dpBinder)
	fmt.Printf("IBluetoothA2dp: handle %d\n", a2dpBinder.Handle())

	// Query supported codec types.
	codecs, err := a2dp.GetSupportedCodecTypes(ctx, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSupportedCodecTypes: %v\n", err)
	} else {
		fmt.Printf("Supported A2DP codec types: %d\n", len(codecs))
		for i, c := range codecs {
			fmt.Printf("  [%d] CodecType{%+v}\n", i, c)
		}
	}

	// List connected A2DP devices.
	connDevices, err := a2dp.GetConnectedDevices(ctx, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetConnectedDevices: %v\n", err)
	} else {
		fmt.Printf("Connected A2DP devices: %d\n", len(connDevices))
		for i, dev := range connDevices {
			fmt.Printf("  [%d] AddressType=%d\n", i, dev.AddressType)

			connState, err := a2dp.GetConnectionState(ctx, dev, attr)
			if err == nil {
				fmt.Printf("       ConnectionState=%d\n", connState)
			}

			playing, err := a2dp.IsA2dpPlaying(ctx, dev, attr)
			if err == nil {
				fmt.Printf("       IsPlaying=%v\n", playing)
			}
		}
	}

	// Get active A2DP device.
	activeDevice, err := a2dp.GetActiveDevice(ctx, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetActiveDevice: %v\n", err)
	} else {
		fmt.Printf("Active A2DP device: AddressType=%d\n", activeDevice.AddressType)
	}

	// Max connected audio devices.
	maxDevices, err := btProxy.GetMaxConnectedAudioDevices(ctx, attr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetMaxConnectedAudioDevices: %v\n", err)
	} else {
		fmt.Printf("Max connected audio devices: %d\n", maxDevices)
	}
}
