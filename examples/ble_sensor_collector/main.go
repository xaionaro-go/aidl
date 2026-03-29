// BLE sensor collector: scan for BLE devices and register a GATT client.
//
// Demonstrates:
//   - IBluetoothManager -> RegisterAdapter -> IBluetooth
//   - GetBluetoothGatt -> IBluetoothGatt: GATT client registration
//   - GetBluetoothScan -> IBluetoothScan: BLE scanning with callbacks
//   - IBluetoothGattCallback: OnClientRegistered
//   - IScannerCallback: OnScannerRegistered, OnScanResult
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/ble_sensor_collector ./examples/ble_sensor_collector/
//	adb push build/ble_sensor_collector /data/local/tmp/ && adb shell /data/local/tmp/ble_sensor_collector
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	genBluetooth "github.com/AndroidGoLab/binder/android/bluetooth"
	genLE "github.com/AndroidGoLab/binder/android/bluetooth/le"
	"github.com/AndroidGoLab/binder/android/content"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

type noopManagerCallback struct{}

func (noopManagerCallback) OnBluetoothServiceUp(context.Context, binder.IBinder) error { return nil }
func (noopManagerCallback) OnBluetoothServiceDown(context.Context) error               { return nil }
func (noopManagerCallback) OnBluetoothOn(context.Context) error                        { return nil }
func (noopManagerCallback) OnBluetoothOff(context.Context) error                       { return nil }

type gattSpy struct {
	registeredCh chan int32
}

func (g *gattSpy) OnClientRegistered(_ context.Context, status int32) error {
	select {
	case g.registeredCh <- status:
	default:
	}
	return nil
}
func (g *gattSpy) OnClientConnectionState(context.Context, int32, bool, genBluetooth.BluetoothDevice) error {
	return nil
}
func (g *gattSpy) OnPhyUpdate(context.Context, genBluetooth.BluetoothDevice, int32, int32, int32) error {
	return nil
}
func (g *gattSpy) OnPhyRead(context.Context, genBluetooth.BluetoothDevice, int32, int32, int32) error {
	return nil
}
func (g *gattSpy) OnSearchComplete(context.Context, genBluetooth.BluetoothDevice, []genBluetooth.BluetoothGattService, int32) error {
	return nil
}
func (g *gattSpy) OnCharacteristicRead(context.Context, genBluetooth.BluetoothDevice, int32, int32, []byte) error {
	return nil
}
func (g *gattSpy) OnCharacteristicWrite(context.Context, genBluetooth.BluetoothDevice, int32, int32, []byte) error {
	return nil
}
func (g *gattSpy) OnExecuteWrite(context.Context, genBluetooth.BluetoothDevice, int32) error {
	return nil
}
func (g *gattSpy) OnDescriptorRead(context.Context, genBluetooth.BluetoothDevice, int32, int32, []byte) error {
	return nil
}
func (g *gattSpy) OnDescriptorWrite(context.Context, genBluetooth.BluetoothDevice, int32, int32, []byte) error {
	return nil
}
func (g *gattSpy) OnNotify(context.Context, genBluetooth.BluetoothDevice, int32, []byte) error {
	return nil
}
func (g *gattSpy) OnReadRemoteRssi(context.Context, genBluetooth.BluetoothDevice, int32, int32) error {
	return nil
}
func (g *gattSpy) OnConfigureMTU(context.Context, genBluetooth.BluetoothDevice, int32, int32) error {
	return nil
}
func (g *gattSpy) OnConnectionUpdated(context.Context, genBluetooth.BluetoothDevice, int32, int32, int32, int32) error {
	return nil
}
func (g *gattSpy) OnServiceChanged(context.Context, genBluetooth.BluetoothDevice) error {
	return nil
}
func (g *gattSpy) OnSubrateChange(context.Context, genBluetooth.BluetoothDevice, int32, int32) error {
	return nil
}

type scanSpy struct {
	registeredCh chan int32
	results      chan genLE.ScanResult
}

