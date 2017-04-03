package flows

import (
	"fmt"
)

type Feature interface {
	Event(interface{}, Time)
	Value() interface{}
	SetValue(interface{}, Time)
	Start(Time)
	Stop(FlowEndReason, Time)
	Key() FlowKey
	Type() string
	BaseType() string
	setFlow(Flow)
	setBaseType(string)
	getBaseFeature() *BaseFeature
	Reset()
}

type BaseFeature struct {
	value     interface{}
	dependent []Feature
	flow      Flow
	basetype  string
}

func (f *BaseFeature) Event(interface{}, Time)      {}
func (f *BaseFeature) Value() interface{}           { return f.value }
func (f *BaseFeature) Start(Time)                   {}
func (f *BaseFeature) Stop(FlowEndReason, Time)     {}
func (f *BaseFeature) Key() FlowKey                 { return f.flow.Key() }
func (f *BaseFeature) Type() string                 { return f.basetype }
func (f *BaseFeature) BaseType() string             { return f.basetype }
func (f *BaseFeature) setFlow(flow Flow)            { f.flow = flow }
func (f *BaseFeature) setBaseType(basetype string)  { f.basetype = basetype }
func (f *BaseFeature) getBaseFeature() *BaseFeature { return f }
func (f *BaseFeature) Reset()                       { f.value = nil }

func (f *BaseFeature) SetValue(new interface{}, when Time) {
	f.value = new
	if new != nil {
		for _, v := range f.dependent {
			v.Event(new, when)
		}
	}
}

type FeatureCreator func() Feature

type metaFeature struct {
	creator  FeatureCreator
	basetype string
}

func (f metaFeature) NewFeature() Feature {
	ret := f.creator()
	ret.setBaseType(f.basetype)
	return ret
}

type BaseFeatureCreator interface {
	NewFeature() Feature
	BaseType() string
}

func (f metaFeature) BaseType() string { return f.basetype }

var featureRegistry = make(map[string]metaFeature)

func RegisterFeature(name string, f FeatureCreator) BaseFeatureCreator {
	if _, ok := featureRegistry[name]; ok {
		panic(fmt.Sprintf("Feature %s already exists!", name))
	}
	ret := metaFeature{f, name}
	featureRegistry[name] = ret
	return ret
}

type FeatureList struct {
	event    []Feature
	export   []Feature
	startup  []Feature
	exporter Exporter
}

func (list *FeatureList) Init(flow Flow) {
	for _, feature := range list.startup {
		feature.setFlow(flow)
		feature.Reset()
	}
}

func (list *FeatureList) Start(start Time) {
	for _, feature := range list.startup {
		feature.Start(start)
	}
}

func (list *FeatureList) Stop(reason FlowEndReason, time Time) {
	for _, feature := range list.startup {
		feature.Stop(reason, time)
	}
}

func (list *FeatureList) Event(data interface{}, when Time) {
	for _, feature := range list.event {
		feature.Event(data, when)
	}
}

func (list *FeatureList) Export(when Time) {
	list.exporter.Export(list.export, when)
}

func NewFeatureListCreator(features []interface{}, exporter Exporter) FeatureListCreator {
	list := make([]BaseFeatureCreator, len(features))
	basetypes := make([]string, len(features))
	for i, feature := range features {
		switch feature.(type) {
		case string:
			if basetype, ok := featureRegistry[feature.(string)]; !ok {
				panic(fmt.Sprintf("Feature %s not found", feature))
			} else {
				list[i] = basetype
				basetypes[i] = basetype.BaseType()
			}
		case bool, complex128, complex64, float32, float64, int, int16, int32, int64, int8, uint, uint16, uint32, uint64, uint8:
			basetype := NewConstantMetaFeature(feature)
			list[i] = basetype
			basetypes[i] = basetype.BaseType()
		default:
			panic(fmt.Sprint("Don't know what to do with ", feature))
		}
	}

	exporter.Fields(basetypes)

	return func() *FeatureList {
		f := make([]Feature, len(list))
		for i, feature := range list {
			f[i] = feature.NewFeature()
		}
		return &FeatureList{f, f, f, exporter}
	}
}

type ConstantFeature struct {
	value interface{}
	t     string
}

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
func (f *ConstantFeature) Reset()                       {}

func NewConstantMetaFeature(value interface{}) BaseFeatureCreator {
	t := fmt.Sprintf("___const<%v>", value)
	feature := &ConstantFeature{value, t}
	return metaFeature{func() Feature { return feature }, t}
}
