package binder

import (
	"context"

	"github.com/xaionaro-go/binder/parcel"
)

// TransactionReceiver processes incoming binder transactions.
// Implemented by generated Stub types that dispatch transaction
// codes to typed Go interface methods.
type TransactionReceiver interface {
	OnTransaction(
		ctx context.Context,
		code TransactionCode,
		data *parcel.Parcel,
	) (*parcel.Parcel, error)
}
