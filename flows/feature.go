package flows

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"
)

// Feature interfaces, which all features need to implement
type Feature interface {
	// Event gets called for every event. Data is provided via the first argument and current time via the second.
	Event(interface{}, Time, interface{})
	// FinishEvent gets called after every Event happened
	FinishEvent()
	// Value provides the current stored value.
	Value() interface{}
	// SetValue stores a new value with the associated time.
	SetValue(interface{}, Time, interface{})
	// Start gets called when the flow starts.
	Start(Time)
	// Stop gets called with an end reason and time when a flow stops
	Stop(FlowEndReason, Time)
	// Key returns the current flow key.
	Key() FlowKey
	// Type returns the type associated with the current value, which can be different from BaseType.
	Type() string
	// BaseType returns the type of the feature.
	BaseType() string
	// Emit sends value new, with time when, and source self to the dependent Features
	Emit(new interface{}, when Time, self interface{})
	setFlow(Flow)
	setBaseType(string)
	getBaseFeature() *BaseFeature
	setDependent([]Feature)
	getDependent() []Feature
	SetArguments([]Feature)
	IsConstant() bool
}

type EmptyBaseFeature struct {
	dependent []Feature
	flow      Flow
}

func (f *EmptyBaseFeature) setDependent(dep []Feature) { f.dependent = dep }
func (f *EmptyBaseFeature) getDependent() []Feature    { return f.dependent }
func (f *EmptyBaseFeature) SetArguments([]Feature)     {}

// Key returns the current flow key.
func (f *EmptyBaseFeature) Key() FlowKey { return f.flow.Key() }

// Event gets called for every event. Data is provided via the first argument and current time via the second.
func (f *EmptyBaseFeature) Event(interface{}, Time, interface{}) {}

// FinishEvent gets called after every Event happened
func (f *EmptyBaseFeature) FinishEvent() {
	for _, v := range f.dependent {
		v.FinishEvent()
	}
}

// Value provides the current stored value.
func (f *EmptyBaseFeature) Value() interface{} { return nil }

// Start gets called when the flow starts.
func (f *EmptyBaseFeature) Start(Time) {}

// Stop gets called with an end reason and time when a flow stops
func (f *EmptyBaseFeature) Stop(FlowEndReason, Time) {}

// Type returns the type associated with the current value, which can be different from BaseType.
func (f *EmptyBaseFeature) Type() string { log.Fatal("Not implemented"); return "" }

// BaseType returns the type of the feature.
func (f *EmptyBaseFeature) BaseType() string             { log.Fatal("Not implemented"); return "" }
func (f *EmptyBaseFeature) setFlow(flow Flow)            { f.flow = flow }
func (f *EmptyBaseFeature) setBaseType(basetype string)  {}
func (f *EmptyBaseFeature) getBaseFeature() *BaseFeature { log.Fatal("Not implemented"); return nil }

// IsConstant returns true if the feature is constant
func (f *EmptyBaseFeature) IsConstant() bool { return false }

// SetValue stores a new value with the associated time.
func (f *EmptyBaseFeature) SetValue(new interface{}, when Time, self interface{}) {
}

// Emit sends value new, with time when, and source self to the dependent Features
func (f *EmptyBaseFeature) Emit(new interface{}, when Time, self interface{}) {
	for _, v := range f.dependent {
		v.Event(new, when, self)
	}
}

// BaseFeature includes all the basic functionality to fulfill the Feature interface.
// Embedd this struct for creating new features.
type BaseFeature struct {
	EmptyBaseFeature
	value    interface{}
	basetype string
}

// Value provides the current stored value.
func (f *BaseFeature) Value() interface{} { return f.value }

// Type returns the type associated with the current value, which can be different from BaseType.
func (f *BaseFeature) Type() string { return f.basetype }

// BaseType returns the type of the feature.
func (f *BaseFeature) BaseType() string             { return f.basetype }
func (f *BaseFeature) setBaseType(basetype string)  { f.basetype = basetype }
func (f *BaseFeature) getBaseFeature() *BaseFeature { return f }

// SetValue stores a new value with the associated time.
func (f *BaseFeature) SetValue(new interface{}, when Time, self interface{}) {
	f.value = new
	if new != nil {
		f.Emit(new, when, self)
	}
}

type multiEvent interface {
	CheckAll(interface{}, interface{}) []interface{}
	Reset()
}

