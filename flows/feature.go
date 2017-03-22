package flows

type Feature interface {
	Event(interface{}, int64)
	Value() interface{}
	SetValue(interface{}, int64)
	Start()
	Stop()
	Key() FlowKey
}

type BaseFeature struct {
	value     interface{}
	dependent []Feature
	flow      *BaseFlow
}

func (f *BaseFeature) Event(interface{}, int64) {

}

func (f *BaseFeature) Value() interface{} {
	return f.value
}

func (f *BaseFeature) SetValue(new interface{}, when int64) {
	f.value = new
	if new != nil {
		for _, v := range f.dependent {
			v.Event(new, when)
		}
	}
}

func (f *BaseFeature) Start() {

}

func (f *BaseFeature) Stop() {

}

func (f *BaseFeature) Key() FlowKey {
	return f.flow.Key
}

func NewBaseFeature(flow *BaseFlow) BaseFeature {
	return BaseFeature{flow: flow}
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

func (list *FeatureList) Event(data interface{}, when int64) {
	for _, feature := range list.event {
		feature.Event(data, when)
	}
}

func (list *FeatureList) Export(why string, when int64) {
	list.exporter.Export(list.export, why, when)
}

func NewFeatureList(event, export, startup []Feature, exporter Exporter) FeatureList {
	return FeatureList{event, export, startup, exporter}
}
