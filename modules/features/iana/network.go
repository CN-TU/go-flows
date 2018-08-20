package iana

import (
	"net"

	"github.com/CN-TU/go-ipfix"
	"github.com/google/gopacket/layers"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
)

////////////////////////////////////////////////////////////////////////////////

type sourceIPAddress struct {
	flows.BaseFeature
}

func (f *sourceIPAddress) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		network := new.(packet.Buffer).NetworkLayer()
		if network != nil {
			ipaddr := network.NetworkFlow().Src().Raw()
			if ipaddr != nil {
				fin := net.IP(ipaddr)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func (f *sourceIPAddress) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "sourceIPv4Address"
	}
	return 1 // "sourceIPv6Address"
}

func init() {
	flows.RegisterStandardVariantFeature("sourceIPAddress", []ipfix.InformationElement{
		ipfix.GetInformationElement("sourceIPv4Address"),
		ipfix.GetInformationElement("sourceIPv6Address"),
	}, flows.FlowFeature, func() flows.Feature { return &sourceIPAddress{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationIPAddress struct {
	flows.BaseFeature
}

func (f *destinationIPAddress) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		network := new.(packet.Buffer).NetworkLayer()
		if network != nil {
			ipaddr := network.NetworkFlow().Dst().Raw()
			if ipaddr != nil {
				fin := net.IP(ipaddr)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func (f *destinationIPAddress) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "destinationIPv4Address"
	}
	return 1 // "destinationIPv6Address"
}

func init() {
	flows.RegisterStandardVariantFeature("destinationIPAddress", []ipfix.InformationElement{
		ipfix.GetInformationElement("destinationIPv4Address"),
		ipfix.GetInformationElement("destinationIPv6Address"),
	}, flows.FlowFeature, func() flows.Feature { return &destinationIPAddress{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type protocolIdentifier struct {
	flows.BaseFeature
}

func (f *protocolIdentifier) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(new.(packet.Buffer).Proto(), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("protocolIdentifier", flows.FlowFeature, func() flows.Feature { return &protocolIdentifier{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type octetTotalCountPacket struct {
	flows.BaseFeature
}

func (f *octetTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(new.(packet.Buffer).NetworkLayerLength(), context, f)
}

func init() {
	flows.RegisterStandardFeature("octetTotalCount", flows.PacketFeature, func() flows.Feature { return &octetTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type octetTotalCountFlow struct {
	flows.BaseFeature
	total uint64
}

func (f *octetTotalCountFlow) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.total = 0
}

func (f *octetTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.total += uint64(new.(packet.Buffer).NetworkLayerLength())
}

func (f *octetTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.total, context, f)
}

func init() {
	flows.RegisterStandardFeature("octetTotalCount", flows.FlowFeature, func() flows.Feature { return &octetTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type ipTotalLengthPacket struct {
	flows.BaseFeature
}

func (f *ipTotalLengthPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(new.(packet.Buffer).NetworkLayerLength(), context, f)
}

func init() {
	flows.RegisterStandardFeature("ipTotalLength", flows.PacketFeature, func() flows.Feature { return &ipTotalLengthPacket{} }, flows.RawPacket)
}

type ipTotalLengthFlow struct {
	flows.BaseFeature
	total uint64
}

func (f *ipTotalLengthFlow) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.total = 0
}

func (f *ipTotalLengthFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.total += uint64(new.(packet.Buffer).NetworkLayerLength())
}

func (f *ipTotalLengthFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.total, context, f)
}

func init() {
	flows.RegisterStandardFeature("ipTotalLength", flows.FlowFeature, func() flows.Feature { return &ipTotalLengthFlow{} }, flows.RawPacket)
	flows.RegisterStandardCompositeFeature("minimumIpTotalLength", "min", "ipTotalLength")
	flows.RegisterStandardCompositeFeature("maximumIpTotalLength", "max", "ipTotalLength")
}

////////////////////////////////////////////////////////////////////////////////

type ipTTL struct {
	flows.BaseFeature
}

func (f *ipTTL) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(uint64(ip.TTL), context, f)
	}
	if ip, ok := network.(*layers.IPv6); ok {
		f.SetValue(uint64(ip.HopLimit), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("ipTTL", flows.PacketFeature, func() flows.Feature { return &ipTTL{} }, flows.RawPacket)
	flows.RegisterStandardCompositeFeature("minimumTTL", "min", "ipTTL")
	flows.RegisterStandardCompositeFeature("maximumTTL", "max", "ipTTL")
}

////////////////////////////////////////////////////////////////////////////////

type ipClassOfService struct {
	flows.BaseFeature
}

func (f *ipClassOfService) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.TOS, context, f)
	}
	if ip, ok := network.(*layers.IPv6); ok {
		f.SetValue(ip.TrafficClass, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("ipClassOfService", flows.PacketFeature, func() flows.Feature { return &ipClassOfService{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
