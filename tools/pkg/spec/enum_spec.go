package spec

// EnumSpec describes an AIDL enum.
type EnumSpec struct {
	Name        string           `yaml:"name"`
	BackingType string           `yaml:"backing_type"`
	Values      []EnumeratorSpec `yaml:"values"`
	Annotations []string         `yaml:"annotations,omitempty"`
}

// EnumeratorSpec describes a single enum value.
type EnumeratorSpec struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
