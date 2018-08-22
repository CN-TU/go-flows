package operations

import (
	"bytes"
	"fmt"

	"github.com/CN-TU/go-flows/flows"
	ipfix "github.com/CN-TU/go-ipfix"
)

type join struct {
	flows.MultiBasePacketFeature
}

func (f *join) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	f.SetValue(values, context, f)
}

func init() {
	flows.RegisterFunction("join", "returns arguments as list", flows.PacketFeature, func() flows.Feature { return &join{} }, flows.PacketFeature, flows.Ellipsis)
}

////////////////////////////////////////////////////////////////////////////////

type accumulate struct {
	flows.BaseFeature
	vector []interface{}
}

func (f *accumulate) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.vector = make([]interface{}, 0)
}

func (f *accumulate) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if len(f.vector) != 0 {
		f.SetValue(f.vector, context, f)
	}
}

func (f *accumulate) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.vector = append(f.vector, new)
}

func resolveAccumulate(args []ipfix.InformationElement) ipfix.InformationElement {
	return ipfix.NewBasicList("accumulate", args[0], 0)
}

//FIXME: this has a bad name
func init() {
	flows.RegisterCustomFunction("accumulate", "returns per-packet values as a list", resolveAccumulate, flows.FlowFeature, func() flows.Feature { return &accumulate{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////

type concatenate struct {
	flows.BaseFeature
	buffer *bytes.Buffer
}

func (f *concatenate) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.buffer = new(bytes.Buffer)
}

func (f *concatenate) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if val, ok := new.([]byte); ok {
		fmt.Fprintf(f.buffer, "%s", val)
	} else {
		fmt.Fprint(f.buffer, new)
	}
}

func (f *concatenate) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.buffer.Bytes(), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("concatenate", "concatenates per-packet values into a string", ipfix.OctetArrayType, 0, flows.FlowFeature, func() flows.Feature { return &concatenate{} }, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////
