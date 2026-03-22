//go:build linux

// Binder proxy daemon — runs on an Android device, listens on TCP,
// and forwards binder transactions from host-side clients.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/binder-proxyd ./cmd/binder-proxyd/
//	adb push build/binder-proxyd /data/local/tmp/ && adb shell /data/local/tmp/binder-proxyd
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AndroidGoLab/binder/interop/gadb/proxy"
)

func main() {
	listenAddr := flag.String("listen", ":7100", "TCP address to listen on")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, *listenAddr); err != nil {
		fmt.Fprintf(os.Stderr, "binder-proxyd: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, listenAddr string) error {
	daemon, err := proxy.NewDaemon(ctx, proxy.DaemonOptionListenAddr(listenAddr))
	if err != nil {
		return fmt.Errorf("creating daemon: %w", err)
	}
	defer daemon.Close(ctx)

	return daemon.Serve(ctx)
}
