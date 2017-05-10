package flows

import "fmt"
import "reflect"

type constantFeature struct {
	value interface{}
	t     string
}

func (f *constantFeature) setDependent([]Feature)                  {}
func (f *constantFeature) getDependent() []Feature                 { return nil }
func (f *constantFeature) setArguments([]Feature)                  {}
func (f *constantFeature) Event(interface{}, Time, interface{})    {}
func (f *constantFeature) FinishEvent()                            {}
func (f *constantFeature) Value() interface{}                      { return f.value }
func (f *constantFeature) SetValue(interface{}, Time, interface{}) {}
func (f *constantFeature) Start(Time)                              {}
func (f *constantFeature) Stop(FlowEndReason, Time)                {}
func (f *constantFeature) Key() FlowKey                            { return nil }
func (f *constantFeature) Type() string                            { return f.t }
func (f *constantFeature) BaseType() string                        { return f.t }
func (f *constantFeature) setFlow(Flow)                            {}
func (f *constantFeature) setBaseType(string)                      {}
func (f *constantFeature) getBaseFeature() *BaseFeature            { return nil }
func (f *constantFeature) isConstant() bool                        { return true }

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
	dependent []Feature
	sel       bool
}

func (f *selectF) setDependent(dependent []Feature)        { f.dependent = dependent }
func (f *selectF) getDependent() []Feature                 { return f.dependent }
func (f *selectF) setArguments([]Feature)                  {}
func (f *selectF) FinishEvent()                            {}
func (f *selectF) Value() interface{}                      { return nil }
func (f *selectF) SetValue(interface{}, Time, interface{}) {}
func (f *selectF) Start(Time)                              { f.sel = false }
func (f *selectF) Stop(FlowEndReason, Time)                {}
func (f *selectF) Key() FlowKey                            { return nil }
func (f *selectF) Type() string                            { return "select" }
func (f *selectF) BaseType() string                        { return "select" }
func (f *selectF) setFlow(Flow)                            {}
func (f *selectF) setBaseType(string)                      {}
func (f *selectF) getBaseFeature() *BaseFeature            { return nil }
func (f *selectF) isConstant() bool                        { return false }

func (f *selectF) Event(new interface{}, when Time, src interface{}) {
	/* If src is not nil we got an event from the argument -> Store the boolean value (This always happens before events from the flow)
	   otherwise we have an event from the flow -> forward it in case we should and reset sel
	*/
	if src != nil {
		f.sel = new.(bool)
	} else {
		if f.sel {
			for _, v := range f.dependent {
				v.Event(new, when, nil) // is it ok to use nil as source? (we are faking flow source here)
			}
			f.sel = false
		}
	}
}

type selectS struct {
	dependent            []Feature
	start, stop, current int
}

func (f *selectS) setDependent(dependent []Feature) { f.dependent = dependent }
func (f *selectS) getDependent() []Feature          { return f.dependent }
func (f *selectS) setArguments(arguments []Feature) {
	f.start = int(arguments[0].Value().(Number).ToInt())
	f.stop = int(arguments[1].Value().(Number).ToInt())
}
func (f *selectS) FinishEvent()                            {}
func (f *selectS) Value() interface{}                      { return nil }
func (f *selectS) SetValue(interface{}, Time, interface{}) {}
func (f *selectS) Start(Time)                              { f.current = 0 }
func (f *selectS) Stop(FlowEndReason, Time)                {}
func (f *selectS) Key() FlowKey                            { return nil }
func (f *selectS) Type() string                            { return "select" }
func (f *selectS) BaseType() string                        { return "select" }
func (f *selectS) setFlow(Flow)                            {}
func (f *selectS) setBaseType(string)                      {}
func (f *selectS) getBaseFeature() *BaseFeature            { return nil }
func (f *selectS) isConstant() bool                        { return false }

func (f *selectS) Event(new interface{}, when Time, src interface{}) {
	if f.current >= f.start && f.current < f.stop {
		for _, v := range f.dependent {
			v.Event(new, when, nil) // is it ok to use nil as source? (we are faking flow source here)
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

type mean struct {
	BaseFeature
	total Number
	count int
}

func (f *mean) Start(when Time) {
	f.total = nil
	f.count = 0
}

func (f *mean) Event(new interface{}, when Time, src interface{}) {
	num := new.(Number)
	if f.total == nil {
		f.total = num
	} else {
		f.total = f.total.Add(num)
	}
	f.count++
}

func (f *mean) Stop(reason FlowEndReason, when Time) {
	f.SetValue(f.total.ToFloat()/float64(f.count), when, f)
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

func (f *min) Event(new interface{}, when Time, src interface{}) {
	if f.value == nil || new.(Number).Less(f.value.(Number)) {
		f.value = new
	}
}

func (f *min) Stop(reason FlowEndReason, when Time) {
	f.SetValue(f.value, when, f)
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

func (f *max) Event(new interface{}, when Time, src interface{}) {
	if f.value == nil || new.(Number).Greater(f.value.(Number)) {
		f.value = new
	}
}

func (f *max) Stop(reason FlowEndReason, when Time) {
	f.SetValue(f.value, when, f)
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

func (f *less) Event(new interface{}, when Time, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	a, b := UpConvert(values[0].(Number), values[1].(Number))
	if a.Less(b) {
		f.SetValue(true, when, f)
	} else {
		f.SetValue(false, when, f)
	}
}

func init() {
	RegisterFeature("less", []FeatureCreator{
		{FeatureTypeMatch, func() Feature { return &less{} }, []FeatureType{FeatureTypeMatch, FeatureTypeMatch}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type accumulate struct {
	MultiBaseFeature
    vector []interface{}
}

func (f *accumulate) Start(when Time) {
    f.vector = nil
}

func (f *accumulate) Stop(reason FlowEndReason, when Time) {
    if len(f.vector) != 0 {
        f.SetValue(f.vector, when, f)
    }
}

func (f *accumulate) Event(new interface{}, when Time, src interface{}) {
    f.vector = append(f.vector, new)
}

func init() {
    RegisterFeature("accumulate", []FeatureCreator{
        {FeatureTypeMatch, func() Feature { return &accumulate{} }, []FeatureType{FeatureTypePacket}},
    })
}