// singleMultiEvent (one not const) (every event is the right one!)
type singleMultiEvent struct {
	c []interface{}
}

func (m *singleMultiEvent) CheckAll(new interface{}, which interface{}) []interface{} {
	ret := make([]interface{}, len(m.c))
	for i, c := range m.c {
		if c == nil {
			ret[i] = new
		} else {
			ret[i] = c
		}
	}
	return ret
}

func (m *singleMultiEvent) Reset() {}

// dualMultiEvent (two not const)
type dualMultiEvent struct {
	c     []interface{}
	nc    [2]Feature
	state [2]bool
}

func (m *dualMultiEvent) CheckAll(new interface{}, which interface{}) []interface{} {
	if which == m.nc[0] {
		m.state[0] = true
	} else if which == m.nc[1] {
		m.state[1] = true
	}
	if m.state[0] && m.state[1] {
		ret := make([]interface{}, len(m.c))
		j := 0
		for i, c := range m.c {
			if c == nil {
				ret[i] = m.nc[j].Value()
				j++
			} else {
				ret[i] = c
			}
		}
		return ret
	}
	return nil
}

func (m *dualMultiEvent) Reset() {
	m.state[0] = false
	m.state[1] = false
}

// genericMultiEvent (generic map implementation)
type genericMultiEvent struct {
	c     []interface{}
	nc    map[Feature]int
	state []bool
}

func (m *genericMultiEvent) CheckAll(new interface{}, which interface{}) []interface{} {
	m.state[m.nc[which.(Feature)]] = true
	for _, state := range m.state {
		if !state {
			return nil
		}
	}

	ret := make([]interface{}, len(m.c))
	for i, c := range m.c {
		if f, ok := c.(Feature); ok {
			ret[i] = f.Value()
		} else {
			ret[i] = c
		}
	}
	return ret

}

func (m *genericMultiEvent) Reset() {
	for i := range m.state {
		m.state[i] = false
	}
}

// MultiBaseFeature extends BaseFeature with event tracking.
// Embedd this struct for creating new features with multiple arguments.
type MultiBaseFeature struct {
	BaseFeature
	eventReady multiEvent
}

// EventResult returns the list of values for a multievent or nil if not every argument had an event
func (f *MultiBaseFeature) EventResult(new interface{}, which interface{}) []interface{} {
	return f.eventReady.CheckAll(new, which)
}

// FinishEvent gets called after every Event happened
func (f *MultiBaseFeature) FinishEvent() {
	f.eventReady.Reset()
	f.BaseFeature.FinishEvent()
}

func (f *MultiBaseFeature) SetArguments(args []Feature) {
	featurelist := make([]interface{}, len(args))
	features := make([]int, 0, len(args))
	for i, feature := range args {
		if feature.IsConstant() {
			featurelist[i] = feature.Value()
		} else {
			featurelist[i] = feature
			features = append(features, i)
		}
	}
	switch len(features) {
	case 1:
		featurelist[features[0]] = nil
		f.eventReady = &singleMultiEvent{featurelist}
	case 2:
		event := &dualMultiEvent{}
		event.nc[0] = featurelist[features[0]].(Feature)
		event.nc[1] = featurelist[features[1]].(Feature)
		featurelist[features[0]] = nil
		featurelist[features[1]] = nil
		event.c = featurelist
		f.eventReady = event
	default:
		nc := make(map[Feature]int, len(features))
		for _, feature := range features {
			nc[featurelist[feature].(Feature)] = feature
		}
		f.eventReady = &genericMultiEvent{c: featurelist, nc: nc, state: make([]bool, len(features))}
	}
}

type featureToInit struct {
	feature   metaFeature
	ret       interface{}
	arguments []int
	call      []int
	event     bool
	export    bool
	composite string
	function  string
}

// FeatureListCreator represents a way to instantiate a tree of features.
type FeatureListCreator struct {
	init      []featureToInit
	basetypes []string
	exporter  []Exporter
	creator   func() *featureList
}

// Fields writes the field definitions to the exporter
func (fl FeatureListCreator) Fields() {
	for _, exporter := range fl.exporter {
		exporter.Fields(fl.basetypes)
	}
}

// FeatureCreator represents a single uninstantiated feature.
type FeatureCreator struct {
	// Ret specifies the return type of the feature.
	Ret FeatureType
	// Create is a function for creating a new feature of this type.
	Create func() Feature
	// Arguments specifies the feature types expected for computing this feature.
	Arguments []FeatureType
}

