package iana

import (
	"net"

	"github.com/google/gopacket/layers"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
)

type layer2OctetTotalCountPacket struct {
	flows.BaseFeature
}

func (f *layer2OctetTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(new.(packet.Buffer).LinkLayerLength(), context, f)
}

func init() {
	flows.RegisterStandardFeature("layer2OctetTotalCount", flows.PacketFeature, func() flows.Feature { return &layer2OctetTotalCountPacket{} }, flows.RawPacket)
}

type layer2OctetTotalCountFlow struct {
	flows.BaseFeature
	total uint64
}

func (f *layer2OctetTotalCountFlow) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.total = 0
}

func (f *layer2OctetTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.total += uint64(new.(packet.Buffer).LinkLayerLength())
}

func (f *layer2OctetTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.total, context, f)
}

func init() {
	flows.RegisterStandardFeature("layer2OctetTotalCount", flows.FlowFeature, func() flows.Feature { return &layer2OctetTotalCountFlow{} }, flows.RawPacket)
	flows.RegisterStandardCompositeFeature("minimumLayer2TotalLength", "min", "layer2OctetTotalCount")
	flows.RegisterStandardCompositeFeature("maximumLayer2TotalLength", "max", "layer2OctetTotalCount")
}

////////////////////////////////////////////////////////////////////////////////

type ethernetType struct {
	flows.BaseFeature
}

func (f *ethernetType) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(uint16(new.(packet.Buffer).EtherType()), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("ethernetType", flows.FlowFeature, func() flows.Feature { return &ethernetType{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type sourceMacAddress struct {
	flows.BaseFeature
}

func (f *sourceMacAddress) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		link, ok := new.(packet.Buffer).LinkLayer().(*layers.Ethernet)
		if ok {
			f.SetValue(append(net.HardwareAddr(nil), link.SrcMAC...), context, f)
		}
	}
}

func init() {
	flows.RegisterStandardFeature("sourceMacAddress", flows.FlowFeature, func() flows.Feature { return &sourceMacAddress{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationMacAddress struct {
	flows.BaseFeature
}

func (f *destinationMacAddress) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		link, ok := new.(packet.Buffer).LinkLayer().(*layers.Ethernet)
		if ok {
			f.SetValue(append(net.HardwareAddr(nil), link.DstMAC...), context, f)
		}
	}
}

func init() {
	flows.RegisterStandardFeature("destinationMacAddress", flows.FlowFeature, func() flows.Feature { return &destinationMacAddress{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type dot1qVlanID struct {
	flows.BaseFeature
}

func (f *dot1qVlanID) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		dot1q := new.(packet.Buffer).Dot1QLayers()
		if len(dot1q) != 0 {
			f.SetValue(dot1q[0].VLANIdentifier, context, f)
		}
	}
}

func init() {
	flows.RegisterStandardFeature("dot1qVlanId", flows.FlowFeature, func() flows.Feature { return &dot1qVlanID{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type dot1qPriority struct {
	flows.BaseFeature
}

func (f *dot1qPriority) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		dot1q := new.(packet.Buffer).Dot1QLayers()
		if len(dot1q) != 0 {
			f.SetValue(dot1q[0].Priority, context, f)
		}
	}
}

func init() {
	flows.RegisterStandardFeature("dot1qPriority", flows.FlowFeature, func() flows.Feature { return &dot1qPriority{} }, flows.RawPacket)
}