func (s *scanSpy) OnScannerRegistered(_ context.Context, status, scannerID int32) error {
	if status != 0 {
		fmt.Fprintf(os.Stderr, "scanner registration failed: status=%d\n", status)
	}
	select {
	case s.registeredCh <- scannerID:
	default:
	}
	return nil
}
func (s *scanSpy) OnScanResult(_ context.Context, result genLE.ScanResult) error {
	select {
	case s.results <- result:
	default:
	}
	return nil
}
func (s *scanSpy) OnBatchScanResults(context.Context, []genLE.ScanResult) error { return nil }
func (s *scanSpy) OnFoundOrLost(context.Context, bool, genLE.ScanResult) error  { return nil }
func (s *scanSpy) OnScanManagerErrorCallback(context.Context, int32) error       { return nil }

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

	mgrCallback := genBluetooth.NewBluetoothManagerCallbackStub(noopManagerCallback{})
	btAdapterBinder, err := mgr.RegisterAdapter(ctx, mgrCallback)
	if err != nil || btAdapterBinder == nil {
		fmt.Println("IBluetooth: not available")
		os.Exit(0)
	}
	btProxy := genBluetooth.NewBluetoothProxy(btAdapterBinder)
	fmt.Printf("IBluetooth: handle %d\n", btAdapterBinder.Handle())

	// Register GATT client for sensor data reads.
	gattBinder, err := btProxy.GetBluetoothGatt(ctx)
	if err != nil || gattBinder == nil {
		fmt.Println("IBluetoothGatt: not available")
		os.Exit(0)
	}
	fmt.Printf("IBluetoothGatt: handle %d\n", gattBinder.Handle())

	spy := &gattSpy{registeredCh: make(chan int32, 1)}
	gattCallback := genBluetooth.NewBluetoothGattCallbackStub(spy)

	// registerClient via raw transaction (the generated proxy may not
	// handle all the complex parameters correctly).
	code, err := gattBinder.ResolveCode(ctx, genBluetooth.DescriptorIBluetoothGatt, "registerClient")
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve registerClient: %v\n", err)
		os.Exit(1)
	}

	data := parcel.New()
	data.WriteInterfaceToken(genBluetooth.DescriptorIBluetoothGatt)
	data.WriteInt32(1) // non-null ParcelUuid
	data.WriteInt64(0x0000180000001000)
	data.WriteInt64(-9223371485494954757)
	binder.WriteBinderToParcel(ctx, data, gattCallback.AsBinder(), transport)
	data.WriteBool(false)
	data.WriteInt32(0)
	attr := shellAttribution()
	data.WriteInt32(1)
	attr.MarshalParcel(data)

	reply, err := gattBinder.Transact(ctx, code, 0, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registerClient: %v\n", err)
		os.Exit(1)
	}
	if statusErr := binder.ReadStatus(reply); statusErr != nil {
		fmt.Fprintf(os.Stderr, "registerClient status: %v\n", statusErr)
		fmt.Fprintf(os.Stderr, "Hint: AIDL version mismatch — the BT stack may not expect AttributionSource.\n")
		return
	}

	select {
	case status := <-spy.registeredCh:
		fmt.Printf("GATT client registered: status=%d\n", status)
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "GATT client registration timed out")
		os.Exit(1)
	}

	// BLE scan for nearby sensors.
	scanBinder, err := btProxy.GetBluetoothScan(ctx)
	if err != nil || scanBinder == nil {
		fmt.Println("IBluetoothScan: not available")
		os.Exit(0)
	}
	scanProxy := genBluetooth.NewBluetoothScanProxy(scanBinder)
	fmt.Printf("IBluetoothScan: handle %d\n", scanBinder.Handle())

	scanSpy := &scanSpy{
		registeredCh: make(chan int32, 1),
		results:      make(chan genLE.ScanResult, 100),
	}
	scanCallback := genLE.NewScannerCallbackStub(scanSpy)

	if err := scanProxy.RegisterScanner(ctx, scanCallback, genOs.WorkSource{}, shellAttribution()); err != nil {
		fmt.Fprintf(os.Stderr, "registerScanner: %v\n", err)
		os.Exit(1)
	}

	var scannerID int32
	select {
	case scannerID = <-scanSpy.registeredCh:
		fmt.Printf("Scanner registered: id=%d\n", scannerID)
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "scanner registration timed out")
		os.Exit(1)
	}

	ss := genLE.ScanSettings{
		CallbackType:          1,
		MatchMode:             1,
		NumOfMatchesPerFilter: 3,
		Phy:                   255,
	}
	if err := scanProxy.StartScan(ctx, scannerID, ss, nil, shellAttribution()); err != nil {
		fmt.Fprintf(os.Stderr, "startScan: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Scanning for BLE sensors (5s)...")
	deadline := time.After(5 * time.Second)
	count := 0
loop:
	for {
		select {
		case result := <-scanSpy.results:
			count++
			fmt.Printf("  Device RSSI=%d dBm\n", result.Rssi)
		case <-deadline:
			break loop
		}
	}
	fmt.Printf("Found %d BLE devices in 5s\n", count)

	if err := scanProxy.StopScan(ctx, scannerID, shellAttribution()); err != nil {
		fmt.Fprintf(os.Stderr, "stopScan: %v\n", err)
	}
}
