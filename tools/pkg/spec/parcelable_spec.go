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

	// NativeParcelable marks a parcelable whose wire format is defined
	// by native C++/JNI code rather than AIDL fields. The codegen skips
	// these types; hand-written implementations are provided via native_impls/.
	NativeParcelable bool `yaml:"native_parcelable,omitempty"`
}
