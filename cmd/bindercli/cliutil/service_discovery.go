//go:build linux

package cliutil

import (
	"context"
	"fmt"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// KnownServiceNames maps AIDL descriptors to well-known Android
// ServiceManager names, allowing fast lookup without enumeration.
// Populated by generated code via SetKnownServiceNames.
var KnownServiceNames map[string]string

// FindServiceByDescriptor locates a binder service by its AIDL descriptor.
// It first tries the static map of well-known service names to avoid
// slow enumeration, then falls back to listing all services.
func FindServiceByDescriptor(
	ctx context.Context,
	conn *Conn,
	descriptor string,
) (binder.IBinder, error) {
	// Try the static map of well-known service names first to avoid
	// slow enumeration of all registered services.
	if name, ok := KnownServiceNames[descriptor]; ok {
		svc, err := conn.SM.CheckService(ctx, servicemanager.ServiceName(name))
		if err == nil && svc != nil {
			return svc, nil
		}
	}

	// Fall back to enumeration.
	services, err := conn.SM.ListServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	for _, name := range services {
		svc, err := conn.SM.CheckService(ctx, name)
		if err != nil || svc == nil {
			continue
		}
		desc := QueryDescriptor(ctx, svc)
		if desc == descriptor {
			return svc, nil
		}
	}

	return nil, fmt.Errorf("no service with descriptor %q found", descriptor)
}

// QueryDescriptor sends an InterfaceTransaction to the binder service
// and reads back the interface descriptor string.
// Returns "(unknown)" if the query fails.
func QueryDescriptor(
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
