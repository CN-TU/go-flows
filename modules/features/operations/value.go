package operations

import (
	"github.com/CN-TU/go-flows/flows"
	ipfix "github.com/CN-TU/go-ipfix"
)

type get struct {
	flows.BaseFeature
	index, current int64
}

func (f *get) SetArguments(arguments []flows.Feature) {
	f.index = flows.ToInt(arguments[0].Value())
}
func (f *get) Start(ec *flows.EventContext) {
	f.BaseFeature.Start(ec)
	f.current = 0
}

func (f *get) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.current == f.index {
		f.SetValue(new, context, f)
	}
	f.current++
}

func resolveGet(args []ipfix.InformationElement) ipfix.InformationElement {
	return args[1]
}

func init() {
	flows.RegisterCustomFunction("get", "gets the <value>-th element of the second argument; indexing is like in Python", resolveGet, flows.FlowFeature, func() flows.Feature { return &get{} }, flows.Const, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type count struct {
	flows.BaseFeature
	count uint64
}

func (f *count) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *count) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.count++
}

func (f *count) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("count", "returns number of selected objects", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &count{} }, flows.Selection)
}

////////////////////////////////////////////////////////////////////////////////

type mean struct {
	flows.BaseFeature
	total float64
	count uint64
}

func (f *mean) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.total = 0
	f.count = 0
}

func (f *mean) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.total += flows.ToFloat(new)
	f.count++
}

func (f *mean) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.total/float64(f.count), context, f)
}

func init() {
	flows.RegisterTypedFunction("mean", "returns mean of input", ipfix.Float64Type, 0, flows.FlowFeature, func() flows.Feature { return &mean{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type min struct {
	flows.BaseFeature
	current interface{}
}

func (f *min) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.current == nil {
		f.current = new
	} else {
		_, fl, a, b := flows.UpConvert(new, f.current)
		switch fl {
		case flows.UIntType:
			if a.(uint64) < b.(uint64) {
				f.current = new
			}
		case flows.IntType:
			if a.(int64) < b.(int64) {
				f.current = new
			}
		case flows.FloatType:
			if a.(float64) < b.(float64) {
				f.current = new
			}
		}
	}
}

func (f *min) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.current, context, f)
}

func init() {
	flows.RegisterFunction("min", "returns min of input", flows.FlowFeature, func() flows.Feature { return &min{} }, flows.PacketFeature)
	flows.RegisterFunction("minimum", "returns min of input", flows.FlowFeature, func() flows.Feature { return &min{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type max struct {
	flows.BaseFeature
	current interface{}
}

func (f *max) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.current == nil {
		f.current = new
	} else {
		_, fl, a, b := flows.UpConvert(new, f.current)
		switch fl {
		case flows.UIntType:
			if a.(uint64) > b.(uint64) {
				f.current = new
			}
		case flows.IntType:
			if a.(int64) > b.(int64) {
				f.current = new
			}
		case flows.FloatType:
			if a.(float64) > b.(float64) {
				f.current = new
			}
		}
	}
}

func (f *max) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.current, context, f)
}

func init() {
	flows.RegisterFunction("max", "returns max of input", flows.FlowFeature, func() flows.Feature { return &max{} }, flows.PacketFeature)
	flows.RegisterFunction("maximum", "returns max of input", flows.FlowFeature, func() flows.Feature { return &max{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////
