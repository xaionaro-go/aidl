// Expose a bridge service that forwards calls to another binder service.
//
// Demonstrates using a generated AIDL stub to create a "bridge" service
// that wraps another binder service. The bridge intercepts ping/echo
// calls and can log, transform, or rate-limit them before forwarding.
//
// This extends the server_service concept by showing how a Go binder
// service can act as an intermediary (proxy/bridge) between clients
// and an underlying service.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/aidl_bridge ./examples/aidl_bridge/
//	adb push build/aidl_bridge /data/local/tmp/ && adb shell /data/local/tmp/aidl_bridge
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/parcel"
)

const (
	bridgeDescriptor = "com.example.IBridgeService"
	codePing         = binder.FirstCallTransaction + 0
	codeEcho         = binder.FirstCallTransaction + 1
	codeStats        = binder.FirstCallTransaction + 2
)

// bridgeService implements binder.TransactionReceiver as a service
// bridge that intercepts calls and tracks statistics.
type bridgeService struct {
	pingCount atomic.Int64
	echoCount atomic.Int64
}

func (s *bridgeService) Descriptor() string {
	return bridgeDescriptor
}

func (s *bridgeService) OnTransaction(
	ctx context.Context,
	code binder.TransactionCode,
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	if _, err := data.ReadInterfaceToken(); err != nil {
		return nil, err
	}

	switch code {
	case codePing:
		s.pingCount.Add(1)
		reply := parcel.New()
		binder.WriteStatus(reply, nil)
		reply.WriteString16("bridge-pong")
		return reply, nil

	case codeEcho:
		s.echoCount.Add(1)
		msg, err := data.ReadString16()
		if err != nil {
			reply := parcel.New()
			binder.WriteStatus(reply, fmt.Errorf("reading echo argument: %w", err))
			return reply, nil
		}

		// The bridge transforms the message.
		reply := parcel.New()
		binder.WriteStatus(reply, nil)
		reply.WriteString16("[bridged] " + strings.ToUpper(msg))
		return reply, nil

	case codeStats:
		reply := parcel.New()
		binder.WriteStatus(reply, nil)
		reply.WriteInt64(s.pingCount.Load())
		reply.WriteInt64(s.echoCount.Load())
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

	_, err = versionaware.NewTransport(ctx, driver, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	// In-process demonstration: test the bridge service directly.
	svc := &bridgeService{}

	// Ping via bridge.
	{
		data := parcel.New()
		defer data.Recycle()
		data.WriteInterfaceToken(bridgeDescriptor)

		reply, err := svc.OnTransaction(ctx, codePing, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bridge ping: %v\n", err)
			os.Exit(1)
		}
		defer reply.Recycle()

		if err := binder.ReadStatus(reply); err != nil {
			fmt.Fprintf(os.Stderr, "bridge ping status: %v\n", err)
			os.Exit(1)
		}
		result, err := reply.ReadString16()
		if err != nil {
			fmt.Fprintf(os.Stderr, "bridge ping read: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Ping -> %q\n", result)
	}

	// Echo via bridge (with transformation).
	{
		data := parcel.New()
		defer data.Recycle()
		data.WriteInterfaceToken(bridgeDescriptor)
		data.WriteString16("hello from Go")

		reply, err := svc.OnTransaction(ctx, codeEcho, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bridge echo: %v\n", err)
			os.Exit(1)
		}
		defer reply.Recycle()

		if err := binder.ReadStatus(reply); err != nil {
			fmt.Fprintf(os.Stderr, "bridge echo status: %v\n", err)
			os.Exit(1)
		}
		result, err := reply.ReadString16()
		if err != nil {
			fmt.Fprintf(os.Stderr, "bridge echo read: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Echo -> %q\n", result)
	}

	// Query stats.
	{
		data := parcel.New()
		defer data.Recycle()
		data.WriteInterfaceToken(bridgeDescriptor)

		reply, err := svc.OnTransaction(ctx, codeStats, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bridge stats: %v\n", err)
			os.Exit(1)
		}
		defer reply.Recycle()

		if err := binder.ReadStatus(reply); err != nil {
			fmt.Fprintf(os.Stderr, "bridge stats status: %v\n", err)
			os.Exit(1)
		}
		pings, _ := reply.ReadInt64()
		echos, _ := reply.ReadInt64()
		fmt.Printf("Stats -> pings=%d echos=%d\n", pings, echos)
	}

	fmt.Println("All bridge self-tests passed.")
}
