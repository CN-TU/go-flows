package operations

// Created by gen_logic.go, don't edit manually!
// Generated at 2019-05-03 16:04:30.469237292 +0200 CEST m=+0.000798366

import (
	"github.com/CN-TU/go-flows/flows"
	ipfix "github.com/CN-TU/go-ipfix"
)

type geqPacket struct {
	flows.MultiBasePacketFeature
}

func (f *geqPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) >= b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) >= b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) >= b.(float64), context, f)
	}
}

type geqFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *geqFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues(context)

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) >= b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) >= b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) >= b.(float64), context, f)
	}
}

func init() {
	flows.RegisterTypedFunction("geq", "returns true if a >= b", ipfix.BooleanType, 0, flows.PacketFeature, func() flows.Feature { return &geqPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterTypedFunction("geq", "returns true if a >= b", ipfix.BooleanType, 0, flows.FlowFeature, func() flows.Feature { return &geqFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type leqPacket struct {
	flows.MultiBasePacketFeature
}

func (f *leqPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) <= b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) <= b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) <= b.(float64), context, f)
	}
}

type leqFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *leqFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues(context)

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) <= b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) <= b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) <= b.(float64), context, f)
	}
}

func init() {
	flows.RegisterTypedFunction("leq", "returns true if a <= b", ipfix.BooleanType, 0, flows.PacketFeature, func() flows.Feature { return &leqPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterTypedFunction("leq", "returns true if a <= b", ipfix.BooleanType, 0, flows.FlowFeature, func() flows.Feature { return &leqFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type lessPacket struct {
	flows.MultiBasePacketFeature
}

func (f *lessPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) < b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) < b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) < b.(float64), context, f)
	}
}

type lessFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *lessFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues(context)

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) < b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) < b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) < b.(float64), context, f)
	}
}

func init() {
	flows.RegisterTypedFunction("less", "returns true if a < b", ipfix.BooleanType, 0, flows.PacketFeature, func() flows.Feature { return &lessPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterTypedFunction("less", "returns true if a < b", ipfix.BooleanType, 0, flows.FlowFeature, func() flows.Feature { return &lessFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type greaterPacket struct {
	flows.MultiBasePacketFeature
}

func (f *greaterPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) > b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) > b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) > b.(float64), context, f)
	}
}

type greaterFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *greaterFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues(context)

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) > b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) > b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) > b.(float64), context, f)
	}
}

func init() {
	flows.RegisterTypedFunction("greater", "returns true if a > b", ipfix.BooleanType, 0, flows.PacketFeature, func() flows.Feature { return &greaterPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterTypedFunction("greater", "returns true if a > b", ipfix.BooleanType, 0, flows.FlowFeature, func() flows.Feature { return &greaterFlow{} }, flows.FlowFeature, flows.FlowFeature)
}

type equalPacket struct {
	flows.MultiBasePacketFeature
}

func (f *equalPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) == b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) == b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) == b.(float64), context, f)
	}
}

type equalFlow struct {
	flows.MultiBaseFlowFeature
}

func (f *equalFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	values := f.GetValues(context)

	_, fl, a, b := flows.UpConvert(values[0], values[1])
	switch fl {
	case flows.UIntType:
		f.SetValue(a.(uint64) == b.(uint64), context, f)
	case flows.IntType:
		f.SetValue(a.(int64) == b.(int64), context, f)
	case flows.FloatType:
		f.SetValue(a.(float64) == b.(float64), context, f)
	}
}

func init() {
	flows.RegisterTypedFunction("equal", "returns true if a == b", ipfix.BooleanType, 0, flows.PacketFeature, func() flows.Feature { return &equalPacket{} }, flows.PacketFeature, flows.PacketFeature)
	flows.RegisterTypedFunction("equal", "returns true if a == b", ipfix.BooleanType, 0, flows.FlowFeature, func() flows.Feature { return &equalFlow{} }, flows.FlowFeature, flows.FlowFeature)
}
