package flows

import (
	"bytes"
	"fmt"
	"math"
	"reflect"

	"github.com/CN-TU/go-ipfix"
)

type constantFeature struct {
	value interface{}
}

func (f *constantFeature) Event(interface{}, *EventContext, interface{})    {}
func (f *constantFeature) FinishEvent()                                     {}
func (f *constantFeature) Value() interface{}                               { return f.value }
func (f *constantFeature) SetValue(interface{}, *EventContext, interface{}) {}
func (f *constantFeature) Start(*EventContext)                              {}
func (f *constantFeature) Stop(FlowEndReason, *EventContext)                {}
func (f *constantFeature) Variant() int                                     { return NoVariant }
func (f *constantFeature) Emit(interface{}, *EventContext, interface{})     {}
func (f *constantFeature) setDependent([]Feature)                           {}
func (f *constantFeature) SetArguments([]Feature)                           {}
func (f *constantFeature) IsConstant() bool                                 { return true }
func (f *constantFeature) setRecord(*record)                                {}

var _ Feature = (*constantFeature)(nil)

func newConstantMetaFeature(value interface{}) featureMaker {
	var t ipfix.Type
	switch cv := value.(type) {
	case bool:
		t = ipfix.BooleanType
	case float64:
		t = ipfix.Float64Type
	case int:
		value = int64(cv)
		t = ipfix.Signed64Type
	case int64:
		t = ipfix.Signed64Type
	case uint:
		value = uint64(cv)
		t = ipfix.Unsigned64Type
	case uint64:
		t = ipfix.Unsigned64Type
	default:
		panic(fmt.Sprint("Can't create constant of type ", reflect.TypeOf(value)))
	}
	feature := &constantFeature{value}
	return featureMaker{
		ret:  Const,
		make: func() Feature { return feature },
		ie:   ipfix.NewInformationElement(fmt.Sprintf("_const{%v}", value), 0, 0, t, 0),
	}
}

////////////////////////////////////////////////////////////////////////////////

type selectF struct {
	EmptyBaseFeature
	sel bool
}

func (f *selectF) Start(*EventContext) { f.sel = false }

func (f *selectF) Event(new interface{}, context *EventContext, src interface{}) {
	/* If src is not nil we got an event from the argument -> Store the boolean value (This always happens before events from the flow)
	   otherwise we have an event from the flow -> forward it in case we should and reset sel
	*/
	if src != nil {
		f.sel = new.(bool)
	} else {
		if f.sel {
			f.Emit(new, context, nil)
			f.sel = false
		}
	}
}

type selectS struct {
	EmptyBaseFeature
	start, stop, current int64
}

func (f *selectS) SetArguments(arguments []Feature) {
	f.start = ToInt(arguments[0].Value())
	f.stop = ToInt(arguments[1].Value())
}
func (f *selectS) Start(*EventContext) { f.current = 0 }

func (f *selectS) Event(new interface{}, context *EventContext, src interface{}) {
	if f.current >= f.start && f.current < f.stop {
		f.Emit(new, context, nil)
	}
	f.current++
}

func init() {
	RegisterFunction("select", Selection, func() Feature { return &selectF{} }, PacketFeature)
	RegisterFunction("select_slice", Selection, func() Feature { return &selectS{} }, Const, Const)
	RegisterFunction("select_slice", Selection, func() Feature { return &selectS{} }, Const, Const, Selection)
}

////////////////////////////////////////////////////////////////////////////////

type slice struct {
	BaseFeature
	start, stop, current int64
}

func (f *slice) SetArguments(arguments []Feature) {
	f.start = ToInt(arguments[0].Value())
	f.stop = ToInt(arguments[1].Value())
}
func (f *slice) Start(*EventContext) { f.current = 0 }

func (f *slice) Event(new interface{}, context *EventContext, src interface{}) {
	if f.current >= f.start && f.current < f.stop {
		f.SetValue(new, context, f)
	}
	f.current++
}

func resolveSlice(args []ipfix.InformationElement) ipfix.InformationElement {
	return args[2]
}

