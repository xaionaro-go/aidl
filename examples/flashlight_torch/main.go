// Toggle the Android flashlight/torch on and off via the camera service binder.
//
// Uses ICameraService.SetTorchMode to control the torch for camera "0".
// By default, turns the torch ON for 3 seconds, then turns it OFF.
// Pass "on" or "off" as a command-line argument to set a specific state.
//
// Note: the framework camera service (media.camera) requires the caller
// to hold android.permission.CAMERA. From the adb shell context this
// typically fails with a permission error. Run from an app context or
// with elevated privileges for full functionality.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/flashlight_torch ./examples/flashlight_torch/
//	adb push build/flashlight_torch /data/local/tmp/ && adb shell /data/local/tmp/flashlight_torch
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AndroidGoLab/binder/android/hardware"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// torchClientToken is a minimal TransactionReceiver used as the client
// binder token for SetTorchMode. The camera service requires a non-null
// binder to track torch ownership.
type torchClientToken struct{}

func (t *torchClientToken) Descriptor() string { return "torch.client" }

func (t *torchClientToken) OnTransaction(
	_ context.Context,
	_ binder.TransactionCode,
	_ *parcel.Parcel,
) (*parcel.Parcel, error) {
	return parcel.New(), nil
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

	svc, err := sm.GetService(ctx, servicemanager.MediaCameraService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get media.camera service: %v\n", err)
		os.Exit(1)
	}

	cam := hardware.NewCameraServiceProxy(svc)

	// Report the number of available cameras (informational; may fail
	// if the caller lacks android.permission.CAMERA).
	numCameras, err := cam.GetNumberOfCameras(ctx, hardware.ICameraServiceCameraTypeBackwardCompatible)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetNumberOfCameras: %v (continuing anyway)\n", err)
	} else {
		fmt.Printf("Number of cameras: %d\n", numCameras)
	}

	// Determine the desired action from command-line arguments.
	action := "toggle" // default: on, wait, off
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "on":
			action = "on"
		case "off":
			action = "off"
		default:
			fmt.Fprintf(os.Stderr, "usage: %s [on|off]\n", os.Args[0])
			os.Exit(1)
		}
	}

	// The camera service requires a non-null client binder token to
	// track torch ownership. Create a stub and register it with the
	// transport so it gets a valid cookie.
	clientToken := binder.NewStubBinder(&torchClientToken{})
	clientToken.RegisterWithTransport(ctx, transport)

	const cameraID = "0"

	switch action {
	case "on":
		setTorch(ctx, cam, cameraID, true, clientToken)

	case "off":
		setTorch(ctx, cam, cameraID, false, clientToken)

	default: // toggle
		setTorch(ctx, cam, cameraID, true, clientToken)
		fmt.Println("Waiting 3 seconds...")
		time.Sleep(3 * time.Second)
		setTorch(ctx, cam, cameraID, false, clientToken)
	}
}

func setTorch(
	ctx context.Context,
	cam *hardware.CameraServiceProxy,
	cameraID string,
	enabled bool,
	clientToken binder.IBinder,
) {
	state := "OFF"
	if enabled {
		state = "ON"
	}

	fmt.Printf("Turning torch %s for camera %s\n", state, cameraID)
	if err := cam.SetTorchMode(ctx, cameraID, enabled, clientToken); err != nil {
		fmt.Fprintf(os.Stderr, "SetTorchMode(%s): %v\n", state, err)
		fmt.Fprintf(os.Stderr, "Hint: SELinux denies shell→cameraserver binder calls.\n")
		fmt.Fprintf(os.Stderr, "Requires: adb root + setenforce 0 (permissive mode).\n")
		return
	}
	fmt.Printf("Torch is %s.\n", state)
}