type metaFeature struct {
	creator  FeatureCreator
	basetype string
}

func (m metaFeature) String() string {
	return fmt.Sprintf("<%s>%s(%s)", m.creator.Ret, m.basetype, m.creator.Arguments)
}

func (m metaFeature) NewFeature() Feature {
	ret := m.creator.Create()
	ret.setBaseType(m.basetype)
	return ret
}

func (m metaFeature) BaseType() string { return m.basetype }

// FeatureType represents if the feature is a flow or packet feature.
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
	case FeatureTypeSelection:
		return "Selection"
	case FeatureTypeMatch:
		return "Match"
	}
	return "???"
}

const (
	// FeatureTypePacket represents a packet feature.
	FeatureTypePacket FeatureType = iota
	// FeatureTypeFlow represents a flow feature.
	FeatureTypeFlow
	featureTypeAny //for constants
	// FeatureTypeEllipsis can be used to mark a function with variadic arguments. It represents a continuation of the previous argument type.
	FeatureTypeEllipsis
	// FeatureTypeMatch specifies that the argument type has to match the return type
	FeatureTypeMatch
	// FeatureTypeSelection specifies a selection
	FeatureTypeSelection
	// RawPacket specifies a packet from the packet source
	RawPacket
	// RawFlow specifies a flow from the flow source
	RawFlow
	featureTypeMax
)

var featureRegistry = make([]map[string][]metaFeature, featureTypeMax)
var compositeFeatures = make(map[string][]interface{})

func init() {
	for i := range featureRegistry {
		featureRegistry[i] = make(map[string][]metaFeature)
	}
}

// RegisterFeature registers a new feature with the given name. types can be used to create features returning different FeatureType with the same name.
func RegisterFeature(name string, types []FeatureCreator) {
	for _, t := range types {
		/* if _, ok := featureRegistry[t.Ret][name]; ok {
			panic(fmt.Sprintf("Feature (%v) %s already defined!", t.Ret, name))
		}*/ //FIXME add some kind of konsistency check!
		featureRegistry[t.Ret][name] = append(featureRegistry[t.Ret][name], metaFeature{t, name})
	}
}

// RegisterCompositeFeature registers a new composite feature with the given name. Composite features are features that depend on other features and need to be
// represented in the form ["featurea", ["featureb", "featurec"]]
func RegisterCompositeFeature(name string, definition []interface{}) {
	if _, ok := compositeFeatures[name]; ok {
		panic(fmt.Sprintf("Feature %s already registered", name))
	}
	compositeFeatures[name] = definition
}

func compositeToCall(features []interface{}) []string {
	var ret []string
	flen := len(features) - 1
	for i, feature := range features {
		if list, ok := feature.([]interface{}); ok {
			ret = append(ret, compositeToCall(list)...)
		} else {
			ret = append(ret, fmt.Sprint(feature))
		}
		if i == 0 {
			ret = append(ret, "(")
		} else if i < flen {
			ret = append(ret, ",")
		} else {
			ret = append(ret, ")")
		}
	}
	return ret
}

