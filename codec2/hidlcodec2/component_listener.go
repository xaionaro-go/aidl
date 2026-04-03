package hidlcodec2

import (
	"context"

	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/logger"
	"github.com/AndroidGoLab/binder/parcel"
)

const (
	componentListenerFQName = "android.hardware.media.c2@1.0::IComponentListener"
)

// ComponentListenerStub is a minimal HIDL binder stub for
// IComponentListener. It receives oneway callbacks from the Codec2
// component (onWorkDone, onTripped, onError, etc.) and silently
// acknowledges them.
//
// The stub implements binder.TransactionReceiver so it can be registered
// with the kernelbinder.Driver to receive incoming BR_TRANSACTION events.
type ComponentListenerStub struct {
	// OnWorkDone is called when the component finishes processing work.
	// The raw reply data is passed for callers who want to inspect it.
	// May be nil if the caller does not need callbacks.
	OnWorkDone func(data []byte)
}

var _ binder.TransactionReceiver = (*ComponentListenerStub)(nil)

// Descriptor returns the HIDL interface descriptor.
func (s *ComponentListenerStub) Descriptor() string {
	return componentListenerFQName
}

// OnTransaction handles incoming HIDL transactions from the component.
// IComponentListener methods are all oneway, so no reply is expected.
//
// HIDL transaction codes for IComponentListener 1.0:
//
//	1: onWorkDone(WorkBundle)
//	2: onTripped(vec<SettingResult>)
//	3: onError(Status, uint32)
//	4: onFramesRendered(vec<RenderedFrame>)
//	5: onInputBuffersReleased(vec<InputBuffer>)
func (s *ComponentListenerStub) OnTransaction(
	ctx context.Context,
	code binder.TransactionCode,
	data *parcel.Parcel,
) (*parcel.Parcel, error) {
	switch code {
	case 1: // onWorkDone
		logger.Debugf(ctx, "IComponentListener.onWorkDone received (%d bytes)", data.Len())
		if s.OnWorkDone != nil {
			s.OnWorkDone(data.Data())
		}
	case 2: // onTripped
		logger.Debugf(ctx, "IComponentListener.onTripped received")
	case 3: // onError
		logger.Debugf(ctx, "IComponentListener.onError received")
	case 4: // onFramesRendered
		logger.Debugf(ctx, "IComponentListener.onFramesRendered received")
	case 5: // onInputBuffersReleased
		logger.Debugf(ctx, "IComponentListener.onInputBuffersReleased received")
	default:
		logger.Debugf(ctx, "IComponentListener: unknown transaction code %d", code)
	}

	// Oneway callbacks have no reply.
	return parcel.New(), nil
}

// RegisterListener registers the listener stub with the hwbinder driver
// and returns the cookie that can be used to write a local binder
// reference in HIDL transactions.
func RegisterListener(
	ctx context.Context,
	driver *kernelbinder.Driver,
	stub *ComponentListenerStub,
) uintptr {
	return driver.RegisterReceiver(ctx, stub)
}

// UnregisterListener removes the listener stub from the hwbinder driver.
func UnregisterListener(
	ctx context.Context,
	driver *kernelbinder.Driver,
	cookie uintptr,
) {
	driver.UnregisterReceiver(ctx, cookie)
}
