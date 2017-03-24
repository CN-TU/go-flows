package flows

import (
	"fmt"
)

type Feature interface {
	Event(interface{}, Time)
	Value() interface{}
	SetValue(interface{}, Time)
	Start()
	Stop()
	Key() FlowKey
	Type() string
	BaseType() string
	setFlow(Flow)
	setBaseType(string)
}

type BaseFeature struct {
	value     interface{}
	dependent []Feature
	flow      Flow
	basetype  string
}

func (f *BaseFeature) Event(interface{}, Time)     {}
func (f *BaseFeature) Value() interface{}          { return f.value }
func (f *BaseFeature) Start()                      {}
func (f *BaseFeature) Stop()                       {}
func (f *BaseFeature) Key() FlowKey                { return f.flow.Key() }
func (f *BaseFeature) Type() string                { return f.basetype }
func (f *BaseFeature) BaseType() string            { return f.basetype }
func (f *BaseFeature) setFlow(flow Flow)           { f.flow = flow }
func (f *BaseFeature) setBaseType(basetype string) { f.basetype = basetype }

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

func (f metaFeature) NewFeature(flow Flow) Feature {
	ret := f.creator()
	ret.setFlow(flow)
	ret.setBaseType(f.basetype)
	return ret
}

type BaseFeatureCreator interface {
	NewFeature(Flow) Feature
}

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

func (list *FeatureList) Start() {
	for _, feature := range list.startup {
		feature.Start()
	}
}

func (list *FeatureList) Stop() {
	for _, feature := range list.startup {
		feature.Stop()
	}
}

func (list *FeatureList) Event(data interface{}, when Time) {
	for _, feature := range list.event {
		feature.Event(data, when)
	}
}

func (list *FeatureList) Export(why FlowEndReason, when Time) {
	list.exporter.Export(list.export, why, when)
}

func NewFeatureListCreator(features []string, exporter Exporter) FeatureListCreator {
	list := make([]BaseFeatureCreator, len(features))
	var ok bool
	for i, feature := range features {
		if list[i], ok = featureRegistry[feature]; !ok {
			panic(fmt.Sprint("Feature %s not found", feature))
		}
	}

	return func(flow Flow) FeatureList {
		f := make([]Feature, len(list))
		for i, feature := range list {
			f[i] = feature.NewFeature(flow)
		}
		return FeatureList{f, f, f, exporter}
	}
}
