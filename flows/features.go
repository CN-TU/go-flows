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

func toFloat(v interface{}) float64 {
	switch v.(type) {
	case float32:
		return float64(v.(float32))
	case float64:
		return float64(v.(float64))
	case int:
		return float64(v.(int))
	case int16:
		return float64(v.(int16))
	case int32:
		return float64(v.(int32))
	case int64:
		return float64(v.(int64))
	case int8:
		return float64(v.(int8))
	case uint:
		return float64(v.(uint))
	case uint16:
		return float64(v.(uint16))
	case uint32:
		return float64(v.(uint32))
	case uint64:
		return float64(v.(uint64))
	case uint8:
		return float64(v.(uint8))
	}
	panic("Cannot convert v to float!")
}

type mean struct {
	BaseFeature
	total float64
	count int
}

func (f *mean) Start(when Time) {
	f.total = 0
	f.count = 0
}

func (f *mean) Event(new interface{}, when Time) {
	f.total += toFloat(new) //smell this feels wrong
	f.count++
}

func (f *mean) Stop(reason FlowEndReason, when Time) {
	f.SetValue(f.total/float64(f.count), when)
}

func init() {
	RegisterFeature("mean", []FeatureCreator{
		{FeatureTypeFlow, func() Feature { return &mean{} }, []FeatureType{FeatureTypePacket}},
	})
}
