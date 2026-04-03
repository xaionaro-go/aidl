// Package gralloc provides gralloc buffer allocation via the Android
// IAllocator HAL service.
package gralloc

import (
	"fmt"

	common "github.com/AndroidGoLab/binder/android/hardware/common"

	"golang.org/x/sys/unix"
)

// Buffer holds a gralloc-allocated buffer with its NativeHandle.
type Buffer struct {
	Handle common.NativeHandle
	Stride int32
	Width  uint32
	Height uint32
	Format int32
	Usage  uint64

	// MmapData holds a persistent mmap of the dmabuf, set by Mmap().
	// Keeping it mapped avoids mmap/munmap syscalls per frame read.
	MmapData []byte
}

// mmapAttempt describes one combination of flags to try when mapping a
// gralloc buffer FD.
type mmapAttempt struct {
	prot  int
	flags int
	label string
}

// mmapStrategies lists the mmap flag combinations to try, in order.
// Different allocator backends (AIDL gralloc, HIDL gralloc, dma-buf heap,
// memfd) produce FDs with different mmap requirements:
//   - dma-buf and memfd FDs typically work with PROT_READ | MAP_SHARED
//   - Some gralloc FDs require PROT_READ|PROT_WRITE
//   - Some FDs only support MAP_PRIVATE
var mmapStrategies = []mmapAttempt{
	{unix.PROT_READ, unix.MAP_SHARED, "PROT_READ|MAP_SHARED"},
	{unix.PROT_READ | unix.PROT_WRITE, unix.MAP_SHARED, "PROT_READ|PROT_WRITE|MAP_SHARED"},
	{unix.PROT_READ, unix.MAP_PRIVATE, "PROT_READ|MAP_PRIVATE"},
	{unix.PROT_READ | unix.PROT_WRITE, unix.MAP_PRIVATE, "PROT_READ|PROT_WRITE|MAP_PRIVATE"},
}

// Mmap creates a persistent mmap of this buffer's dmabuf FD.
// It tries several flag combinations to handle different allocator
// backends (AIDL gralloc, HIDL gralloc, dma-buf heap, memfd).
// The MmapData field can then be read directly. Call Munmap to release.
func (b *Buffer) Mmap() error {
	if len(b.Handle.Fds) == 0 {
		return fmt.Errorf("no FDs in gralloc buffer")
	}
	fd := int(b.Handle.Fds[0])
	// YCbCr_420_888: Y plane (w*h) + CbCr interleaved (w*h/2).
	bufSize := int(b.Width) * int(b.Height) * 3 / 2

	var lastErr error
	for _, strategy := range mmapStrategies {
		data, err := unix.Mmap(fd, 0, bufSize, strategy.prot, strategy.flags)
		if err == nil {
			b.MmapData = data
			return nil
		}
		lastErr = fmt.Errorf("%s: %w", strategy.label, err)
	}
	return fmt.Errorf("mmap fd=%d size=%d: all strategies failed, last: %w", fd, bufSize, lastErr)
}

// Munmap releases the persistent mmap created by Mmap.
func (b *Buffer) Munmap() {
	if b.MmapData != nil {
		_ = unix.Munmap(b.MmapData)
		b.MmapData = nil
	}
}
