// Register a Go service with the ServiceManager and call it back.
//
// Demonstrates implementing binder.TransactionReceiver — the server-side
// binder interface — and registering it via AddService. When running in
// a restricted SELinux context (e.g. shell), AddService will be denied;
// in that case the example falls back to an in-process self-test that
// exercises the same OnTransaction code path.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/server_service ./examples/server_service/
//	adb push build/server_service /data/local/tmp/ && adb shell /data/local/tmp/server_service
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	aidlerrors "github.com/AndroidGoLab/binder/errors"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	pingServiceDescriptor = "com.example.IPingService"
	pingServiceName       = "com.example.ping"

	codePing = binder.FirstCallTransaction + 0
	codeEcho = binder.FirstCallTransaction + 1
)

// pingService implements binder.TransactionReceiver for a simple
// ping/echo service.
type pingService struct{}

func (s *pingService) Descriptor() string {
	return pingServiceDescriptor
}

func (s *pingService) OnTransaction(
	ctx context.Context,
	code binder.TransactionCode,
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	if _, err := data.ReadInterfaceToken(); err != nil {
		return nil, err
	}

	switch code {
	case codePing:
		reply := parcel.New()
		binder.WriteStatus(reply, nil)
		reply.WriteString16("pong")
		return reply, nil

	case codeEcho:
		msg, err := data.ReadString16()
		if err != nil {
			reply := parcel.New()
			binder.WriteStatus(reply, fmt.Errorf("reading echo argument: %w", err))
			return reply, nil
		}

		reply := parcel.New()
		binder.WriteStatus(reply, nil)
		reply.WriteString16(msg)
		return reply, nil

	default:
		return nil, fmt.Errorf("unknown transaction code %d", code)
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

	// Register our ping service with the service manager.
	svc := &pingService{}
	err = sm.AddService(ctx, servicemanager.ServiceName(pingServiceName), svc, false, 0)

	switch {
	case err == nil:
		fmt.Printf("Registered service %q\n", pingServiceName)
		remoteTest(ctx, sm)

	default:
		// AddService typically fails from the shell SELinux context.
		// Show the error and fall back to an in-process self-test.
		var se *aidlerrors.StatusError
		if errors.As(err, &se) && se.Exception == aidlerrors.ExceptionSecurity {
			fmt.Printf("AddService denied by SELinux (expected from shell context).\n")
			fmt.Printf("Falling back to in-process self-test.\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "add service: %v\n", err)
			fmt.Fprintf(os.Stderr, "Falling back to in-process self-test.\n\n")
		}
		inProcessTest(ctx, svc)
	}
}

// remoteTest looks up the service via the ServiceManager and calls it
// through the binder driver, exercising the full IPC path.
func remoteTest(ctx context.Context, sm *servicemanager.ServiceManager) {
	remote, err := sm.GetService(ctx, servicemanager.ServiceName(pingServiceName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get service: %v\n", err)
		os.Exit(1)
	}

	callPing(ctx, remote)
	callEcho(ctx, remote, "hello from Go")
	fmt.Println("All remote self-tests passed.")
}

// inProcessTest calls OnTransaction directly, verifying the handler
// logic without requiring ServiceManager registration.
func inProcessTest(ctx context.Context, svc *pingService) {
	// Ping
	{
		data := parcel.New()
		defer data.Recycle()
		data.WriteInterfaceToken(pingServiceDescriptor)

		reply, err := svc.OnTransaction(ctx, codePing, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "in-process ping: %v\n", err)
			os.Exit(1)
		}
		defer reply.Recycle()

		if err := binder.ReadStatus(reply); err != nil {
			fmt.Fprintf(os.Stderr, "in-process ping status: %v\n", err)
			os.Exit(1)
		}

		result, err := reply.ReadString16()
		if err != nil {
			fmt.Fprintf(os.Stderr, "in-process ping read: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Ping -> %q\n", result)
	}

	// Echo
	{
		data := parcel.New()
		defer data.Recycle()
		data.WriteInterfaceToken(pingServiceDescriptor)
		data.WriteString16("hello from Go")

		reply, err := svc.OnTransaction(ctx, codeEcho, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "in-process echo: %v\n", err)
			os.Exit(1)
		}
		defer reply.Recycle()

		if err := binder.ReadStatus(reply); err != nil {
			fmt.Fprintf(os.Stderr, "in-process echo status: %v\n", err)
			os.Exit(1)
		}

		result, err := reply.ReadString16()
		if err != nil {
			fmt.Fprintf(os.Stderr, "in-process echo read: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Echo -> %q\n", result)
	}

	fmt.Println("All in-process self-tests passed.")
}

func callPing(ctx context.Context, remote binder.IBinder) {
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(pingServiceDescriptor)

	reply, err := remote.Transact(ctx, codePing, 0, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ping transact: %v\n", err)
		os.Exit(1)
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		fmt.Fprintf(os.Stderr, "ping status: %v\n", err)
		os.Exit(1)
	}

	result, err := reply.ReadString16()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ping read result: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Ping -> %q\n", result)
}

func callEcho(ctx context.Context, remote binder.IBinder, msg string) {
	data := parcel.New()
	defer data.Recycle()
	data.WriteInterfaceToken(pingServiceDescriptor)
	data.WriteString16(msg)

	reply, err := remote.Transact(ctx, codeEcho, 0, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "echo transact: %v\n", err)
		os.Exit(1)
	}
	defer reply.Recycle()

	if err := binder.ReadStatus(reply); err != nil {
		fmt.Fprintf(os.Stderr, "echo status: %v\n", err)
		os.Exit(1)
	}

	result, err := reply.ReadString16()
	if err != nil {
		fmt.Fprintf(os.Stderr, "echo read result: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Echo -> %q\n", result)
}
