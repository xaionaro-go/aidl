package spec

// UnionSpec describes an AIDL union (tagged variant type).
type UnionSpec struct {
	Name        string      `yaml:"name"`
	Fields      []FieldSpec `yaml:"fields"`
	Annotations []string    `yaml:"annotations,omitempty"`
}
