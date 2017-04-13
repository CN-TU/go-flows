package packet

import (
	"net"

	"encoding/binary"

	"github.com/google/gopacket/layers"
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

func init() {
	flows.RegisterFeature("sourceIPAddress", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &sourceIPAddress{} }, nil},
	})
}

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

func init() {
	flows.RegisterFeature("destinationIPAddress", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &destinationIPAddress{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////
type protocolIdentifier struct {
	flows.BaseFeature
}

func (f *protocolIdentifier) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(f.Key().Proto(), when)
	}
}

func init() {
	flows.RegisterFeature("protocolIdentifier", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &protocolIdentifier{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////
type flowEndReason struct {
	flows.BaseFeature
}

func (f *flowEndReason) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(reason, when)
}

func init() {
	flows.RegisterFeature("flowEndReason", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &flowEndReason{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////
type flowEndNanoSeconds struct {
	flows.BaseFeature
}

func (f *flowEndNanoSeconds) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(when, when)
}

func init() {
	flows.RegisterFeature("flowEndNanoSeconds", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &flowEndNanoSeconds{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type octetTotalCountPacket struct {
	flows.BaseFeature
}

func octetCount(packet *packetBuffer) flows.Unsigned64 {
	length := packet.Metadata().Length
	if net := packet.NetworkLayer(); net != nil {
		length -= len(net.LayerContents())
	}
	return flows.Unsigned64(length)
}

func (f *octetTotalCountPacket) Event(new interface{}, when flows.Time) {
	f.SetValue(octetCount(new.(*packetBuffer)), when)
}

type octetTotalCountFlow struct {
	flows.BaseFeature
	total flows.Unsigned64
}

func (f *octetTotalCountFlow) Start(when flows.Time) {
	f.total = 0
}

func (f *octetTotalCountFlow) Event(new interface{}, when flows.Time) {
	f.total = f.total.Add(octetCount(new.(*packetBuffer))).(flows.Unsigned64)
}

func (f *octetTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.total, when)
}

func init() {
	flows.RegisterFeature("octetTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &octetTotalCountFlow{} }, nil},
		{flows.FeatureTypePacket, func() flows.Feature { return &octetTotalCountPacket{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

func ipTotalLength(packet *packetBuffer) flows.Unsigned64 {
	network := packet.NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		return flows.Unsigned64(ip.Length)
	}
	if ip, ok := network.(*layers.IPv6); ok {
		if ip.HopByHop != nil {
			var tlv *layers.IPv6HopByHopOption
			for _, t := range ip.HopByHop.Options {
				if t.OptionType == layers.IPv6HopByHopOptionJumbogram {
					tlv = t
					break
				}
			}
			if tlv != nil && len(tlv.OptionData) == 4 {
				l := binary.BigEndian.Uint32(tlv.OptionData)
				if l > 65535 {
					return flows.Unsigned64(l)
				}
			}
		}
		return flows.Unsigned64(ip.Length)
	}
	return 0
}

type ipTotalLengthPacket struct {
	flows.BaseFeature
}

func (f *ipTotalLengthPacket) Event(new interface{}, when flows.Time) {
	f.SetValue(ipTotalLength(new.(*packetBuffer)), when)
}

type ipTotalLengthFlow struct {
	flows.BaseFeature
	total flows.Unsigned64
}

func (f *ipTotalLengthFlow) Start(when flows.Time) {
	f.total = 0
}

func (f *ipTotalLengthFlow) Event(new interface{}, when flows.Time) {
	f.total = f.total.Add(ipTotalLength(new.(*packetBuffer))).(flows.Unsigned64)
}

func (f *ipTotalLengthFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.total, when)
}

func init() {
	flows.RegisterFeature("ipTotalLength", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &ipTotalLengthFlow{} }, nil},
		{flows.FeatureTypePacket, func() flows.Feature { return &ipTotalLengthPacket{} }, nil},
	})
	flows.RegisterCompositeFeature("minimumIpTotalLength", []interface{}{"min", "ipTotalLength"})
	flows.RegisterCompositeFeature("maximumIpTotalLength", []interface{}{"max", "ipTotalLength"})
}
