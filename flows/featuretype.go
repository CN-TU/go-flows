package flows

// FeatureType represents if the feature is a flow or packet feature.
type FeatureType int

const (
	// Const is a constant
	Const FeatureType = iota

	// RawPacket is a packet from the packet source
	RawPacket
	// RawFlow is a flow from the flow source
	RawFlow

	// PacketFeature is a packet feature
	PacketFeature
	// FlowFeature is a flow feature
	FlowFeature

	// MatchType specifies that the argument type has to match the return type
	MatchType

	// Selection specifies a packet/flow selection
	Selection

	// Ellipsis represents a continuation of the last argument
	Ellipsis

	featureTypeMax
)

func (f FeatureType) String() string {
	switch f {
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
	}
	return "<InvalidFeatureType>"
}
