package spec

// ConstantSpec describes an AIDL constant declaration within
// an interface or parcelable.
type ConstantSpec struct {
	Name  string `yaml:"name"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}