func init() {
	RegisterCustomFunction("slice", resolveSlice, PacketFeature, func() Feature { return &slice{} }, Const, Const, PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type get struct {
	BaseFeature
	index, current int64
}

func (f *get) SetArguments(arguments []Feature) {
	f.index = ToInt(arguments[0].Value())
}
func (f *get) Start(*EventContext) { f.current = 0 }

func (f *get) Event(new interface{}, context *EventContext, src interface{}) {
	if f.current == f.index {
		f.SetValue(new, context, f)
	}
	f.current++
}

func resolveGet(args []ipfix.InformationElement) ipfix.InformationElement {
	return args[1]
}

func init() {
	RegisterCustomFunction("get", resolveGet, FlowFeature, func() Feature { return &get{} }, Const, PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

//apply and map pseudofeatures
func init() {
	RegisterFunction("apply", FlowFeature, nil, FlowFeature, Selection)
	RegisterFunction("map", PacketFeature, nil, PacketFeature, Selection)
}

////////////////////////////////////////////////////////////////////////////////

type count struct {
	BaseFeature
	count uint64
}

func (f *count) Start(context *EventContext) {
	f.count = 0
}

func (f *count) Event(new interface{}, context *EventContext, src interface{}) {
	f.count++
}

func (f *count) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.count, context, f)
}

func init() {
	RegisterTemporaryFeature("count", ipfix.Unsigned64Type, 0, FlowFeature, func() Feature { return &count{} }, Selection)
}

////////////////////////////////////////////////////////////////////////////////

type mean struct {
	BaseFeature
	total float64
	count uint64
}

func (f *mean) Start(context *EventContext) {
	f.total = 0
	f.count = 0
}

func (f *mean) Event(new interface{}, context *EventContext, src interface{}) {
	f.total += ToFloat(new)
	f.count++
}

func (f *mean) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.total/float64(f.count), context, f)
}

