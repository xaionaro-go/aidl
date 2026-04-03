package types

import (
	"fmt"

	"github.com/AndroidGoLab/binder/parcel"
)

// Notification is the types sub-package copy of android.app.Notification.
// It reads and discards all fields to advance the parcel position correctly.
type Notification struct{}

var _ parcel.Parcelable = (*Notification)(nil)

func (s *Notification) MarshalParcel(p *parcel.Parcel) error {
	return fmt.Errorf("Notification.MarshalParcel: not implemented")
}

func (s *Notification) UnmarshalParcel(p *parcel.Parcel) error {
	return parcel.SkipNotification(p)
}
