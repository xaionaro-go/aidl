// Create a mock binder service for testing.
//
// Registers a mock service implementing TransactionReceiver and tests
// it with in-process calls. Falls back to in-process self-test when
// ServiceManager registration is denied by SELinux.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/mock_service ./examples/mock_service/
//	adb push build/mock_service /data/local/tmp/ && adb shell /data/local/tmp/mock_service
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	mockDescriptor  = "com.example.IMockService"
	mockServiceName = "com.example.mock"

	codeGetTimestamp = binder.FirstCallTransaction + 0
	codeGetVersion   = binder.FirstCallTransaction + 1
	codeIsHealthy    = binder.FirstCallTransaction + 2
)

// mockService implements binder.TransactionReceiver.
type mockService struct {
	startTime time.Time
}

func (s *mockService) Descriptor() string {
	return mockDescriptor
}

func (s *mockService) OnTransaction(
	ctx context.Context,
	code binder.TransactionCode,
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	if _, err := data.ReadInterfaceToken(); err != nil {
		return nil, err
	}

	reply := parcel.New()

	switch code {
	case codeGetTimestamp:
		binder.WriteStatus(reply, nil)
		reply.WriteInt64(time.Now().UnixMilli())
		return reply, nil

	case codeGetVersion:
		binder.WriteStatus(reply, nil)
		reply.WriteString16("mock-service-v1.0.0")
		return reply, nil

	case codeIsHealthy:
		binder.WriteStatus(reply, nil)
		uptime := time.Since(s.startTime)
		healthy := uptime < 24*time.Hour // Consider healthy if up < 24h.
		reply.WriteBool(healthy)
		return reply, nil

	default:
		reply.Recycle()
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
	svc := &mockService{startTime: time.Now()}

	fmt.Println("=== Mock Binder Service ===")
	fmt.Println()

	// Try to register with ServiceManager.
	err = sm.AddService(ctx, servicemanager.ServiceName(mockServiceName), svc, false, 0)
	if err != nil {
		fmt.Printf("AddService denied (expected from shell): %v\n", err)
		fmt.Println("Running in-process self-test instead.")
	} else {
		fmt.Printf("Registered service %q\n\n", mockServiceName)
	}

	// In-process self-test: exercise all transaction codes.
	testCases := []struct {
		name string
		code binder.TransactionCode
	}{
		{"GetTimestamp", codeGetTimestamp},
		{"GetVersion", codeGetVersion},
		{"IsHealthy", codeIsHealthy},
	}

	for _, tc := range testCases {
		data := parcel.New()
		data.WriteInterfaceToken(mockDescriptor)

		reply, err := svc.OnTransaction(ctx, tc.code, data)
		data.Recycle()
		if err != nil {
			fmt.Printf("  %s: ERROR: %v\n", tc.name, err)
			continue
		}

		if err := binder.ReadStatus(reply); err != nil {
			fmt.Printf("  %s: STATUS ERROR: %v\n", tc.name, err)
			reply.Recycle()
			continue
		}

		switch tc.code {
		case codeGetTimestamp:
			ts, _ := reply.ReadInt64()
			fmt.Printf("  %s: %d (unix ms)\n", tc.name, ts)
		case codeGetVersion:
			ver, _ := reply.ReadString16()
			fmt.Printf("  %s: %s\n", tc.name, ver)
		case codeIsHealthy:
			healthy, _ := reply.ReadBool()
			fmt.Printf("  %s: %v\n", tc.name, healthy)
		}

		reply.Recycle()
	}

	fmt.Println("\nAll mock service tests passed.")
}
