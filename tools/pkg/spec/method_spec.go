package spec

// MethodSpec describes a single AIDL interface method.
type MethodSpec struct {
	Name string `yaml:"name"`

	// TransactionCodeOffset is the offset from binder.FirstCallTransaction.
	// The actual code is FirstCallTransaction + TransactionCodeOffset.
	TransactionCodeOffset int `yaml:"transaction_code_offset"`

	Oneway      bool        `yaml:"oneway,omitempty"`
	Params      []ParamSpec `yaml:"params,omitempty"`
	ReturnType  TypeRef     `yaml:"return_type,omitempty"`
	Annotations []string    `yaml:"annotations,omitempty"`
}

// ParamSpec describes a method parameter.
type ParamSpec struct {
	Name        string    `yaml:"name"`
	Type        TypeRef   `yaml:"type"`
	Direction   Direction `yaml:"direction,omitempty"`
	Annotations []string  `yaml:"annotations,omitempty"`
}

// Direction indicates parameter directionality in AIDL.
type Direction string

const (
	DirectionIn    Direction = "in"
	DirectionOut   Direction = "out"
	DirectionInOut Direction = "inout"
)
