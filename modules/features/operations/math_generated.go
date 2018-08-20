package operations

// Created by gen_math.go, don't edit manually!
// Generated at 2018-08-20 14:07:29.66653769 +0200 CEST m=+0.001376418

import (
	"github.com/CN-TU/go-flows/flows"
	"math"
)

type floorPacket struct {
	flows.BaseFeature
}

func (f *floorPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(math.Floor(flows.ToFloat(new)), context, f)
}

type floorFlow struct {
	flows.BaseFeature
}

func (f *floorFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(math.Floor(flows.ToFloat(new)), context, f)
	}
}

func init() {
	flows.RegisterFunction("floor", "returns ⌊a⌋", flows.PacketFeature, func() flows.Feature { return &floorPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("floor", "returns ⌊a⌋", flows.FlowFeature, func() flows.Feature { return &floorFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type ceilPacket struct {
	flows.BaseFeature
}

func (f *ceilPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(math.Ceil(flows.ToFloat(new)), context, f)
}

type ceilFlow struct {
	flows.BaseFeature
}

func (f *ceilFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(math.Ceil(flows.ToFloat(new)), context, f)
	}
}

func init() {
	flows.RegisterFunction("ceil", "returns ⌈a⌉", flows.PacketFeature, func() flows.Feature { return &ceilPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("ceil", "returns ⌈a⌉", flows.FlowFeature, func() flows.Feature { return &ceilFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type logPacket struct {
	flows.BaseFeature
}

func (f *logPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(math.Log(flows.ToFloat(new)), context, f)
}

type logFlow struct {
	flows.BaseFeature
}

func (f *logFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(math.Log(flows.ToFloat(new)), context, f)
	}
}

func init() {
	flows.RegisterFunction("log", "returns log(a)", flows.PacketFeature, func() flows.Feature { return &logPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("log", "returns log(a)", flows.FlowFeature, func() flows.Feature { return &logFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type expPacket struct {
	flows.BaseFeature
}

func (f *expPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(math.Exp(flows.ToFloat(new)), context, f)
}

type expFlow struct {
	flows.BaseFeature
}

func (f *expFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(math.Exp(flows.ToFloat(new)), context, f)
	}
}

func init() {
	flows.RegisterFunction("exp", "returns exp(a)", flows.PacketFeature, func() flows.Feature { return &expPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("exp", "returns exp(a)", flows.FlowFeature, func() flows.Feature { return &expFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type addPacket struct {
	flows.MultiBasePacketFeature
}

func (f *addPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) + b.(uint64)
	case flows.IntType:
		result = a.(int64) + b.(int64)
	case flows.FloatType:
		result = a.(float64) + b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

type addFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *addFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues()

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) + b.(uint64)
	case flows.IntType:
		result = a.(int64) + b.(int64)
	case flows.FloatType:
		result = a.(float64) + b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

func init() {
	flows.RegisterFunction("add", "returns a + b", flows.PacketFeature, func() flows.Feature { return &addPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("add", "returns a + b", flows.FlowFeature, func() flows.Feature { return &addFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type subtractPacket struct {
	flows.MultiBasePacketFeature
}

func (f *subtractPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) - b.(uint64)
	case flows.IntType:
		result = a.(int64) - b.(int64)
	case flows.FloatType:
		result = a.(float64) - b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

type subtractFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *subtractFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues()

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) - b.(uint64)
	case flows.IntType:
		result = a.(int64) - b.(int64)
	case flows.FloatType:
		result = a.(float64) - b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

func init() {
	flows.RegisterFunction("subtract", "returns a - b", flows.PacketFeature, func() flows.Feature { return &subtractPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("subtract", "returns a - b", flows.FlowFeature, func() flows.Feature { return &subtractFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type multiplyPacket struct {
	flows.MultiBasePacketFeature
}

func (f *multiplyPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) * b.(uint64)
	case flows.IntType:
		result = a.(int64) * b.(int64)
	case flows.FloatType:
		result = a.(float64) * b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

type multiplyFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *multiplyFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues()

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) * b.(uint64)
	case flows.IntType:
		result = a.(int64) * b.(int64)
	case flows.FloatType:
		result = a.(float64) * b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

func init() {
	flows.RegisterFunction("multiply", "returns a * b", flows.PacketFeature, func() flows.Feature { return &multiplyPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("multiply", "returns a * b", flows.FlowFeature, func() flows.Feature { return &multiplyFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type dividePacket struct {
	flows.MultiBasePacketFeature
}

func (f *dividePacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) / b.(uint64)
	case flows.IntType:
		result = a.(int64) / b.(int64)
	case flows.FloatType:
		result = a.(float64) / b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

type divideFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *divideFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues()

	dst, fl, a, b := flows.UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case flows.UIntType:
		result = a.(uint64) / b.(uint64)
	case flows.IntType:
		result = a.(int64) / b.(int64)
	case flows.FloatType:
		result = a.(float64) / b.(float64)
	}
	f.SetValue(flows.FixType(result, dst), context, f)
}

func init() {
	flows.RegisterFunction("divide", "returns a / b", flows.PacketFeature, func() flows.Feature { return &dividePacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterFunction("divide", "returns a / b", flows.FlowFeature, func() flows.Feature { return &divideFlow{} }, flows.FlowFeature, flows.FlowFeature)
}
