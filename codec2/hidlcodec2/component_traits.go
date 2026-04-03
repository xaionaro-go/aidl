package hidlcodec2

// Domain classifies the media type of a component.
type Domain uint32

const (
	DomainOther Domain = 0
	DomainVideo Domain = 1
	DomainAudio Domain = 2
	DomainImage Domain = 3
)

// Kind classifies whether a component is a decoder or encoder.
type Kind uint32

const (
	KindOther   Kind = 0
	KindDecoder Kind = 1
	KindEncoder Kind = 2
)

// ComponentTraits describes a single Codec2 component.
type ComponentTraits struct {
	Name      string
	Domain    Domain
	Kind      Kind
	Rank      uint32
	MediaType string
	Aliases   []string
}
