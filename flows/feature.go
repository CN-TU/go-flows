package flows

import (
	"fmt"
	"strings"

	"github.com/CN-TU/go-ipfix"
)

// generates a textual representation of a feature usable for comparison loosely based on <type>name
// or [<type>name,argumentA,...] (with argumentX being an id) for composites
func feature2id(feature interface{}, ret FeatureType) string {
	switch feature := feature.(type) {
	case string:
		return fmt.Sprintf("<%d>%s", ret, feature)
	case bool, float64, int64, uint64, int, uint:
		return fmt.Sprintf("Const{%v}", feature)
	case []interface{}:
		features := make([]string, len(feature))
		f, found := getFeature(feature[0].(string), ret, len(feature)-1)
		if !found {
			panic(fmt.Sprintf("Feature %s with return type %s and %d arguments not found!", feature[0].(string), ret, len(feature)-1))
		}
		arguments := append([]FeatureType{ret}, f.getArguments(ret, len(feature)-1)...)
		for i, f := range feature {
			features[i] = feature2id(f, arguments[i])
		}
		return "[" + strings.Join(features, ",") + "]"
	default:
		panic(fmt.Sprint("Don't know what to do with ", feature))
	}
}

// Feature interfaces, which all features need to implement
type Feature interface {
	// Event gets called for every event. Data is provided via the first argument and current time via the second.
	Event(interface{}, EventContext, interface{})
	// FinishEvent gets called after every Event happened
	FinishEvent()
	// Value provides the current stored value.
	Value() interface{}
	// SetValue stores a new value with the associated time.
	SetValue(interface{}, EventContext, interface{})
	// Start gets called when the flow starts.
	Start(EventContext)
	// Stop gets called with an end reason and time when a flow stops
	Stop(FlowEndReason, EventContext)
	// Type returns the InformationElement
	Variant() int
	// Emit sends value new, with time when, and source self to the dependent Features
	Emit(new interface{}, when EventContext, self interface{})
	setDependent([]Feature)
	SetArguments([]Feature)
	IsConstant() bool
}

const NoVariant = -1

type EmptyBaseFeature struct {
	dependent []Feature
}

func (f *EmptyBaseFeature) Event(interface{}, EventContext, interface{}) {}
func (f *EmptyBaseFeature) FinishEvent() {
	for _, v := range f.dependent {
		v.FinishEvent()
	}
}
func (f *EmptyBaseFeature) Value() interface{}                                            { return nil }
func (f *EmptyBaseFeature) SetValue(new interface{}, when EventContext, self interface{}) {}
func (f *EmptyBaseFeature) Start(EventContext)                                            {}
func (f *EmptyBaseFeature) Stop(FlowEndReason, EventContext)                              {}
func (f *EmptyBaseFeature) Variant() int                                                  { return NoVariant }
func (f *EmptyBaseFeature) Emit(new interface{}, context EventContext, self interface{}) {
	for _, v := range f.dependent {
		v.Event(new, context, self)
	}
}
func (f *EmptyBaseFeature) setDependent(dep []Feature) { f.dependent = dep }
func (f *EmptyBaseFeature) SetArguments([]Feature)     {}
func (f *EmptyBaseFeature) IsConstant() bool           { return false }

var _ Feature = (*EmptyBaseFeature)(nil)

// BaseFeature includes all the basic functionality to fulfill the Feature interface.
// Embedd this struct for creating new features.
type BaseFeature struct {
	EmptyBaseFeature
	value interface{}
}

func (f *BaseFeature) Value() interface{} { return f.value }
func (f *BaseFeature) SetValue(new interface{}, context EventContext, self interface{}) {
	f.value = new
	if new != nil {
		f.Emit(new, context, self)
	}
}

type multiEvent interface {
	CheckAll(interface{}, interface{}) []interface{}
	Reset()
}

// singleMultiEvent (one not const) (every event is the right one!)
type singleMultiEvent struct {
	c   []interface{}
	ret []interface{}
}

func (m *singleMultiEvent) CheckAll(new interface{}, which interface{}) []interface{} {
	ret := m.ret[:len(m.c)]
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
		ret = ret[:len(m.c)]
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
	ret = ret[:len(m.c)]
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
	featurelist = featurelist[:len(args)]
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
		f.eventReady = &singleMultiEvent{featurelist, make([]interface{}, len(featurelist))}
	case 2:
		event := &dualMultiEvent{} //FIXME preallocate ret
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
		f.eventReady = &genericMultiEvent{c: featurelist, nc: nc, state: make([]bool, len(features))} //FIXME preallocate ret
	}
}

type featureMaker struct {
	ret       FeatureType
	make      func() Feature
	arguments []FeatureType
	ie        ipfix.InformationElement
	variants  []ipfix.InformationElement
	function  bool
}

func (f featureMaker) String() string {
	return fmt.Sprintf("<%s>%s(%s)", f.ret, f.ie, f.arguments)
}

func (f featureMaker) getArguments(ret FeatureType, nargs int) []FeatureType {
	if f.arguments[len(f.arguments)-1] == Ellipsis {
		r := make([]FeatureType, nargs)
		last := len(f.arguments) - 2
		variadic := f.arguments[last]
		if variadic == MatchType {
			variadic = ret
		}
		for i := 0; i < nargs; i++ {
			if i > last {
				r[i] = variadic
			} else {
				if f.arguments[i] == MatchType {
					r[i] = ret
				} else {
					r[i] = f.arguments[i]
				}
			}
		}
		return r
	}
	if f.ret == MatchType {
		r := make([]FeatureType, nargs)
		for i := range r {
			if f.arguments[i] == MatchType {
				r[i] = ret
			} else {
				r[i] = f.arguments[i]
			}
		}
		return r
	}
	return f.arguments
}
