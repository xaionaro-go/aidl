package types

import (
	"encoding/binary"

	"github.com/AndroidGoLab/binder/parcel"
)

// ComponentName mirrors android.content.ComponentName.
// See android/content/componentname.go for rationale.
type ComponentName struct {
}

var _ parcel.Parcelable = (*ComponentName)(nil)

func (s *ComponentName) MarshalParcel(
	p *parcel.Parcel,
) error {
	// Overwrite the non-null flag the proxy already wrote (1 → 0)
	// so the Java server treats this ComponentName as null.
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
