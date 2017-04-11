package flows

import (
	"fmt"
	"strings"
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
	setDependent([]Feature)
}

type BaseFeature struct {
	value     interface{}
	dependent []Feature
	flow      Flow
	basetype  string
}

func (f *BaseFeature) setDependent(dep []Feature)   { f.dependent = dep }
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

func (f *BaseFeature) SetValue(new interface{}, when Time) {
	f.value = new
	if new != nil {
		for _, v := range f.dependent {
			v.Event(new, when)
		}
	}
}

type FeatureCreator struct {
	Ret       FeatureType
	Create    func() Feature
	Arguments []FeatureType
}

type metaFeature struct {
	creator  FeatureCreator
	basetype string
}

func (m metaFeature) String() string {
	return fmt.Sprintf("<%s>%s(%s)", m.creator.Ret, m.basetype, m.creator.Arguments)
}

func (f metaFeature) NewFeature() Feature {
	ret := f.creator.Create()
	ret.setBaseType(f.basetype)
	return ret
}

type FeatureType int

func (f FeatureType) String() string {
	switch f {
	case FeatureTypePacket:
		return "PacketFeature"
	case FeatureTypeFlow:
		return "FlowFeature"
	case featureTypeAny:
		return "AnyFeature"
	case FeatureTypeEllipsis:
		return "..."
	}
	return "???"
}

const (
	FeatureTypePacket FeatureType = iota
	FeatureTypeFlow
	featureTypeAny      //for constants
	FeatureTypeEllipsis //for variadic
	featureTypeMax
)

type BaseFeatureCreator interface {
	NewFeature() Feature
	BaseType() string
}

func (f metaFeature) BaseType() string { return f.basetype }

var featureRegistry = make([]map[string][]metaFeature, featureTypeMax)

func init() {
	for i := range featureRegistry {
		featureRegistry[i] = make(map[string][]metaFeature)
	}
}

func RegisterFeature(name string, types []FeatureCreator) string {
	for _, t := range types {
		/* if _, ok := featureRegistry[t.Ret][name]; ok {
			panic(fmt.Sprintf("Feature (%v) %s already defined!", t.Ret, name))
		}*/ //FIXME add some kind of konsistency check!
		featureRegistry[t.Ret][name] = append(featureRegistry[t.Ret][name], metaFeature{t, name})
	}
	return name
}

type FeatureList struct {
	event    []Feature
	export   []Feature
	startup  []Feature
	exporter Exporter
}

//Rework stop (for slice?)
//stop event features and propagate?
//same for start?

