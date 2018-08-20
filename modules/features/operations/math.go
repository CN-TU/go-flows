package operations

import "github.com/CN-TU/go-flows/flows"

type addPacketFlow struct {
	flows.BaseFeature
	current interface{}
}

func (f *addPacketFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.current == nil {
		f.current = new
		return
	}
	dst, fl, a, b := flows.UpConvert(f.current, new)
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) + b.(uint64)
	case flows.IntType:
		result = a.(int64) + b.(int64)
	case flows.FloatType:
		result = a.(float64) + b.(float64)
	}
	f.current = flows.FixType(result, dst)
}

func (f *addPacketFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if f.current != nil {
		f.SetValue(f.current, context, f)
	}
}

func init() {
	flows.RegisterFunction("add", "returns ∑ a", flows.FlowFeature, func() flows.Feature { return &addPacketFlow{} }, flows.PacketFeature)
	flows.RegisterFunction("sum", "returns ∑ a", flows.FlowFeature, func() flows.Feature { return &addPacketFlow{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type multiplyPacketFlow struct {
	flows.BaseFeature
	current interface{}
}

func (f *multiplyPacketFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.current == nil {
		f.current = new
		return
	}
	dst, fl, a, b := flows.UpConvert(f.current, new)
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) * b.(uint64)
	case flows.IntType:
		result = a.(int64) * b.(int64)
	case flows.FloatType:
		result = a.(float64) * b.(float64)
	}
	f.current = flows.FixType(result, dst)
}

func (f *multiplyPacketFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if f.current != nil {
		f.SetValue(f.current, context, f)
	}
}

func init() {
	flows.RegisterFunction("multiply", "returns ∏ a", flows.FlowFeature, func() flows.Feature { return &multiplyPacketFlow{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////
