package iana

import (
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
