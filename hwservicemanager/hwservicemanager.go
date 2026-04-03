// Package hwservicemanager provides a client for Android's HIDL
// hwservicemanager, which runs on /dev/hwbinder at handle 0.
// It allows looking up HIDL HAL services by their versioned
// fully-qualified names (e.g. "android.hardware.graphics.allocator@3.0::IAllocator/default").
package hwservicemanager

import (
	"context"
	"fmt"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/hwparcel"
	"github.com/AndroidGoLab/binder/logger"
	"github.com/AndroidGoLab/binder/parcel"
)

const (
	// hwServiceManagerHandle is the well-known handle for hwservicemanager.
	hwServiceManagerHandle = uint32(0)

	// hwServiceManagerDescriptor is the HIDL interface descriptor.
	hwServiceManagerDescriptor = "android.hidl.manager@1.0::IServiceManager"

	// transactionGet is the HIDL transaction code for IServiceManager::get().
	// HIDL uses FIRST_CALL_TRANSACTION (1) + method_index.
	// In IServiceManager 1.0, get() is the first method (index 0).
	transactionGet = binder.TransactionCode(1)
)

// HwServiceManager provides access to Android's HIDL hwservicemanager.
type HwServiceManager struct {
	transport binder.Transport
}

// New creates a HwServiceManager client using the given transport
// (which must be connected to /dev/hwbinder).
func New(transport binder.Transport) *HwServiceManager {
	return &HwServiceManager{transport: transport}
}

// GetService retrieves a HIDL service by its fully-qualified name and instance.
// fqName is e.g. "android.hardware.graphics.allocator@3.0::IAllocator"
// instance is e.g. "default"
func (sm *HwServiceManager) GetService(
	ctx context.Context,
	fqName string,
	instance string,
) (_ uint32, _err error) {
	logger.Tracef(ctx, "hwservicemanager.GetService(%q, %q)", fqName, instance)
	defer func() { logger.Tracef(ctx, "/hwservicemanager.GetService: %v", _err) }()

	// Build HwParcel for get(string fqName, string name).
	hp := hwparcel.New()
	hp.WriteInterfaceToken(hwServiceManagerDescriptor)
	hp.WriteHidlString(fqName)
	hp.WriteHidlString(instance)

	data, keepAlive := hp.ToParcel()

	reply, err := sm.transport.Transact(
		ctx,
		hwServiceManagerHandle,
		transactionGet,
		binder.FlagAcceptFDs,
		data,
	)
	hwparcel.KeepBuffersAlive(keepAlive)
	if err != nil {
		return 0, fmt.Errorf("hwservicemanager: get(%q, %q): %w", fqName, instance, err)
	}
	defer reply.Recycle()

	// HIDL reply format for get():
	//   int32  HIDL status (0 = OK)
	//   flat_binder_object (24 bytes) containing the service handle
	resp := hwparcel.NewResponseParcel(reply)

	status, err := resp.ReadInt32()
	if err != nil {
		return 0, fmt.Errorf("hwservicemanager: get(%q, %q): reading status: %w", fqName, instance, err)
	}
	if status != 0 {
		return 0, fmt.Errorf("hwservicemanager: get(%q, %q): HIDL status error: %d", fqName, instance, status)
	}

	handle, err := resp.ReadStrongBinder()
	if err != nil {
		return 0, fmt.Errorf("hwservicemanager: get(%q, %q): reading binder: %w", fqName, instance, err)
	}

	return handle, nil
}

// TransactHidl performs a raw HIDL transaction on a handle obtained from
// GetService. The caller builds an HwParcel with the request data.
func TransactHidl(
	ctx context.Context,
	transport binder.Transport,
	handle uint32,
	code binder.TransactionCode,
	hp *hwparcel.HwParcel,
) (*parcel.Parcel, error) {
	data, keepAlive := hp.ToParcel()

	reply, err := transport.Transact(ctx, handle, code, binder.FlagAcceptFDs, data)
	hwparcel.KeepBuffersAlive(keepAlive)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
