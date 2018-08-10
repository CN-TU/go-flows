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
	var time int64
	if f.time != 0 {
		time = int64(context.When()) - int64(f.time)
	}
	f.time = context.When()
	f.SetValue(time, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_interPacketTimeNanoseconds", ipfix.Signed64Type, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryCompositeFeature("_interPacketTimeMicroseconds", ipfix.Signed64Type, 0, "divide", "_interPacketTimeNanoseconds", 1000)
	flows.RegisterTemporaryCompositeFeature("_interPacketTimeMilliseconds", ipfix.Signed64Type, 0, "divide", "_interPacketTimeNanoseconds", 1000000)
	flows.RegisterTemporaryCompositeFeature("_interPacketTimeSeconds", ipfix.Signed32Type, 0, "divide", "_interPacketTimeNanoseconds", 1000000000)
}

////////////////////////////////////////////////////////////////////////////////
