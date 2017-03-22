package flows

type Feature interface {
	Event(interface{}, Time)
	Value() interface{}
	SetValue(interface{}, Time)
	Start()
	Stop()
	Key() FlowKey
}

type BaseFeature struct {
	value     interface{}
	dependent []Feature
	flow      *BaseFlow
}

func (f *BaseFeature) Event(interface{}, Time) {

}

func (f *BaseFeature) Value() interface{} {
	return f.value
}

func (f *BaseFeature) SetValue(new interface{}, when Time) {
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
	return f.flow.key
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

func (list *FeatureList) Event(data interface{}, when Time) {
	for _, feature := range list.event {
		feature.Event(data, when)
	}
}

func (list *FeatureList) Export(why string, when Time) {
	list.exporter.Export(list.export, why, when)
}

func NewFeatureList(event, export, startup []Feature, exporter Exporter) FeatureList {
	return FeatureList{event, export, startup, exporter}
}
