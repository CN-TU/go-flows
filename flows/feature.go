package flows

// Feature interfaces, which all features need to implement
type Feature interface {
	// Event gets called for every event. Data is provided via the first argument and a context providing addional information/control via the second argument.
	Event(interface{}, *EventContext, interface{})
	// FinishEvent gets called after every Feature was processed for the current event.
	FinishEvent(*EventContext)
	// Value provides the current stored value.
	Value() interface{}
	// SetValue stores a new value with the associated time.
	SetValue(interface{}, *EventContext, interface{})
	// Start gets called when the flow starts.
	Start(*EventContext)
	// Stop gets called with an end reason and time when a flow stops
	Stop(FlowEndReason, *EventContext)
	// Variant must return the current variant id, if the Feature can represent multiple types (e.g. ipv4Address vs ipv6Address). Must be NoVariant otherwise.
	Variant() int
	// Emit sends value new, with time when, and source self to the dependent Features
	Emit(new interface{}, when *EventContext, self interface{})
	// IsConstant must return true, if this feature is a constant
	IsConstant() bool
	// setDependent is used internally for setting features that depend on this features' value.
	setDependent([]int)
}

// FeatureWithArguments represents a feature that needs arguments (e.g. MultiBase*Feature or select)
type FeatureWithArguments interface {
	// SetArguments gets called during Feature initialization with the arguments of the features (needed for operations)
	SetArguments([]Feature)
}

// NoVariant represents the value returned from Variant if this Feature has only a single type.
const NoVariant = -1

// NoopFeature implements the feature interface and represents a feature without built in functionality.
//
// Use this as a base for features that don't hold values and only emit them. Good examples are filter or control features.
type NoopFeature struct{}

// Event is an empty function to ignore every event. Overload this if you need events.
func (f *NoopFeature) Event(interface{}, *EventContext, interface{}) {}

// FinishEvent is an empty function to ignore end of event-processing. Overload this if you need such events.
func (f *NoopFeature) FinishEvent(*EventContext) {}

// Value returns the empty value (nil). Overload this if you need to return a value.
func (f *NoopFeature) Value() interface{} { return nil }

// SetValue is an empty function to ignore constant arguments. Overload this if you need this data.
func (f *NoopFeature) SetValue(new interface{}, when *EventContext, self interface{}) {}

// Start is an empty function to ignore start events. Overload this if you need start events.
func (f *NoopFeature) Start(*EventContext) {}

// Stop is an empty function to ignore stop events. Overload this if you need stop events.
func (f *NoopFeature) Stop(FlowEndReason, *EventContext) {}

// Variant returns NoVariant. Overload this if your feature has multiple types.
func (f *NoopFeature) Variant() int { return NoVariant }

// Emit is an empty function to ignore emitted values. Overload this if you need to emit values.
func (f *NoopFeature) Emit(new interface{}, context *EventContext, self interface{}) {}

// setDependent is an empty function to adding dependent features. Overload this if you need to support dependent features.
func (f *NoopFeature) setDependent(dep []int) {}

// IsConstant returns false to signal that this feature is not a constant. Overload this if you need to emulate a constant.
func (f *NoopFeature) IsConstant() bool { return false }

// check if EmptyBaseFeature fulfills Feature interface
var _ Feature = (*NoopFeature)(nil)

// EmptyBaseFeature implements the feature interface with some added functionality to support the most basic operation (e.g. value passing to dependent features).
//
// Use this as a base for features that don't need to hold values.
type EmptyBaseFeature struct {
	dependent []int
}

// Event is an empty function to ignore every event. Overload this if you need events.
func (f *EmptyBaseFeature) Event(interface{}, *EventContext, interface{}) {}

// FinishEvent propagates this event to all dependent features. Do not overload unless you know what you're doing!
func (f *EmptyBaseFeature) FinishEvent(context *EventContext) {
	for _, v := range f.dependent {
		context.record.features[v].FinishEvent(context)
	}
}

// Value returns the empty value (nil). Overload this if you need to return a value.
func (f *EmptyBaseFeature) Value() interface{} { return nil }

