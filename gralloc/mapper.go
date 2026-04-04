package gralloc

import (
	"context"
	"sync"
)

// Mapper provides CPU access to gralloc buffers via the HIDL IMapper HAL.
type Mapper interface {
	LockBuffer(ctx context.Context, b *Buffer) ([]byte, error)
}

// GetMapper returns a Mapper for the current device. The result is cached.
// Returns an error when no mapper is available.
func GetMapper(ctx context.Context) (Mapper, error) {
	globalMapperOnce.Do(func() {
		globalMapper, globalMapperErr = newHIDLMapper(ctx)
	})
	return globalMapper, globalMapperErr
}

var (
	globalMapper     Mapper
	globalMapperOnce sync.Once
	globalMapperErr  error
)
