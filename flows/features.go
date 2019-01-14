package flows

import (
	"fmt"

	"github.com/CN-TU/go-ipfix"
)

type constantFeature struct {
	value interface{}
}

func (f *constantFeature) Event(interface{}, *EventContext, interface{})    {}
func (f *constantFeature) FinishEvent(*EventContext)                        {}
func (f *constantFeature) Value() interface{}                               { return f.value }
func (f *constantFeature) SetValue(interface{}, *EventContext, interface{}) {}
func (f *constantFeature) Start(*EventContext)                              {}
func (f *constantFeature) Stop(FlowEndReason, *EventContext)                {}
func (f *constantFeature) Variant() int                                     { return NoVariant }
func (f *constantFeature) Emit(interface{}, *EventContext, interface{})     {}
func (f *constantFeature) setDependent([]int)                               {}
func (f *constantFeature) IsConstant() bool                                 { return true }

var _ Feature = (*constantFeature)(nil)

// newConstantMetaFeature creates a new constant feature, which holds the given value
func newConstantMetaFeature(value interface{}) (featureMaker, error) {
	var t ipfix.Type
	switch cv := value.(type) {
	case bool:
		t = ipfix.BooleanType
	case float64:
		t = ipfix.Float64Type
	case int:
		value = int64(cv)
		t = ipfix.Signed64Type
	case int64:
		t = ipfix.Signed64Type
	case uint:
		value = uint64(cv)
		t = ipfix.Unsigned64Type
	case uint64:
		t = ipfix.Unsigned64Type
	default:
		return featureMaker{}, fmt.Errorf("can't create constant of type %T", value)
	}
	feature := &constantFeature{value}
	return featureMaker{
		ret:  Const,
		make: func() Feature { return feature },
		ie:   ipfix.NewInformationElement(fmt.Sprintf("_const{%v}", value), 0, 0, t, 0),
	}, nil
}

//apply and map pseudofeatures; those are handled during callgraph building
func init() {
	RegisterFunction("apply", "returns a single feature value for the selection of objects", FlowFeature, nil, FlowFeature, Selection)
	RegisterFunction("map", "returns a feature value for each object in selection", PacketFeature, nil, PacketFeature, Selection)
}

// select and select_slice features; needed by the base implementation

type selectF struct {
	EmptyBaseFeature
	sel bool
}

func (f *selectF) Start(*EventContext) { f.sel = false }

func (f *selectF) Event(new interface{}, context *EventContext, src interface{}) {
	/* If src is not nil we got an event from the argument -> Store the boolean value (This always happens before events from the flow)
	   otherwise we have an event from the flow -> forward it in case we should and reset sel
	*/
	if src != nil {
		f.sel = new.(bool)
	} else {
		if f.sel {
			f.Emit(new, context, nil)
			f.sel = false
		}
	}
}

type selectS struct {
	EmptyBaseFeature
	start, stop, current int64
}

func (f *selectS) SetArguments(arguments []int, features []Feature) {
	f.start = ToInt(features[arguments[0]].Value())
	f.stop = ToInt(features[arguments[1]].Value())
}
func (f *selectS) Start(*EventContext) { f.current = 0 }

func (f *selectS) Event(new interface{}, context *EventContext, src interface{}) {
	if f.current >= f.start && f.current < f.stop {
		f.Emit(new, context, nil)
	}
	f.current++
}

func init() {
	RegisterFunction("select", "select a subsect of packets", Selection, func() Feature { return &selectF{} }, PacketFeature)
	RegisterFunction("select_slice", "selects a slice from the first value to the second value, with Python-like indexing (if a <selection is not provided, default to selecting everything)", Selection, func() Feature { return &selectS{} }, Const, Const)
	RegisterFunction("select_slice", "selects a slice from the first value to the second value, with Python-like indexing (if a <selection is not provided, default to selecting everything)", Selection, func() Feature { return &selectS{} }, Const, Const, Selection)
}
