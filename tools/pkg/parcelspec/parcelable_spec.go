package parcelspec

// ParcelableSpec describes the wire format of a single Java Parcelable type.
type ParcelableSpec struct {
	Package string      `yaml:"package"`
	Type    string      `yaml:"type"`
	Fields  []FieldSpec `yaml:"fields"`
}
