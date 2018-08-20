package operations

import (
	"math"
	"sort"

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

type stdev struct {
	flows.BaseFeature
	vector []interface{}
	total  float64
	count  uint64
	mean   float64
	sd     float64
}

func (f *stdev) Start(context *flows.EventContext) {
	f.vector = make([]interface{}, 0)
	f.total = 0
	f.count = 0
	f.mean = 0
	f.sd = 0
}

func (f *stdev) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if len(f.vector) != 0 {
		f.mean = f.total / float64(f.count)
		for j := 0; j < len(f.vector); j++ {
			f.sd += math.Pow(flows.ToFloat(f.vector[j])-f.mean, 2)
		}

		f.sd = math.Sqrt(f.sd / flows.ToFloat(len(f.vector)))
		f.SetValue(f.sd, context, f)
	}
}

func (f *stdev) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.vector = append(f.vector, new)
	f.total += flows.ToFloat(new)
	f.count++
}

func init() {
	flows.RegisterTypedFunction("stdev", "", ipfix.Float64Type, 0, flows.FlowFeature, func() flows.Feature { return &stdev{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type variance struct {
	flows.BaseFeature
	vector []interface{}
	total  float64
	count  uint64
	mean   float64
	sd     float64
}

func (f *variance) Start(context *flows.EventContext) {
	f.vector = make([]interface{}, 0)
	f.total = 0
	f.count = 0
	f.mean = 0
	f.sd = 0
}

func (f *variance) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if len(f.vector) != 0 {
		f.mean = f.total / float64(f.count)
		for j := 0; j < len(f.vector); j++ {
			f.sd += math.Pow(flows.ToFloat(f.vector[j])-f.mean, 2)
		}

		f.sd = math.Sqrt(f.sd / flows.ToFloat(len(f.vector)))
		f.SetValue(math.Pow(f.sd, 2), context, f)
	}
}

func (f *variance) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.vector = append(f.vector, new)
	f.total += flows.ToFloat(new)
	f.count++
}

func init() {
	flows.RegisterTypedFunction("variance", "", ipfix.Float64Type, 0, flows.FlowFeature, func() flows.Feature { return &variance{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type median struct {
	flows.BaseFeature
	vector []float64
	med    float64
}

func (f *median) Start(context *flows.EventContext) {
	f.vector = make([]float64, 0)
	f.med = 0
}

func (f *median) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	sort.Float64s(f.vector)
	if len(f.vector) == 0 {
		f.SetValue(0, context, f)
	} else if len(f.vector)%2 == 0 {
		f.med = float64((f.vector[len(f.vector)/2-1] + f.vector[len(f.vector)/2]) / 2)
	} else {
		f.med = float64(f.vector[(len(f.vector)-1)/2])
	}

	f.SetValue(f.med, context, f)

}

func (f *median) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.vector = append(f.vector, flows.ToFloat(new))
}

func init() {
	flows.RegisterTypedFunction("median", "", ipfix.Float64Type, 0, flows.FlowFeature, func() flows.Feature { return &median{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type mode struct {
	flows.BaseFeature
	vector map[float64]uint64
	max    uint64
	m      float64
}

func (f *mode) Start(context *flows.EventContext) {
	f.vector = make(map[float64]uint64, 0)
	f.max = 0
	f.m = 0
}

func (f *mode) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	for key, value := range f.vector {
		if value > f.max {
			f.max = value
			f.m = key
		}
	}
	f.SetValue(f.m, context, f)

}

func (f *mode) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.vector[flows.ToFloat(new)] += 1
}

func init() {
	flows.RegisterTypedFunction("mode", "", ipfix.Float64Type, 0, flows.FlowFeature, func() flows.Feature { return &mode{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////
