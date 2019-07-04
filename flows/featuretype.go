package flows

// FeatureType represents if the feature is a flow or packet feature. This is used for argument and return type specification.
type FeatureType int

const (
	_ FeatureType = iota
	// Const is a constant
	Const

	// RawPacket is a packet from the packet source
	RawPacket
	// RawFlow is a flow from the flow source
	RawFlow

	// PacketFeature is a packet feature, i.e., emits one value per packet
	PacketFeature
	// FlowFeature is a flow feature, i.e., emits one value per flow
	FlowFeature

	// MatchType specifies that the argument type has to match the return type
	MatchType

	// Selection specifies a packet/flow selection
	Selection

	// Ellipsis represents a continuation of the last argument
	Ellipsis

	// ControlFeature is a feature, that is called first and is able to modify Flow behaviour
	ControlFeature

	featureTypeMax
)

func (f FeatureType) String() string {
	switch f {
	case 0:
		return "Unknown"
	case Const:
		return "Const"
	case RawPacket:
		return "RawPacket"
	case RawFlow:
		return "RawFlow"
	case PacketFeature:
		return "PacketFeature"
	case FlowFeature:
		return "FlowFeature"
	case MatchType:
		return "MatchType"
	case Selection:
		return "Selection"
	case Ellipsis:
		return "..."
	case ControlFeature:
		return "ControlFeature"
	}
	return "<InvalidFeatureType>"
}
