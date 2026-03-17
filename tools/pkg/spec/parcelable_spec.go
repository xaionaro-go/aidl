package spec

// ParcelableSpec describes an AIDL parcelable (struct).
type ParcelableSpec struct {
	Name string `yaml:"name"`

	Fields      []FieldSpec    `yaml:"fields,omitempty"`
	Constants   []ConstantSpec `yaml:"constants,omitempty"`
	NestedTypes []string       `yaml:"nested_types,omitempty"`
	Annotations []string       `yaml:"annotations,omitempty"`

	// JavaWireFormat describes the field serialization order and methods
	// as extracted from the Java writeToParcel() implementation.
	// Present only when java2spec has analyzed the corresponding Java class.
	JavaWireFormat []JavaWireField `yaml:"java_wire_format,omitempty"`
}

// FieldSpec describes a parcelable or union field.
type FieldSpec struct {
	Name         string `yaml:"name"`
	Type         TypeRef `yaml:"type"`
	DefaultValue string `yaml:"default_value,omitempty"`
	Annotations  []string `yaml:"annotations,omitempty"`
}

// JavaWireField describes one field's serialization in the Java
// writeToParcel() method. Used to generate Go marshal/unmarshal
// code that matches the Java wire format.
type JavaWireField struct {
	Name        string `yaml:"name"`
	WriteMethod string `yaml:"write_method"`
	Condition   string `yaml:"condition,omitempty"`
}
