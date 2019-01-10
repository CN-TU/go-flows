package operations

import "github.com/CN-TU/go-flows/flows"

type forward struct {
	flows.EmptyBaseFeature
}

func (f *forward) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if context.Forward() {
		f.Emit(new, context, f)
	}
}

func init() {
	flows.RegisterFunction("forward", "select only packets in the forward direction", flows.Selection, func() flows.Feature { return &forward{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type backward struct {
	flows.EmptyBaseFeature
}

func (f *backward) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if !context.Forward() {
		f.Emit(new, context, f)
	}
}

func init() {
	flows.RegisterFunction("backward", "select only packets in the backward direction", flows.Selection, func() flows.Feature { return &backward{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
