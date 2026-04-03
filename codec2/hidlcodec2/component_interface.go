package hidlcodec2

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/hwparcel"
	"github.com/AndroidGoLab/binder/hwservicemanager"
	"github.com/AndroidGoLab/binder/logger"
)

const (
	componentInterfaceFQName = "android.hardware.media.c2@1.0::IComponentInterface"
)

// ComponentInterface is a HIDL client for IComponentInterface.
type ComponentInterface struct {
	transport binder.Transport
	handle    uint32
}

// GetConfigurable calls IComponentInterface::getConfigurable() and
// returns a Configurable client.
func (ci *ComponentInterface) GetConfigurable(
	ctx context.Context,
) (_ *Configurable, _err error) {
	logger.Tracef(ctx, "hidlcodec2.ComponentInterface.GetConfigurable")
	defer func() { logger.Tracef(ctx, "/hidlcodec2.ComponentInterface.GetConfigurable: %v", _err) }()

	hp := hwparcel.New()
	hp.WriteInterfaceToken(componentInterfaceFQName)

	reply, err := hwservicemanager.TransactHidl(
		ctx, ci.transport, ci.handle, transactionGetConfigurable, hp,
	)
	if err != nil {
		return nil, fmt.Errorf("getConfigurable transaction: %w", err)
	}
	defer reply.Recycle()

	data := reply.Data()
	if len(data) < 28 {
		return nil, fmt.Errorf("getConfigurable response too short: %d bytes", len(data))
	}

	hidlStatus := int32(binary.LittleEndian.Uint32(data[0:]))
	if hidlStatus != 0 {
		return nil, fmt.Errorf("HIDL status error: %d", hidlStatus)
	}

	cfgHandle, err := readBinderHandle(data[4:])
	if err != nil {
		return nil, fmt.Errorf("reading configurable binder: %w", err)
	}

	return &Configurable{
		transport: ci.transport,
		handle:    cfgHandle,
	}, nil
}