// ListFeatures creates a table of available features and outputs it to w.
func ListFeatures(w io.Writer) {
	t := tabwriter.NewWriter(w, 0, 1, 1, ' ', 0)
	pf := make(map[string]string)
	ff := make(map[string]string)
	args := make(map[string]string)
	impl := make(map[string]string)
	var base, functions, filters []string
	for ret, features := range featureRegistry {
		for name, featurelist := range features {
			for _, feature := range featurelist {
				if feature.creator.Ret == RawPacket || feature.creator.Ret == RawFlow {
					filters = append(filters, name)
					tmp := make([]string, len(feature.creator.Arguments))
					for i := range feature.creator.Arguments {
						switch feature.creator.Arguments[i] {
						case RawFlow, FeatureTypeFlow:
							tmp[i] = "F"
						case RawPacket, FeatureTypePacket:
							tmp[i] = "P"
						case FeatureTypeEllipsis:
							tmp[i] = "..."
						case FeatureTypeMatch:
							tmp[i] = "X"
						case FeatureTypeSelection:
							tmp[i] = "S"
						case featureTypeAny:
							tmp[i] = "C"
						}
					}
					args[name] = strings.Join(tmp, ",")
				} else if len(feature.creator.Arguments) == 1 &&
					(feature.creator.Arguments[0] == RawPacket || feature.creator.Arguments[0] == RawFlow) {
					base = append(base, name)
				} else {
					tmp := make([]string, len(feature.creator.Arguments))
					for i := range feature.creator.Arguments {
						switch feature.creator.Arguments[i] {
						case FeatureTypeFlow:
							tmp[i] = "F"
						case FeatureTypePacket:
							tmp[i] = "P"
						case FeatureTypeEllipsis:
							tmp[i] = "..."
						case FeatureTypeMatch:
							tmp[i] = "X"
						case FeatureTypeSelection:
							tmp[i] = "S"
						case featureTypeAny:
							tmp[i] = "C"
						}
					}
					args[name] = strings.Join(tmp, ",")
					functions = append(functions, name)
				}
				switch FeatureType(ret) {
				case RawPacket, FeatureTypePacket:
					pf[name] = "X"
				case RawFlow, FeatureTypeFlow:
					ff[name] = "X"
				case FeatureTypeMatch:
					pf[name] = "X"
					ff[name] = "X"
				}

			}
		}
	}
	for name, implementation := range compositeFeatures {
		impl[name] = fmt.Sprint(" = ", strings.Join(compositeToCall(implementation), ""))
		fun := implementation[0].(string)
		if _, ok := featureRegistry[FeatureTypeFlow][fun]; ok {
			ff[name] = "X"
		}
		if _, ok := featureRegistry[FeatureTypePacket][fun]; ok {
			pf[name] = "X"
		}
		if _, ok := featureRegistry[FeatureTypeMatch][fun]; ok {
			ff[name] = "X"
			pf[name] = "X"
		}
		base = append(base, name)
	}
	sort.Strings(base)
	sort.Strings(functions)
	sort.Strings(filters)
	fmt.Fprintln(w, "P ... Packet Feature")
	fmt.Fprintln(w, "F ... Flow Feature")
	fmt.Fprintln(w, "S ... Selection")
	fmt.Fprintln(w, "C ... Constant")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Base Features:")
	fmt.Fprintln(w, "  P F Name")
	var last string
	for _, name := range base {
		if name == last {
			continue
		}
		last = name
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "  %1s\t%1s\t%s%s\n", pf[name], ff[name], name, impl[name])
		t.Write(line.Bytes())
	}
	t.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Functions:")
	fmt.Fprintln(w, "  P F Name")
	for _, name := range functions {
		if name == last {
			continue
		}
		last = name
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "  %1s\t%1s\t%s(%s)\n", pf[name], ff[name], name, args[name])
		t.Write(line.Bytes())
	}
	t.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Filters:")
	fmt.Fprintln(w, "  P F Name")
	for _, name := range filters {
		if name == last {
			continue
		}
		last = name
		line := new(bytes.Buffer)
		fmt.Fprintf(line, "  %1s\t%1s\t%s(%s)\n", pf[name], ff[name], name, args[name])
		t.Write(line.Bytes())
	}
	t.Flush()

}

// CleanupFeatures deletes _all_ feature definitions for conserving memory. Call this after you've finished creating all feature lists with NewFeatureListCreator.
func CleanupFeatures() {
	featureRegistry = nil
	compositeFeatures = nil
}

type featureList struct {
	event    []Feature
	export   []Feature
	startup  []Feature
	exporter []Exporter
}

func (list *featureList) Init(flow Flow) {
	for _, feature := range list.startup {
		feature.setFlow(flow)
	}
}

func (list *featureList) Start(start Time) {
	for _, feature := range list.startup {
		feature.Start(start)
	}
}

func (list *featureList) Stop(reason FlowEndReason, time Time) {
	for _, feature := range list.startup {
		feature.Stop(reason, time)
	}
}

func (list *featureList) Event(data interface{}, when Time) {
	for _, feature := range list.event {
		feature.Event(data, when, nil)
	}
	for _, feature := range list.event {
		feature.FinishEvent()
	}
}

func (list *featureList) Export(when Time) {
	for _, exporter := range list.exporter {
		exporter.Export(list.export, when)
	}
}

