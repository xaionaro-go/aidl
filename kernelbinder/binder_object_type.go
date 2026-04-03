//go:build linux

package kernelbinder

// binderObjectType represents a binder object type code (e.g. BINDER_TYPE_HANDLE).
type binderObjectType uint32

// binderTypeHandle is BINDER_TYPE_HANDLE: B_PACK_CHARS('s','h','*',0x85) = 0x73682a85.
// Used to identify flat_binder_object entries containing remote binder handles.
const binderTypeHandle = binderObjectType(0x73682a85)

// binderTypePTR is BINDER_TYPE_PTR: B_PACK_CHARS('p','t','*',0x85) = 0x70742a85.
// Used for scatter-gather buffer objects in HIDL (hwbinder) transactions.
const binderTypePTR = binderObjectType(0x70742a85)

// binderTypeFDA is BINDER_TYPE_FDA: B_PACK_CHARS('f','d','a',0x85) = 0x66646185.
// Used for file descriptor arrays in HIDL transactions.
const binderTypeFDA = binderObjectType(0x66646185)
