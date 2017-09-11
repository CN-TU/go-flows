package flows

import "bytes"
import "fmt"
import "reflect"
import "math"

type constantFeature struct {
	value interface{}
	t     string
}

func (f *constantFeature) setDependent([]Feature)                          {}
func (f *constantFeature) getDependent() []Feature                         { return nil }
func (f *constantFeature) SetArguments([]Feature)                          {}
func (f *constantFeature) Event(interface{}, EventContext, interface{})    {}
func (f *constantFeature) FinishEvent()                                    {}
func (f *constantFeature) Value() interface{}                              { return f.value }
func (f *constantFeature) SetValue(interface{}, EventContext, interface{}) {}
func (f *constantFeature) Start(EventContext)                              {}
func (f *constantFeature) Stop(FlowEndReason, EventContext)                {}
func (f *constantFeature) Type() string                                    { return f.t }
func (f *constantFeature) BaseType() string                                { return f.t }
func (f *constantFeature) setBaseType(string)                              {}
func (f *constantFeature) getBaseFeature() *BaseFeature                    { return nil }
func (f *constantFeature) IsConstant() bool                                { return true }
func (f *constantFeature) Emit(interface{}, EventContext, interface{})     {}

func newConstantMetaFeature(value interface{}) metaFeature {
	var f interface{}
	switch value.(type) {
	case bool:
		f = value
	case float64:
		f = Float64(value.(float64))
	case int64:
		f = Signed64(value.(int64))
	default:
		panic(fmt.Sprint("Can't create constant of type ", reflect.TypeOf(value)))
	}
	t := fmt.Sprintf("___const{%v}", f)
	feature := &constantFeature{f, t}
	return metaFeature{FeatureCreator{featureTypeAny, func() Feature { return feature }, nil}, t}
}

////////////////////////////////////////////////////////////////////////////////

type selectF struct {
	EmptyBaseFeature
	sel bool
}

func (f *selectF) Start(EventContext) { f.sel = false }
func (f *selectF) Type() string       { return "select" }
func (f *selectF) BaseType() string   { return "select" }

func (f *selectF) Event(new interface{}, context EventContext, src interface{}) {
	/* If src is not nil we got an event from the argument -> Store the boolean value (This always happens before events from the flow)
	   otherwise we have an event from the flow -> forward it in case we should and reset sel
	*/
	if src != nil {
		f.sel = new.(bool)
	} else {
		if f.sel {
			for _, v := range f.dependent {
				v.Event(new, context, nil) // is it ok to use nil as source? (we are faking flow source here)
			}
			f.sel = false
		}
	}
}

type selectS struct {
	EmptyBaseFeature
	start, stop, current int
}

func (f *selectS) SetArguments(arguments []Feature) {
	f.start = int(arguments[0].Value().(Number).ToInt())
	f.stop = int(arguments[1].Value().(Number).ToInt())
}
func (f *selectS) Start(EventContext) { f.current = 0 }
func (f *selectS) Type() string       { return "select" }
func (f *selectS) BaseType() string   { return "select" }

func (f *selectS) Event(new interface{}, context EventContext, src interface{}) {
	if f.current >= f.start && f.current < f.stop {
		for _, v := range f.dependent {
			v.Event(new, context, nil) // is it ok to use nil as source? (we are faking flow source here)
		}
	}
	f.current++
}

func init() {
	RegisterFeature("select", []FeatureCreator{
		{FeatureTypeSelection, func() Feature { return &selectF{} }, []FeatureType{FeatureTypePacket}},
	})
	RegisterFeature("select_slice", []FeatureCreator{
		{FeatureTypeSelection, func() Feature { return &selectS{} }, []FeatureType{featureTypeAny, featureTypeAny}},
	})
	RegisterFeature("select_slice", []FeatureCreator{
		{FeatureTypeSelection, func() Feature { return &selectS{} }, []FeatureType{featureTypeAny, featureTypeAny, FeatureTypeSelection}},
	})
}

////////////////////////////////////////////////////////////////////////////////