func getFeature(feature string, ret FeatureType, nargs int) (metaFeature, bool) {
	variadicFound := false
	var variadic metaFeature
	for _, t := range []FeatureType{ret, FeatureTypeMatch} {
		for _, f := range featureRegistry[t][feature] {
			if len(f.creator.Arguments) >= 2 && f.creator.Arguments[len(f.creator.Arguments)-1] == FeatureTypeEllipsis {
				variadicFound = true
				variadic = f
			} else if len(f.creator.Arguments) == nargs {
				return f, true
			}
		}
	}
	if variadicFound {
		return variadic, true
	}
	return metaFeature{}, false
}

func getArgumentTypes(f metaFeature, ret FeatureType, nargs int) []FeatureType {
	if f.creator.Arguments[len(f.creator.Arguments)-1] == FeatureTypeEllipsis {
		r := make([]FeatureType, nargs)
		last := len(f.creator.Arguments) - 2
		variadic := f.creator.Arguments[last]
		if variadic == FeatureTypeMatch {
			variadic = ret
		}
		for i := 0; i < nargs; i++ {
			if i > last {
				r[i] = variadic
			} else {
				if f.creator.Arguments[i] == FeatureTypeMatch {
					r[i] = ret
				} else {
					r[i] = f.creator.Arguments[i]
				}
			}
		}
		return r
	}
	if f.creator.Ret == FeatureTypeMatch {
		r := make([]FeatureType, nargs)
		for i := range r {
			if f.creator.Arguments[i] == FeatureTypeMatch {
				r[i] = ret
			} else {
				r[i] = f.creator.Arguments[i]
			}
		}
		return r
	}
	return f.creator.Arguments
}

func feature2id(feature interface{}, ret FeatureType) string {
	switch feature.(type) {
	case string:
		return fmt.Sprintf("<%d>%s", ret, feature)
	case bool, float64, int64:
		return fmt.Sprintf("Const{%v}", feature)
	case []interface{}:
		feature := feature.([]interface{})
		features := make([]string, len(feature))
		f, found := getFeature(feature[0].(string), ret, len(feature)-1)
		if !found {
			panic(fmt.Sprintf("Feature %s with return type %s and %d arguments not found!", feature[0].(string), ret, len(feature)-1))
		}
		arguments := append([]FeatureType{ret}, getArgumentTypes(f, ret, len(feature)-1)...)
		for i, f := range feature {
			features[i] = feature2id(f, arguments[i])
		}
		return "[" + strings.Join(features, ",") + "]"
	default:
		panic(fmt.Sprint("Don't know what to do with ", feature))
	}
}

