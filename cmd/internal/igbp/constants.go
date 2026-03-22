// Package igbp re-exports IGraphicBufferProducer constants from the
// public android/gui/igbp package for backward compatibility with
// existing cmd/ code.
package igbp

import "github.com/xaionaro-go/binder/android/gui/igbp"

// Re-export all types and constants from the public package.
type IGBPTransaction = igbp.IGBPTransaction
type NativeWindowQuery = igbp.NativeWindowQuery
type PixelFormat = igbp.PixelFormat
type AndroidStatus = igbp.AndroidStatus

const (
	Descriptor = igbp.Descriptor

	RequestBuffer          = igbp.RequestBuffer
	DequeueBuffer          = igbp.DequeueBuffer
	DetachBuffer           = igbp.DetachBuffer
	DetachNextBuffer       = igbp.DetachNextBuffer
	AttachBuffer           = igbp.AttachBuffer
	QueueBuffer            = igbp.QueueBuffer
	CancelBuffer           = igbp.CancelBuffer
	Query                  = igbp.Query
	Connect                = igbp.Connect
	Disconnect             = igbp.Disconnect
	SetSidebandStream      = igbp.SetSidebandStream
	AllocateBuffers        = igbp.AllocateBuffers
	AllowAllocation        = igbp.AllowAllocation
	SetGenerationNumber    = igbp.SetGenerationNumber
	GetConsumerName        = igbp.GetConsumerName
	SetMaxDequeuedBufCount = igbp.SetMaxDequeuedBufCount
	SetAsyncMode           = igbp.SetAsyncMode
	SetSharedBufferMode    = igbp.SetSharedBufferMode
	SetAutoRefresh         = igbp.SetAutoRefresh
	SetDequeueTimeout      = igbp.SetDequeueTimeout
	GetLastQueuedBuffer    = igbp.GetLastQueuedBuffer
	GetFrameTimestamps     = igbp.GetFrameTimestamps
	GetUniqueId            = igbp.GetUniqueId
	GetConsumerUsage       = igbp.GetConsumerUsage
	SetLegacyBufferDrop    = igbp.SetLegacyBufferDrop
	SetAutoPrerotation     = igbp.SetAutoPrerotation

	NativeWindowWidth             = igbp.NativeWindowWidth
	NativeWindowHeight            = igbp.NativeWindowHeight
	NativeWindowFormat            = igbp.NativeWindowFormat
	NativeWindowMinUndequeued     = igbp.NativeWindowMinUndequeued
	NativeWindowQueuesToComposer  = igbp.NativeWindowQueuesToComposer
	NativeWindowConcreteType      = igbp.NativeWindowConcreteType
	NativeWindowDefaultWidth      = igbp.NativeWindowDefaultWidth
	NativeWindowDefaultHeight     = igbp.NativeWindowDefaultHeight
	NativeWindowTransformHint     = igbp.NativeWindowTransformHint
	NativeWindowConsumerRunning   = igbp.NativeWindowConsumerRunning
	NativeWindowConsumerUsageBits = igbp.NativeWindowConsumerUsageBits
	NativeWindowStickyTransform   = igbp.NativeWindowStickyTransform
	NativeWindowDefaultDataspace  = igbp.NativeWindowDefaultDataspace
	NativeWindowBufferAge         = igbp.NativeWindowBufferAge
	NativeWindowMaxBufferCount    = igbp.NativeWindowMaxBufferCount
	NativeWindowSurface           = igbp.NativeWindowSurface

	PixelFormatImplementationDefined = igbp.PixelFormatImplementationDefined
	PixelFormatYCbCr420_888          = igbp.PixelFormatYCbCr420_888

	StatusOK     = igbp.StatusOK
	StatusNoInit = igbp.StatusNoInit

	BufferNeedsRealloc     = igbp.BufferNeedsRealloc
	GraphicBufferMagicGB01 = igbp.GraphicBufferMagicGB01
	MaxBufferSlots         = igbp.MaxBufferSlots
)
