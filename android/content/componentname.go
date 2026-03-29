package content

import (
	"encoding/binary"

	"github.com/AndroidGoLab/binder/parcel"
)

// ComponentName mirrors android.content.ComponentName.
// It is currently an opaque type: the generated proxy always writes a
// non-null flag (int32 = 1) before calling MarshalParcel.  Because this
// Go struct has no fields, we treat every instance as "null on the wire"
// by rewinding the flag the proxy already wrote from 1 → 0.  The Java
// server then sees 0 and treats the ComponentName as null, which is the
// correct behaviour for queries like "aggregate across all admins".
type ComponentName struct {
}

var _ parcel.Parcelable = (*ComponentName)(nil)

func (s *ComponentName) MarshalParcel(
	p *parcel.Parcel,
) error {
	// The generated proxy wrote int32(1) (non-null flag) immediately
	// before this call.  Overwrite it with 0 so the Java server treats
	// this ComponentName as null and skips deserialization — avoiding
	// NullPointerException("package name is null") inside
	// ComponentName(Parcel).
	d := p.Data()
	if len(d) >= 4 {
		binary.LittleEndian.PutUint32(d[len(d)-4:], 0)
	}
	return nil
}

func (s *ComponentName) UnmarshalParcel(
	p *parcel.Parcel,
) error {
	return nil // opaque: cannot skip without known wire format
}
