package iana

import (
	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	"github.com/CN-TU/go-ipfix"
)

type flowEndReason struct {
	flows.BaseFeature
}

func (f *flowEndReason) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(uint16(reason), context, f)
}

func init() {
	flows.RegisterStandardFeature("flowEndReason", flows.FlowFeature, func() flows.Feature { return &flowEndReason{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowEndNanoseconds struct {
	flows.BaseFeature
	lastTime flows.DateTimeNanoseconds
}

func (f *flowEndNanoseconds) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.lastTime = context.When()
}

func (f *flowEndNanoseconds) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.lastTime, context, f)
}

func init() {
	flows.RegisterStandardFeature("flowEndNanoseconds", flows.FlowFeature, func() flows.Feature { return &flowEndNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowEndMilliseconds", flows.FlowFeature, func() flows.Feature { return &flowEndNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowEndMicroseconds", flows.FlowFeature, func() flows.Feature { return &flowEndNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowEndSeconds", flows.FlowFeature, func() flows.Feature { return &flowEndNanoseconds{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowStartNanoseconds struct {
	flows.BaseFeature
}

func (f *flowStartNanoseconds) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.SetValue(context.When(), context, f)
}

func init() {
	flows.RegisterStandardFeature("flowStartNanoseconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowStartMicroseconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowStartMilliseconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowStartSeconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowDirection struct {
	flows.BaseFeature
}

func (f *flowDirection) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(new.(packet.Buffer).LowToHigh(), context, f)
	}
}

type flowDirectionPacket struct {
	flows.BaseFeature
}

func (f *flowDirectionPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(context.Forward(), context, f)
}

func init() {
	flows.RegisterStandardFeature("flowDirection", flows.FlowFeature, func() flows.Feature { return &flowDirection{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowDirection", flows.PacketFeature, func() flows.Feature { return &flowDirectionPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowID struct {
	flows.BaseFeature
}

func (f *flowID) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		// flowId is a per table flow counter ored with the tableId in the highest byte
		flow := context.Flow()
		flowid := flow.ID() & 0x00FFFFFFFFFFFFFF
		flowid |= uint64(flow.Table().ID()) << 56
		f.SetValue(flowid, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("flowId", flows.FlowFeature, func() flows.Feature { return &flowID{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type packetTotalCount struct {
	flows.BaseFeature
	count uint64
}

func (f *packetTotalCount) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *packetTotalCount) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.count++
}

func (f *packetTotalCount) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func init() {
	flows.RegisterStandardFeature("packetTotalCount", flows.FlowFeature, func() flows.Feature { return &packetTotalCount{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowDurationNanoseconds struct {
	flows.BaseFeature
	start    flows.DateTimeNanoseconds
	lastTime flows.DateTimeNanoseconds
}

func (f *flowDurationNanoseconds) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.start = context.When()
	f.lastTime = context.When()
}

func (f *flowDurationNanoseconds) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.lastTime = context.When()
}

func (f *flowDurationNanoseconds) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.lastTime-f.start, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("flowDurationNanoseconds", "flow duration in nanoseconds", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &flowDurationNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardCompositeFeature("flowDurationMicroseconds", "divide", "flowDurationNanoseconds", 1000)
	flows.RegisterStandardCompositeFeature("flowDurationMilliseconds", "divide", "flowDurationNanoseconds", 1000000)
}

////////////////////////////////////////////////////////////////////////////////
