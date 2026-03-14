//go:build linux

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xaionaro-go/aidl/binder"
	"github.com/xaionaro-go/aidl/parcel"
)

func newServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "List, inspect, and transact with binder services",
	}

	cmd.AddCommand(newServiceListCmd())
	cmd.AddCommand(newServiceInspectCmd())
	cmd.AddCommand(newServiceTransactCmd())

	return cmd
}

func newServiceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered binder services",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			conn, err := OpenConn(ctx, cmd)
			if err != nil {
				return err
			}
			defer conn.Close(ctx)

			services, err := conn.SM.ListServices(ctx)
			if err != nil {
				return fmt.Errorf("listing services: %w", err)
			}

			headers := []string{"NAME", "STATUS"}
			rows := make([][]string, 0, len(services))
			for _, name := range services {
				status := serviceStatus(ctx, conn, name)
				rows = append(rows, []string{name, status})
			}

			mode, err := cmd.Root().PersistentFlags().GetString("format")
			if err != nil {
				return fmt.Errorf("reading --format flag: %w", err)
			}

			NewFormatter(mode, os.Stdout).Table(headers, rows)
			return nil
		},
	}
}

func serviceStatus(
	ctx context.Context,
	conn *Conn,
	name string,
) string {
	svc, err := conn.SM.CheckService(ctx, name)
	if err != nil || svc == nil {
		return "not found"
	}

	if svc.IsAlive(ctx) {
		return "alive"
	}
	return "dead"
}

func newServiceInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <name>",
		Short: "Inspect a binder service by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			conn, err := OpenConn(ctx, cmd)
			if err != nil {
				return err
			}
			defer conn.Close(ctx)

			svc, err := conn.GetService(ctx, name)
			if err != nil {
				return err
			}

			descriptor := queryDescriptor(ctx, svc)

			mode, err := cmd.Root().PersistentFlags().GetString("format")
			if err != nil {
				return fmt.Errorf("reading --format flag: %w", err)
			}

			NewFormatter(mode, os.Stdout).Result(map[string]any{
				"name":       name,
				"handle":     svc.Handle(),
				"alive":      svc.IsAlive(ctx),
				"descriptor": descriptor,
			})
			return nil
		},
	}
}

func newServiceTransactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transact <name> <code> [hex-data]",
		Short: "Send a raw transaction to a binder service",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			code, err := strconv.ParseUint(args[1], 0, 32)
			if err != nil {
				return fmt.Errorf("parsing transaction code: %w", err)
			}

			var data *parcel.Parcel
			if len(args) >= 3 {
				raw, err := hex.DecodeString(args[2])
				if err != nil {
					return fmt.Errorf("decoding hex data: %w", err)
				}
				data = parcel.FromBytes(raw)
			} else {
				data = parcel.New()
			}

			conn, err := OpenConn(ctx, cmd)
			if err != nil {
				return err
			}
			defer conn.Close(ctx)

			svc, err := conn.GetService(ctx, name)
			if err != nil {
				return err
			}

			reply, err := svc.Transact(
				ctx,
				binder.TransactionCode(code),
				0,
				data,
			)
			if err != nil {
				return fmt.Errorf("transact failed: %w", err)
			}

			replyData := reply.Data()

			mode, err := cmd.Root().PersistentFlags().GetString("format")
			if err != nil {
				return fmt.Errorf("reading --format flag: %w", err)
			}

			NewFormatter(mode, os.Stdout).Result(map[string]any{
				"reply_size": len(replyData),
				"reply_hex":  hex.EncodeToString(replyData),
			})
			return nil
		},
	}
}

// queryDescriptor sends an InterfaceTransaction to the binder service
// and reads back the interface descriptor string.
// Returns "(unknown)" if the query fails.
func queryDescriptor(
	ctx context.Context,
	svc binder.IBinder,
) string {
	reply, err := svc.Transact(ctx, binder.InterfaceTransaction, 0, parcel.New())
	if err != nil {
		return "(unknown)"
	}

	desc, err := reply.ReadString16()
	if err != nil {
		return "(unknown)"
	}

	return desc
}