// NewFeatureListCreator creates a new featurelist description for the specified exporter with the given features using base as feature type for exported features.
func NewFeatureListCreator(features []interface{}, exporter []Exporter, base FeatureType) FeatureListCreator {
	type featureWithType struct {
		feature   interface{}
		ret       FeatureType
		export    bool
		composite string
		reset     bool
		selection string
		function  string
	}

	init := make([]featureToInit, 0, len(features))

	stack := make([]featureWithType, len(features))
	for i := range features {
		stack[i] = featureWithType{features[i], base, true, "", false, "", ""}
	}

	type selection struct {
		argument []int
		seen     map[string]int
	}

	selections := make(map[string]*selection)

	mainSelection := &selection{nil, make(map[string]int, len(features))}
	selections[feature2id([]interface{}{"select", true}, FeatureTypeSelection)] = mainSelection
	currentSelection := mainSelection

	var feature featureWithType
MAIN:
	for len(stack) > 0 {
		feature, stack = stack[0], stack[1:]
		id := feature2id(feature.feature, feature.ret)
		seen := currentSelection.seen
		if _, ok := seen[id]; ok {
			continue MAIN
		}
		switch feature.feature.(type) {
		case string:
			if basetype, ok := getFeature(feature.feature.(string), feature.ret, 1); !ok {
				if composite, ok := compositeFeatures[feature.feature.(string)]; !ok {
					panic(fmt.Sprintf("Feature %s returning %s with input raw packet/flow not found", feature.feature, feature.ret))
				} else {
					stack = append([]featureWithType{{composite, feature.ret, feature.export, feature.feature.(string), false, "", ""}}, stack...)
				}
			} else {
				if basetype.creator.Arguments[0] != RawPacket { //TODO: implement flow input
					panic(fmt.Sprintf("Feature %s returning %s with input raw packet not found", feature.feature, feature.ret))
				}
				seen[id] = len(init)
				init = append(init, featureToInit{basetype, feature.ret, currentSelection.argument, nil, currentSelection.argument == nil, feature.export, feature.composite, feature.function})
			}
		case bool, float64, int64:
			basetype := newConstantMetaFeature(feature.feature)
			seen[id] = len(init)
			init = append(init, featureToInit{basetype, feature.feature, nil, nil, false, feature.export, feature.composite, feature.function})
		case []interface{}:
			arguments := feature.feature.([]interface{})
			fun := arguments[0].(string)
			if basetype, ok := getFeature(fun, feature.ret, len(arguments)-1); !ok {
				panic(fmt.Sprintf("Feature %s returning %s with arguments %v not found", fun, feature.ret, arguments[1:]))
			} else {
				if fun == "apply" || fun == "map" {
					sel := feature2id(arguments[2], FeatureTypeSelection)
					if fun == "apply" && feature.ret != FeatureTypeFlow {
						panic("Unexpected apply - did you mean map?")
					} else if fun == "map" && feature.ret != FeatureTypePacket {
						panic("Unexpected map - did you mean apply?")
					}
					if feature.export {
						feature.function = strings.Join(compositeToCall(arguments), "")
					}
					if s, ok := selections[sel]; ok {
						stack = append([]featureWithType{featureWithType{arguments[1], feature.ret, feature.export, fun, true, "", feature.function}}, stack...)
						currentSelection = s
					} else {
						stack = append([]featureWithType{featureWithType{arguments[2], FeatureTypeSelection, false, "", false, sel, ""},
							featureWithType{arguments[1], feature.ret, feature.export, fun, true, "", feature.function}}, stack...)
					}
					continue MAIN
				} else {
					argumentTypes := getArgumentTypes(basetype, feature.ret, len(arguments)-1)
					argumentPos := make([]int, 0, len(arguments)-1)
					for i, f := range arguments[1:] {
						if pos, ok := seen[feature2id(f, argumentTypes[i])]; !ok {
							newstack := make([]featureWithType, len(arguments)-1)
							for i, arg := range arguments[1:] {
								newstack[i] = featureWithType{arg, argumentTypes[i], false, "", false, "", ""}
							}
							stack = append(append(newstack, feature), stack...)
							continue MAIN
						} else {
							argumentPos = append(argumentPos, pos)
						}
					}
					seen[id] = len(init)
					if feature.selection != "" {
						currentSelection = &selection{[]int{len(init)}, make(map[string]int, len(features))}
						selections[feature.selection] = currentSelection
					}
					if feature.export {
						feature.function = strings.Join(compositeToCall(arguments), "")
					}
					//select: set event true (event from logic + event from base)
					init = append(init, featureToInit{basetype, feature.ret, argumentPos, nil, fun == "select" || (fun == "select_slice" && len(argumentPos) == 2), feature.export, feature.composite, feature.function}) //fake BaseType?
				}
			}
		default:
			panic(fmt.Sprint("Don't know what to do with ", feature))
		}
		if feature.reset {
			currentSelection = mainSelection
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
			var basetype string
			if feature.composite != "" && feature.composite != "apply" && feature.composite != "map" {
				basetype = feature.composite
			} else if feature.function != "" {
				basetype = feature.function
			} else {
				basetype = feature.feature.BaseType()
			}
			basetypes = append(basetypes, basetype)

			nexport++
		}
		if feature.event {
			nevent++
		}
	}

	return FeatureListCreator{
		init,
		basetypes,
		exporter,
		func() *featureList {
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
				if len(feature.arguments) > 0 {
					args := make([]Feature, len(feature.arguments))
					for i, arg := range feature.arguments {
						args[i] = f[arg]
					}
					f[i].SetArguments(args)
				}
			}
			return &featureList{
				startup:  f,
				event:    event,
				export:   export,
				exporter: exporter,
			}
		},
	}
}

var graphTemplate = template.Must(template.New("callgraph").Parse(`digraph callgraph {
	label="call graph"
	node [shape=box, gradientangle=90]
	"event" [style="rounded,filled", fillcolor=red]
	{{ range $index, $element := .Nodes }}
	subgraph cluster_{{$index}} {
	{{ range $element.Nodes }}	"{{.Name}}" [label={{if .Label}}"{{.Label}}"{{else}}<{{.HTML}}>{{end}}{{range .Style}}, {{index . 0}}="{{index . 1}}"{{end}}]
	{{end}}}
	"export{{$index}}" [label="export",style="rounded,filled", fillcolor=red]
	{{ range $element.Export }}"{{.Name}}" [label={{if .Label}}"{{.Label}}"{{else}}<{{.HTML}}>{{end}}{{range .Style}}, {{index . 0}}="{{index . 1}}"{{end}}]
	"export{{$index}}" -> "{{.Name}}"
	{{end}}
	{{end}}
	{{ range .Edges }}"{{.Start}}"{{if .StartNode}}:{{.StartNode}}{{end}} -> "{{.Stop}}"{{if .StopNode}}:{{.StopNode}}{{end}}
	{{end}}
}
`))

