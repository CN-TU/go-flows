package iana

import (
	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
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
}

func (f *flowEndNanoseconds) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(context.When(), context, f)
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
		f.SetValue(new.(packet.Buffer).Forward(), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("flowDirection", flows.FlowFeature, func() flows.Feature { return &flowDirection{} }, flows.RawPacket)
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
