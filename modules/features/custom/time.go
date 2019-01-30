package custom

import (
	"github.com/CN-TU/go-flows/flows"
	ipfix "github.com/CN-TU/go-ipfix"
)

type _interPacketTimeNanoseconds struct {
	flows.BaseFeature
	time flows.DateTimeNanoseconds
}

func (f *_interPacketTimeNanoseconds) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.time = 0
}

func (f *_interPacketTimeNanoseconds) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.time != 0 {
		f.SetValue(int64(context.When())-int64(f.time), context, f)
	}
	f.time = context.When()
}

func init() {
	flows.RegisterTemporaryFeature("_interPacketTimeNanoseconds", "time difference between consecutive packets", ipfix.Signed64Type, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryCompositeFeature("_interPacketTimeMicroseconds", "time difference between consecutive packets", ipfix.Signed64Type, 0, "divide", "_interPacketTimeNanoseconds", 1000)
	flows.RegisterTemporaryCompositeFeature("_interPacketTimeMilliseconds", "time difference between consecutive packets", ipfix.Signed64Type, 0, "divide", "_interPacketTimeNanoseconds", 1000000)
	flows.RegisterTemporaryCompositeFeature("_interPacketTimeSeconds", "time difference between consecutive packets", ipfix.Signed32Type, 0, "divide", "_interPacketTimeNanoseconds", 1000000000)
}

////////////////////////////////////////////////////////////////////////////////

type flowExportNanoseconds struct {
	flows.BaseFeature
}

func (f *flowExportNanoseconds) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(context.When(), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("__flowExportNanoseconds", "Export time in nanoseconds", ipfix.DateTimeNanosecondsType, 0, flows.FlowFeature, func() flows.Feature { return &flowExportNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("__flowExportMilliseconds", "Export time in milliseconds", ipfix.DateTimeMillisecondsType, 0, flows.FlowFeature, func() flows.Feature { return &flowExportNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("__flowExportMicroseconds", "Export time in microseconds", ipfix.DateTimeMicrosecondsType, 0, flows.FlowFeature, func() flows.Feature { return &flowExportNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("__flowExportSeconds", "Export time in seconds", ipfix.DateTimeSecondsType, 0, flows.FlowFeature, func() flows.Feature { return &flowExportNanoseconds{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