//apply and map pseudofeatures
func init() {
	RegisterFeature("apply", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return nil }, []FeatureType{FeatureTypeFlow, FeatureTypeSelection}},
	})
	RegisterFeature("map", []FeatureCreator{
		{FeatureTypePacket, func() Feature { return nil }, []FeatureType{FeatureTypePacket, FeatureTypeSelection}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type count struct {
	BaseFeature
	count int
}

func (f *count) Start(context EventContext) {
	f.count = 0
}

func (f *count) Event(new interface{}, context EventContext, src interface{}) {
	f.count++
}

func (f *count) Stop(reason FlowEndReason, context EventContext) {
	f.SetValue(float64(f.count), context, f)
}

func init() {
	RegisterFeature("count", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &count{} }, []FeatureType{FeatureTypePacket}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type mean struct {
	BaseFeature
	total Number
	count int
}

func (f *mean) Start(context EventContext) {
	f.total = nil
	f.count = 0
}

func (f *mean) Event(new interface{}, context EventContext, src interface{}) {
	num := new.(Number)
	if f.total == nil {
		f.total = num
	} else {
		f.total = f.total.Add(num)
	}
	f.count++
}

func (f *mean) Stop(reason FlowEndReason, context EventContext) {
	f.SetValue(f.total.ToFloat()/float64(f.count), context, f)
}

func init() {
	RegisterFeature("mean", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &mean{} }, []FeatureType{FeatureTypePacket}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type min struct {
	BaseFeature
}

func (f *min) Event(new interface{}, context EventContext, src interface{}) {
	if f.value == nil || new.(Number).Less(f.value.(Number)) {
		f.value = new
	}
}

func (f *min) Stop(reason FlowEndReason, context EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterFeature("min", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &min{} }, []FeatureType{FeatureTypePacket}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type max struct {
	BaseFeature
}

func (f *max) Event(new interface{}, context EventContext, src interface{}) {
	if f.value == nil || new.(Number).Greater(f.value.(Number)) {
		f.value = new
	}
}

func (f *max) Stop(reason FlowEndReason, context EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterFeature("max", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &max{} }, []FeatureType{FeatureTypePacket}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type less struct {
	MultiBaseFeature
}

func (f *less) Event(new interface{}, context EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	a, b := UpConvert(values[0].(Number), values[1].(Number))
	if a.Less(b) {
		f.SetValue(true, context, f)
	} else {
		f.SetValue(false, context, f)
	}
}

func init() {
	RegisterFeature("less", []FeatureCreator{
		{FeatureTypeMatch, func() Feature { return &less{} }, []FeatureType{FeatureTypeMatch, FeatureTypeMatch}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type geq struct {
	MultiBaseFeature
}

func (f *geq) Event(new interface{}, context EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	a, b := UpConvert(values[0].(Number), values[1].(Number))
	if ! a.Less(b) {
		f.SetValue(true, context, f)
	} else {
		f.SetValue(false, context, f)
	}
}

func init() {
	RegisterFeature("geq", []FeatureCreator{
		{FeatureTypeMatch, func() Feature { return &geq{} }, []FeatureType{FeatureTypeMatch, FeatureTypeMatch}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type accumulate struct {
	MultiBaseFeature
	vector []interface{}
}

func (f *accumulate) Start(context EventContext) {
	f.vector = make([]interface{}, 0)
}

func (f *accumulate) Stop(reason FlowEndReason, context EventContext) {
	if len(f.vector) != 0 {
		f.SetValue(f.vector, context, f)
	}
}

func (f *accumulate) Event(new interface{}, context EventContext, src interface{}) {
	f.vector = append(f.vector, new)
}

func init() {
	RegisterFeature("accumulate", []FeatureCreator{
		{FeatureTypeMatch, func() Feature { return &accumulate{} }, []FeatureType{FeatureTypePacket}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type concatenate struct {
	BaseFeature
	buffer *bytes.Buffer
}

func (f *concatenate) Start(context EventContext) {
	f.buffer = new(bytes.Buffer)
}

func (f *concatenate) Event(new interface{}, context EventContext, src interface{}) {
	fmt.Fprint(f.buffer, new)
}

func (f *concatenate) Stop(reason FlowEndReason, context EventContext) {
	f.SetValue(f.buffer.String(), context, f)
}

func init() {
	RegisterFeature("concatenate", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &concatenate{} }, []FeatureType{FeatureTypePacket}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type logF struct {
	BaseFeature
}

func (f *logF) Event(new interface{}, context EventContext, src interface{}) {
	num := new.(Number)
    f.value = math.Log(num.ToFloat())
}

func (f *logF) Stop(reason FlowEndReason, context EventContext) {
	f.SetValue(f.value, context, f)
}

func init() {
	RegisterFeature("log", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &logF{} }, []FeatureType{FeatureTypeFlow}},
	})
}
