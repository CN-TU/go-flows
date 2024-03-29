package iana

import (
	"encoding/binary"
	"log"
	"net"

	"github.com/CN-TU/go-ipfix"
	"github.com/google/gopacket/layers"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
)

////////////////////////////////////////////////////////////////////////////////

type sourceIPAddressFlow struct {
	flows.BaseFeature
}

func (f *sourceIPAddressFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		network := new.(packet.Buffer).NetworkLayer()
		if network != nil {
			ipaddr := network.NetworkFlow().Src().Raw() // this makes a copy of the ip
			if ipaddr != nil {
				fin := net.IP(ipaddr)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func (f *sourceIPAddressFlow) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "sourceIPv4Address"
	}
	return 1 // "sourceIPv6Address"
}

func init() {
	ip4, err := ipfix.GetInformationElement("sourceIPv4Address")
	if err != nil {
		log.Panic(err)
	}
	ip6, err := ipfix.GetInformationElement("sourceIPv6Address")
	if err != nil {
		log.Panic(err)
	}
	flows.RegisterStandardVariantFeature("sourceIPAddress", "sourceIPv4Address or sourceIPv6Address depending on ip version", []ipfix.InformationElement{
		ip4,
		ip6,
	}, flows.FlowFeature, func() flows.Feature { return &sourceIPAddressFlow{} }, flows.RawPacket)
}

///////////////////////////////////////////////////////////////////////////////

type sourceIPAddressPacket struct {
	flows.BaseFeature
}

func (f *sourceIPAddressPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if network != nil {
		ipaddr := network.NetworkFlow().Src().Raw() // this makes a copy of the ip
		if ipaddr != nil {
			fin := net.IP(ipaddr)
			f.SetValue(fin, context, f)
		}
	}
}

func (f *sourceIPAddressPacket) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "sourceIPv4Address"
	}
	return 1 // "sourceIPv6Address"
}

func init() {
	ip4, err := ipfix.GetInformationElement("sourceIPv4Address")
	if err != nil {
		log.Panic(err)
	}
	ip6, err := ipfix.GetInformationElement("sourceIPv6Address")
	if err != nil {
		log.Panic(err)
	}
	flows.RegisterStandardVariantFeature("sourceIPAddress", "sourceIPv4Address or sourceIPv6Address depending on ip version", []ipfix.InformationElement{
		ip4,
		ip6,
	}, flows.PacketFeature, func() flows.Feature { return &sourceIPAddressPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationIPAddressFlow struct {
	flows.BaseFeature
}

func (f *destinationIPAddressFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		network := new.(packet.Buffer).NetworkLayer()
		if network != nil {
			ipaddr := network.NetworkFlow().Dst().Raw() // this makes a copy of the ip
			if ipaddr != nil {
				fin := net.IP(ipaddr)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func (f *destinationIPAddressFlow) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "destinationIPv4Address"
	}
	return 1 // "destinationIPv6Address"
}

func init() {
	ip4, err := ipfix.GetInformationElement("destinationIPv4Address")
	if err != nil {
		log.Panic(err)
	}
	ip6, err := ipfix.GetInformationElement("destinationIPv6Address")
	if err != nil {
		log.Panic(err)
	}
	flows.RegisterStandardVariantFeature("destinationIPAddress", "destinationIPv4Address or destinationIPv6Address depending on ip version", []ipfix.InformationElement{
		ip4,
		ip6,
	}, flows.FlowFeature, func() flows.Feature { return &destinationIPAddressFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationIPAddressPacket struct {
	flows.BaseFeature
}

func (f *destinationIPAddressPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if network != nil {
		ipaddr := network.NetworkFlow().Dst().Raw() // this makes a copy of the ip
		if ipaddr != nil {
			fin := net.IP(ipaddr)
			f.SetValue(fin, context, f)
		}
	}
}

func (f *destinationIPAddressPacket) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "destinationIPv4Address"
	}
	return 1 // "destinationIPv6Address"
}

func init() {
	ip4, err := ipfix.GetInformationElement("destinationIPv4Address")
	if err != nil {
		log.Panic(err)
	}
	ip6, err := ipfix.GetInformationElement("destinationIPv6Address")
	if err != nil {
		log.Panic(err)
	}
	flows.RegisterStandardVariantFeature("destinationIPAddress", "destinationIPv4Address or destinationIPv6Address depending on ip version", []ipfix.InformationElement{
		ip4,
		ip6,
	}, flows.PacketFeature, func() flows.Feature { return &destinationIPAddressPacket{} }, flows.RawPacket)
}

///////////////////////////////////////////////////////////////////////////////

type protocolIdentifierFlow struct {
	flows.BaseFeature
}

func (f *protocolIdentifierFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(new.(packet.Buffer).Proto(), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("protocolIdentifier", flows.FlowFeature, func() flows.Feature { return &protocolIdentifierFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type protocolIdentifierPacket struct {
	flows.BaseFeature
}

func (f *protocolIdentifierPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(new.(packet.Buffer).Proto(), context, f)
}

func init() {
	flows.RegisterStandardFeature("protocolIdentifier", flows.PacketFeature, func() flows.Feature { return &protocolIdentifierPacket{} }, flows.RawPacket)
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
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.Length, context, f)
		return
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
					f.SetValue(l, context, f)
					return
				}
			}
		}
		f.SetValue(ip.Length, context, f)
	}
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
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.total += uint64(ip.Length)
		return
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
					f.total += uint64(l)
					return
				}
			}
		}
		f.total += uint64(ip.Length)
	}
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

type fragmentFlags struct {
	flows.BaseFeature
}

func (f *fragmentFlags) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.Flags, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("fragmentFlags", flows.PacketFeature, func() flows.Feature { return &fragmentFlags{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type fragmentIdentification struct {
	flows.BaseFeature
}

func (f *fragmentIdentification) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.Id, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("fragmentIdentification", flows.PacketFeature, func() flows.Feature { return &fragmentIdentification{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type fragmentOffset struct {
	flows.BaseFeature
}

func (f *fragmentOffset) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.FragOffset, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("fragmentOffset", flows.PacketFeature, func() flows.Feature { return &fragmentOffset{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type ipVersion struct {
	flows.BaseFeature
}

func (f *ipVersion) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.Version, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("ipVersion", flows.PacketFeature, func() flows.Feature { return &ipVersion{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type ipHeaderLength struct {
	flows.BaseFeature
}

func (f *ipHeaderLength) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.IHL, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("ipHeaderLength", flows.PacketFeature, func() flows.Feature { return &ipHeaderLength{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////