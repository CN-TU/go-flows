package flows

import "fmt"
import "reflect"

type constantFeature struct {
	value interface{}
	t     string
}

func (f *constantFeature) setDependent([]Feature)       {}
func (f *constantFeature) getDependent() []Feature      { return nil }
func (f *constantFeature) Event(interface{}, Time)      {}
func (f *constantFeature) Value() interface{}           { return f.value }
func (f *constantFeature) SetValue(interface{}, Time)   {}
func (f *constantFeature) Start(Time)                   {}
func (f *constantFeature) Stop(FlowEndReason, Time)     {}
func (f *constantFeature) Key() FlowKey                 { return nil }
func (f *constantFeature) Type() string                 { return f.t }
func (f *constantFeature) BaseType() string             { return f.t }
func (f *constantFeature) setFlow(Flow)                 {}
func (f *constantFeature) setBaseType(string)           {}
func (f *constantFeature) getBaseFeature() *BaseFeature { return nil }

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

type mean struct {
	BaseFeature
	total Number
	count int
}

func (f *mean) Start(when Time) {
	f.total = nil
	f.count = 0
}

func (f *mean) Event(new interface{}, when Time) {
	num := new.(Number)
	if f.total == nil {
		f.total = num
	} else {
		f.total = f.total.Add(num)
	}
	f.count++
}

func (f *mean) Stop(reason FlowEndReason, when Time) {
	f.SetValue(f.total.ToFloat()/float64(f.count), when)
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

func (f *min) Event(new interface{}, when Time) {
	if f.value == nil || new.(Number).Less(f.value.(Number)) {
		f.value = new
	}
}

func (f *min) Stop(reason FlowEndReason, when Time) {
	f.SetValue(f.value, when)
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

func (f *max) Event(new interface{}, when Time) {
	if f.value == nil || new.(Number).Greater(f.value.(Number)) {
		f.value = new
	}
}

func (f *max) Stop(reason FlowEndReason, when Time) {
	f.SetValue(f.value, when)
}

func init() {
	RegisterFeature("max", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &max{} }, []FeatureType{FeatureTypePacket}},
	})
}