func init() {
	RegisterTypedFunction("mean", ipfix.Float64Type, 0, FlowFeature, func() Feature { return &mean{} }, PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type min struct {
	BaseFeature
}

func (f *min) Event(new interface{}, context *EventContext, src interface{}) {
	if f.value == nil {
		f.value = new
	} else {
		_, fl, a, b := UpConvert(new, f.value)
		switch fl {
		case UIntType:
			if a.(uint64) < b.(uint64) {
				f.value = new
			}
		case IntType:
			if a.(int64) < b.(int64) {
				f.value = new
			}
		case FloatType:
			if a.(float64) < b.(float64) {
				f.value = new
			}
		}
	}
}

func (f *min) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterFunction("min", FlowFeature, func() Feature { return &min{} }, PacketFeature)
	RegisterFunction("minimum", FlowFeature, func() Feature { return &min{} }, PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type max struct {
	BaseFeature
}

func (f *max) Event(new interface{}, context *EventContext, src interface{}) {
	if f.value == nil {
		f.value = new
	} else {
		_, fl, a, b := UpConvert(new, f.value)
		switch fl {
		case UIntType:
			if a.(uint64) > b.(uint64) {
				f.value = new
			}
		case IntType:
			if a.(int64) > b.(int64) {
				f.value = new
			}
		case FloatType:
			if a.(float64) > b.(float64) {
				f.value = new
			}
		}
	}
}

func (f *max) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterFunction("max", FlowFeature, func() Feature { return &max{} }, PacketFeature)
	RegisterFunction("maximum", FlowFeature, func() Feature { return &max{} }, PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type less struct {
	MultiBaseFeature
}

func (f *less) Event(new interface{}, context *EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}

	_, fl, a, b := UpConvert(values[0], values[1])
	switch fl {
	case UIntType:
		f.SetValue(a.(uint64) < b.(uint64), context, f)
	case IntType:
		f.SetValue(a.(int64) < b.(int64), context, f)
	case FloatType:
		f.SetValue(a.(float64) < b.(float64), context, f)
	}
}

func init() {
	RegisterTemporaryFeature("less", ipfix.BooleanType, 0, MatchType, func() Feature { return &less{} }, MatchType, MatchType)
}

////////////////////////////////////////////////////////////////////////////////

type geq struct {
	MultiBaseFeature
}

func (f *geq) Event(new interface{}, context *EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	_, fl, a, b := UpConvert(values[0], values[1])
	switch fl {
	case UIntType:
		f.SetValue(a.(uint64) >= b.(uint64), context, f)
	case IntType:
		f.SetValue(a.(int64) >= b.(int64), context, f)
	case FloatType:
		f.SetValue(a.(float64) >= b.(float64), context, f)
	}
}

func init() {
	RegisterTemporaryFeature("geq", ipfix.BooleanType, 0, MatchType, func() Feature { return &geq{} }, MatchType, MatchType)
}

////////////////////////////////////////////////////////////////////////////////

type accumulate struct {
	MultiBaseFeature
	vector []interface{}
}

func (f *accumulate) Start(context *EventContext) {
	f.vector = make([]interface{}, 0)
}

func (f *accumulate) Stop(reason FlowEndReason, context *EventContext) {
	if len(f.vector) != 0 {
		f.SetValue(f.vector, context, f)
	}
}

func (f *accumulate) Event(new interface{}, context *EventContext, src interface{}) {
	f.vector = append(f.vector, new)
}

func resolveAccumulate(args []ipfix.InformationElement) ipfix.InformationElement {
	return ipfix.NewBasicList("accumulate", args[0], 0)
}

//FIXME: this has a bad name
func init() {
	RegisterCustomFunction("accumulate", resolveAccumulate, MatchType, func() Feature { return &accumulate{} }, PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type concatenate struct {
	BaseFeature
	buffer *bytes.Buffer
}

func (f *concatenate) Start(context *EventContext) {
	f.buffer = new(bytes.Buffer)
}

func (f *concatenate) Event(new interface{}, context *EventContext, src interface{}) {
	fmt.Fprint(f.buffer, new)
}

func (f *concatenate) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.buffer.Bytes(), context, f)
}

func init() {
	RegisterTemporaryFeature("concatenate", ipfix.OctetArrayType, 0, FlowFeature, func() Feature { return &concatenate{} }, PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type logFlow struct {
	BaseFeature
}

func (f *logFlow) Event(new interface{}, context *EventContext, src interface{}) {
	f.value = math.Log(ToFloat(new))
}

func (f *logFlow) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterTypedFunction("log", ipfix.Float64Type, 0, FlowFeature, func() Feature { return &logFlow{} }, FlowFeature)
}

////////////////////////////////////////////////////////////////////////////////

type divideFlow struct {
	MultiBaseFeature
}

func (f *divideFlow) Event(new interface{}, context *EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	dst, fl, a, b := UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case UIntType:
		result = a.(uint64) / b.(uint64)
	case IntType:
		result = a.(int64) / b.(int64)
	case FloatType:
		result = a.(float64) / b.(float64)
	}
	f.value = FixType(result, dst)
}

func (f *divideFlow) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterFunction("divide", FlowFeature, func() Feature { return &divideFlow{} }, FlowFeature, FlowFeature)
}

////////////////////////////////////////////////////////////////////////////////

type multiplyFlow struct {
	MultiBaseFeature
}

func (f *multiplyFlow) Event(new interface{}, context *EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	dst, fl, a, b := UpConvert(values[0], values[1])
	var result interface{}
	switch fl {
	case UIntType:
		result = a.(uint64) * b.(uint64)
	case IntType:
		result = a.(int64) * b.(int64)
	case FloatType:
		result = a.(float64) * b.(float64)
	}
	f.value = FixType(result, dst)
}

func (f *multiplyFlow) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterFunction("multiply", FlowFeature, func() Feature { return &multiplyFlow{} }, FlowFeature, FlowFeature)
}

////////////////////////////////////////////////////////////////////////////////

type packetTotalCount struct {
	BaseFeature
	count uint64
}

func (f *packetTotalCount) Start(context *EventContext) {
	f.count = 0
}

func (f *packetTotalCount) Event(new interface{}, context *EventContext, src interface{}) {
	f.count++
}

func (f *packetTotalCount) Stop(reason FlowEndReason, context *EventContext) {
	f.SetValue(f.count, context, f)
}

func init() {
	RegisterStandardFeature("packetTotalCount", FlowFeature, func() Feature { return &packetTotalCount{} }, RawPacket)
}