// SetValue is an empty function to ignore constant arguments. Overload this if you need this data.
func (f *EmptyBaseFeature) SetValue(new interface{}, when *EventContext, self interface{}) {}

// Start is an empty function to ignore start events. Overload this if you need start events.
func (f *EmptyBaseFeature) Start(*EventContext) {}

// Stop is an empty function to ignore stop events. Overload this if you need stop events.
func (f *EmptyBaseFeature) Stop(FlowEndReason, *EventContext) {}

// Variant returns NoVariant. Overload this if your feature has multiple types.
func (f *EmptyBaseFeature) Variant() int { return NoVariant }

// Emit propagates the new value to all dependent features. Do not overload unless you know what you're doing!
func (f *EmptyBaseFeature) Emit(new interface{}, context *EventContext, self interface{}) {
	for _, v := range f.dependent {
		context.record.features[v].Event(new, context, self)
	}
}

// setDependent sets the given list of features for forwarding events to
func (f *EmptyBaseFeature) setDependent(dep []int) { f.dependent = dep }

// IsConstant returns false to signal that this feature is not a constant. Overload this if you need to emulate a constant.
func (f *EmptyBaseFeature) IsConstant() bool { return false }

// check if EmptyBaseFeature fulfills Feature interface
var _ Feature = (*EmptyBaseFeature)(nil)

// BaseFeature includes all the basic functionality to fulfill the Feature interface.
//
// In most cases you need this as the base for implementing feature
type BaseFeature struct {
	EmptyBaseFeature
	value interface{}
}

// Start clears the held value. You must all this in your feature if you override Start!
func (f *BaseFeature) Start(*EventContext) { f.value = nil }

// Value returns the current value. Do not overload unless you know what you're doing!
func (f *BaseFeature) Value() interface{} { return f.value }

// SetValue sets a new value and forwards it to the dependent features. Do not overload unless you know what you're doing!
func (f *BaseFeature) SetValue(new interface{}, context *EventContext, self interface{}) {
	f.value = new
	if new != nil {
		f.Emit(new, context, self)
	}
}

// For speed purposes, features with multiple arguments are split into 3 cathegories:
// - singleMultiEvent: one non const argument
// - dualMultiEvent: two non const arguments
// - genericMultiEvent: more than two non const arguments

type multiEvent interface {
	// Either returns all the argument-values as a list in argument order, or nil if not every argument emitted a value
	CheckAll(interface{}, interface{}) []interface{}
	// Resets event tracking - must be called from FinishEvent
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

// MultiBasePacketFeature extends BaseFeature with event tracking.
//
// Use this as base for creating new features returning PacketFeature with multiple arguments.
type MultiBasePacketFeature struct {
	BaseFeature
	eventReady multiEvent
}

// EventResult returns the list of values for a multievent or nil if not every argument had an event
func (f *MultiBasePacketFeature) EventResult(new interface{}, which interface{}) []interface{} {
	return f.eventReady.CheckAll(new, which)
}

// FinishEvent resets event tracking for all the arguments. Do not overload unless you know what you're doing!
func (f *MultiBasePacketFeature) FinishEvent(context *EventContext) {
	f.eventReady.Reset()
	f.BaseFeature.FinishEvent(context)
}

// SetArguments prepares the internal argument list for event tracking. Do not overload unless you know what you're doing!
func (f *MultiBasePacketFeature) SetArguments(args []Feature) {
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

// MultiBaseFlowFeature extends BaseFeature with argument tracking.
//
// Use this as base for creating new features returning FlowFeature with multiple arguments.
type MultiBaseFlowFeature struct {
	BaseFeature
	arguments []Feature
}

// SetArguments prepares the internal argument list for argument tracking. Do not overload unless you know what you're doing!
func (f *MultiBaseFlowFeature) SetArguments(args []Feature) {
	f.arguments = args
}

// GetValues returns the values of every argument
func (f *MultiBaseFlowFeature) GetValues() []interface{} {
	ret := make([]interface{}, len(f.arguments))
	ret = ret[:len(f.arguments)]
	for i, c := range f.arguments {
		ret[i] = c.Value()
	}
	return ret
}