type FeatureListCreatorList []FeatureListCreator

func (fl FeatureListCreatorList) creator() (ret []*featureList) {
	ret = make([]*featureList, len(fl))
	for i, fl := range fl {
		ret[i] = fl.creator()
	}
	return
}

func (fl FeatureListCreatorList) Fields() {
	for _, fl := range fl {
		fl.Fields()
	}
}

// CallGraph generates a call graph in the graphviz language and writes the result to w.
func (fl FeatureListCreatorList) CallGraph(w io.Writer) {
	toId := func(id, i int) string {
		return fmt.Sprintf("%d,%d", id, i)
	}
	styles := map[FeatureType][][]string{
		FeatureTypeFlow: {
			{"shape", "invhouse"},
			{"style", "filled"},
		},
		FeatureTypePacket: {
			{"style", "filled"},
		},
		featureTypeAny: {
			{"shape", "oval"},
		},
	}
	type Node struct {
		Name  string
		Label string
		HTML  string
		Style [][]string
	}
	type Edge struct {
		Start     string
		StartNode string
		Stop      string
		StopNode  string
	}
	type Subgraph struct {
		Nodes  []Node
		Export []Node
	}
	data := struct {
		Nodes []Subgraph
		Edges []Edge
	}{}
	for listID, fl := range fl {
		var nodes []Node
		export := make([]Node, len(fl.exporter))
		for i, exporter := range fl.exporter {
			export[i] = Node{Name: fmt.Sprintf("%p", exporter), Label: exporter.ID()} //FIXME: better style
		}
		for i, feature := range fl.init {
			var node Node
			node.Label = feature.feature.BaseType()
			if feature.composite == "apply" || feature.composite == "map" {
				node.Label = fmt.Sprintf("%s\\n%s", feature.composite, node.Label)
			} else if feature.composite != "" {
				node.Label = fmt.Sprintf("%s\\n%s", node.Label, feature.composite)
			}
			node.Name = toId(listID, i)
			if ret, ok := feature.ret.(FeatureType); ok {
				if len(feature.arguments) == 0 {
					node.Style = append(styles[ret], []string{"fillcolor", "green"})
				} else if len(feature.arguments) == 1 {
					if feature.composite == "" {
						node.Style = append(styles[ret], []string{"fillcolor", "orange"})
					} else {
						node.Style = append(styles[ret], []string{"fillcolor", "green:orange"})
					}
				} else {
					node.Style = append(styles[ret], []string{"fillcolor", "orange"})
					args := make([]string, len(feature.arguments))
					for i := range args {
						args[i] = fmt.Sprintf(`<TD PORT="%d" BORDER="1">%d</TD>`, i, i)
					}
					node.HTML = fmt.Sprintf(`<TABLE BORDER="0" CELLBORDER="0" CELLSPACING="2"><TR>%s</TR><TR><TD COLSPAN="%d">%s</TD></TR></TABLE>`, strings.Join(args, ""), len(feature.arguments), node.Label)
					node.Label = ""
				}
			} else {
				node.Label = fmt.Sprint(feature.ret)
				node.Style = styles[featureTypeAny]
			}

			nodes = append(nodes, node)
		}
		for i, feature := range fl.init {
			if feature.event {
				data.Edges = append(data.Edges, Edge{"event", "", toId(listID, i), ""})
			}
			if feature.export {
				data.Edges = append(data.Edges, Edge{toId(listID, i), "", fmt.Sprintf("export%d", listID), ""})
			}
			for index, j := range feature.arguments {
				index := strconv.Itoa(index)
				if len(feature.arguments) <= 1 {
					index = ""
				}
				data.Edges = append(data.Edges, Edge{toId(listID, j), "", toId(listID, i), index})
			}
		}
		data.Nodes = append(data.Nodes, Subgraph{nodes, export})
	}
	graphTemplate.Execute(w, data)
}
