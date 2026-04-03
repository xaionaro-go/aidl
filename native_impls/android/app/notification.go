package app

import (
	"fmt"

	"github.com/AndroidGoLab/binder/parcel"
)

// Notification is a native parcelable whose wire format is defined by
// Java's Notification.writeToParcel / readFromParcelImpl.
// This implementation reads and discards all fields to advance the
// parcel position correctly, without storing any data.
type Notification struct{}

var _ parcel.Parcelable = (*Notification)(nil)

func (s *Notification) MarshalParcel(p *parcel.Parcel) error {
	return fmt.Errorf("Notification.MarshalParcel: not implemented")
}

func (s *Notification) UnmarshalParcel(p *parcel.Parcel) error {
	return parcel.SkipNotification(p)
}
