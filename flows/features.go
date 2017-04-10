package flows

import "fmt"

type ConstantFeature struct {
	value interface{}
	t     string
}

func (f *ConstantFeature) setDependent([]Feature)       {}
func (f *ConstantFeature) Event(interface{}, Time)      {}
func (f *ConstantFeature) Value() interface{}           { return f.value }
func (f *ConstantFeature) SetValue(interface{}, Time)   {}
func (f *ConstantFeature) Start(Time)                   {}
func (f *ConstantFeature) Stop(FlowEndReason, Time)     {}
func (f *ConstantFeature) Key() FlowKey                 { return nil }
func (f *ConstantFeature) Type() string                 { return f.t }
func (f *ConstantFeature) BaseType() string             { return f.t }
func (f *ConstantFeature) setFlow(Flow)                 {}
func (f *ConstantFeature) setBaseType(string)           {}
func (f *ConstantFeature) getBaseFeature() *BaseFeature { return nil }

func NewConstantMetaFeature(value interface{}) metaFeature {
	t := fmt.Sprintf("___const<%v>", value)
	feature := &ConstantFeature{value, t}
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