func (list *FeatureList) Init(flow Flow) {
	for _, feature := range list.startup {
		feature.setFlow(flow)
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

func getFeature(feature string, ret FeatureType, nargs int) (metaFeature, bool) {
	variadicFound := false
	var variadic metaFeature
	for _, f := range featureRegistry[ret][feature] {
		if len(f.creator.Arguments) >= 2 && f.creator.Arguments[len(f.creator.Arguments)-1] == FeatureTypeEllipsis {
			variadicFound = true
			variadic = f
		} else if len(f.creator.Arguments) == nargs {
			return f, true
		}
	}
	if variadicFound {
		return variadic, true
	}
	return metaFeature{}, false
}

func getArgumentTypes(feature string, ret FeatureType, nargs int) []FeatureType {
	f, found := getFeature(feature, ret, nargs)
	if !found {
		return nil
	}
	if f.creator.Arguments[len(f.creator.Arguments)-1] == FeatureTypeEllipsis {
		r := make([]FeatureType, nargs)
		variadic := f.creator.Arguments[len(f.creator.Arguments)-2]
		for i := range r {
			r[i] = variadic
		}
		return r
	}
	return f.creator.Arguments
}

func feature2id(feature interface{}, ret FeatureType) string {
	switch feature.(type) {
	case string:
		return fmt.Sprintf("<%d>%s", ret, feature)
	case bool, complex128, complex64, float32, float64, int, int16, int32, int64, int8, uint, uint16, uint32, uint64, uint8:
		return fmt.Sprintf("Const{%v}", feature)
	case []interface{}:
		feature := feature.([]interface{})
		features := make([]string, len(feature))
		arguments := append([]FeatureType{ret}, getArgumentTypes(feature[0].(string), ret, len(feature)-1)...)
		for i, f := range feature {
			features[i] = feature2id(f, arguments[i])
		}
		return "[" + strings.Join(features, ",") + "]"
	default:
		panic(fmt.Sprint("Don't know what to do with ", feature))
	}
}

func NewFeatureListCreator(features []interface{}, exporter Exporter, base FeatureType) FeatureListCreator {
	type featureWithType struct {
		feature interface{}
		ret     FeatureType
		export  bool
	}

	type featureToInit struct {
		feature   metaFeature
		arguments []int
		call      []int
		event     bool
		export    bool
	}

	init := make([]featureToInit, 0, len(features))

	seen := make(map[string]int, len(features))
	stack := make([]featureWithType, len(features))
	for i := range features {
		stack[i] = featureWithType{features[i], base, true}
	}

	var feature featureWithType
MAIN:
	for len(stack) > 0 {
		feature, stack = stack[0], stack[1:]
		id := feature2id(feature.feature, feature.ret)
		if _, ok := seen[id]; ok {
			continue MAIN
		}
		switch feature.feature.(type) {
		case string:
			if basetype, ok := getFeature(feature.feature.(string), feature.ret, 0); !ok {
				panic(fmt.Sprintf("Feature %s returning %s with no arguments not found", feature.feature, feature.ret))
			} else {
				seen[id] = len(init)
				init = append(init, featureToInit{basetype, nil, nil, true, feature.export})
			}
		case bool, float32, float64, int, int16, int32, int64, int8, uint, uint16, uint32, uint64, uint8:
			//Change const to: Float64 Unsigned64 Signed64?
			basetype := NewConstantMetaFeature(feature.feature)
			seen[id] = len(init)
			init = append(init, featureToInit{basetype, nil, nil, false, feature.export})
		case []interface{}:
			arguments := feature.feature.([]interface{})
			if basetype, ok := getFeature(arguments[0].(string), feature.ret, len(arguments)-1); !ok {
				panic(fmt.Sprintf("Feature %s returning %s with arguments %v not found", arguments[0].(string), feature.ret, arguments[1:]))
			} else {
				argumentTypes := getArgumentTypes(arguments[0].(string), feature.ret, len(arguments)-1)
				argumentPos := make([]int, 0, len(arguments)-1)
				for i, f := range arguments[1:] {
					if pos, ok := seen[feature2id(f, argumentTypes[i])]; !ok {
						newstack := make([]featureWithType, len(arguments)-1)
						for i, arg := range arguments[1:] {
							newstack[i] = featureWithType{arg, argumentTypes[i], false}
						}
						stack = append(append(newstack, feature), stack...)
						continue MAIN
					} else {
						argumentPos = append(argumentPos, pos)
					}
				}
				seen[id] = len(init)
				init = append(init, featureToInit{basetype, argumentPos, nil, false, feature.export}) //fake BaseType?
			}
		default:
			panic(fmt.Sprint("Don't know what to do with ", feature))
		}
	}

	for i, f := range init {
		for _, arg := range f.arguments {
			init[arg].call = append(init[arg].call, i)
		}
	}

	basetypes := make([]string, 0, len(features))
	nevent := 0
	nexport := 0
	for _, feature := range init {
		if feature.export {
			basetypes = append(basetypes, feature.feature.BaseType())
			nexport++
		}
		if feature.event {
			nevent++
		}
	}

	exporter.Fields(basetypes)

	return func() *FeatureList {
		f := make([]Feature, len(init))
		event := make([]Feature, 0, nevent)
		export := make([]Feature, 0, nexport)
		for i, feature := range init {
			f[i] = feature.feature.NewFeature()
			if feature.event {
				event = append(event, f[i])
			}
			if feature.export {
				export = append(export, f[i])
			}
		}
		for i, feature := range init {
			if len(feature.call) > 0 {
				args := make([]Feature, len(feature.call))
				for i, call := range feature.call {
					args[i] = f[call]
				}
				f[i].setDependent(args)
			}
		}
		return &FeatureList{
			startup:  f,
			event:    event,
			export:   export,
			exporter: exporter,
		}
	}
}
