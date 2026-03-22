package runner

import (
	"context"
	"fmt"

	"github.com/electricbubble/gadb"
	"github.com/facebookincubator/go-belt/tool/logger"
)

// DiscoverDevices enumerates Android devices connected via ADB.
func DiscoverDevices(
	ctx context.Context,
) ([]DeviceInfo, error) {
	logger.Tracef(ctx, "DiscoverDevices")

	client, err := gadb.NewClient()
	if err != nil {
		return nil, fmt.Errorf("connecting to ADB server: %w", err)
	}

	devices, err := client.DeviceList()
	if err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}

	result := make([]DeviceInfo, 0, len(devices))
	for _, dev := range devices {
		state, err := dev.State()
		if err != nil {
			logger.Warnf(ctx, "unable to query state for device %s: %v", dev.Serial(), err)
			continue
		}
		result = append(result, DeviceInfo{
			Serial: dev.Serial(),
			State:  string(state),
		})
	}

	logger.Debugf(ctx, "discovered %d device(s)", len(result))
	return result, nil
}
