package packet

import (
	"net"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

////////////////////////////////////////////////////////////////////////////////
type sourceIPAddress struct {
	flows.BaseFeature
}

func (f *sourceIPAddress) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().SrcIP()), when)
	}
}

func (f *sourceIPAddress) Type() string {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return "sourceIPv4Address"
	}
	return "sourceIPv6Address"
}

var SourceIPAdress = flows.RegisterFeature("sourceIPAddress", []flows.FeatureCreator{
	{flows.FeatureTypeFlow, func() flows.Feature { return &sourceIPAddress{} }, nil},
})

////////////////////////////////////////////////////////////////////////////////
type destinationIPAddress struct {
	flows.BaseFeature
}

func (f *destinationIPAddress) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().DstIP()), when)
	}
}

func (f *destinationIPAddress) Type() string {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return "destinationIPv4Address"
	}
	return "destinationIPv6Address"
}

var DestinationIPAdress = flows.RegisterFeature("destinationIPAddress", []flows.FeatureCreator{
	{flows.FeatureTypeFlow, func() flows.Feature { return &destinationIPAddress{} }, nil},
})

////////////////////////////////////////////////////////////////////////////////
type protocolIdentifier struct {
	flows.BaseFeature
}

func (f *protocolIdentifier) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(f.Key().Proto(), when)
	}
}

var ProtocolIdentifier = flows.RegisterFeature("protocolIdentifier", []flows.FeatureCreator{
	{flows.FeatureTypeFlow, func() flows.Feature { return &protocolIdentifier{} }, nil},
})

////////////////////////////////////////////////////////////////////////////////
type flowEndReason struct {
	flows.BaseFeature
}

func (f *flowEndReason) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(reason, when)
}

var FlowEndReason = flows.RegisterFeature("flowEndReason", []flows.FeatureCreator{
	{flows.FeatureTypeFlow, func() flows.Feature { return &flowEndReason{} }, nil},
})

////////////////////////////////////////////////////////////////////////////////
type flowEndNanoSeconds struct {
	flows.BaseFeature
}

func (f *flowEndNanoSeconds) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(when, when)
}

var FlowEndNanoSeconds = flows.RegisterFeature("flowEndNanoSeconds", []flows.FeatureCreator{
	{flows.FeatureTypeFlow, func() flows.Feature { return &flowEndNanoSeconds{} }, nil},
})

////////////////////////////////////////////////////////////////////////////////

type octetTotalCountPacket struct {
	flows.BaseFeature
}

func octetCount(packet *packetBuffer) int {
	length := packet.Metadata().Length
	if net := packet.NetworkLayer(); net != nil {
		length -= len(net.LayerContents())
	}
	return length
}

func (f *octetTotalCountPacket) Event(new interface{}, when flows.Time) {
	f.SetValue(octetCount(new.(*packetBuffer)), when)
}

type octetTotalCountFlow struct {
	flows.BaseFeature
	total int
}

func (f *octetTotalCountFlow) Start(when flows.Time) {
	f.total = 0
}

func (f *octetTotalCountFlow) Event(new interface{}, when flows.Time) {
	f.total += octetCount(new.(*packetBuffer))
}

func (f *octetTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.total, when)
}

var OctetTotalCount = flows.RegisterFeature("octetTotalCount", []flows.FeatureCreator{
	{flows.FeatureTypeFlow, func() flows.Feature { return &octetTotalCountFlow{} }, nil},
	{flows.FeatureTypePacket, func() flows.Feature { return &octetTotalCountPacket{} }, nil},
})
