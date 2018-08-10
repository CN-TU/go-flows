package flows

import (
	"fmt"
	"reflect"

	"github.com/CN-TU/go-ipfix"
)

type constantFeature struct {
	value interface{}
}

func (f *constantFeature) Event(interface{}, *EventContext, interface{})    {}
func (f *constantFeature) FinishEvent()                                     {}
func (f *constantFeature) Value() interface{}                               { return f.value }
func (f *constantFeature) SetValue(interface{}, *EventContext, interface{}) {}
func (f *constantFeature) Start(*EventContext)                              {}
func (f *constantFeature) Stop(FlowEndReason, *EventContext)                {}
func (f *constantFeature) Variant() int                                     { return NoVariant }
func (f *constantFeature) Emit(interface{}, *EventContext, interface{})     {}
func (f *constantFeature) setDependent([]Feature)                           {}
func (f *constantFeature) SetArguments([]Feature)                           {}
func (f *constantFeature) IsConstant() bool                                 { return true }
func (f *constantFeature) setRecord(*record)                                {}

var _ Feature = (*constantFeature)(nil)

// newConstantMetaFeature creates a new constant feature, which holds the given value
func newConstantMetaFeature(value interface{}) featureMaker {
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
		panic(fmt.Sprint("Can't create constant of type ", reflect.TypeOf(value)))
	}
	feature := &constantFeature{value}
	return featureMaker{
		ret:  Const,
		make: func() Feature { return feature },
		ie:   ipfix.NewInformationElement(fmt.Sprintf("_const{%v}", value), 0, 0, t, 0),
	}
}

//apply and map pseudofeatures; those are handled during callgraph building
func init() {
	RegisterFunction("apply", FlowFeature, nil, FlowFeature, Selection)
	RegisterFunction("map", PacketFeature, nil, PacketFeature, Selection)
}
