package operations

import (
	"github.com/CN-TU/go-flows/flows"
	ipfix "github.com/CN-TU/go-ipfix"
)

type slice struct {
	flows.BaseFeature
	start, stop, current int64
}

func (f *slice) SetArguments(arguments []flows.Feature) {
	f.start = flows.ToInt(arguments[0].Value())
	f.stop = flows.ToInt(arguments[1].Value())
}
func (f *slice) Start(ec *flows.EventContext) {
	f.BaseFeature.Start(ec)
	f.current = 0
}

func (f *slice) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.current >= f.start && f.current < f.stop {
		f.SetValue(new, context, f)
	}
	f.current++
}

func resolveSlice(args []ipfix.InformationElement) ipfix.InformationElement {
	return args[2]
}

func init() {
	flows.RegisterCustomFunction("slice", resolveSlice, flows.PacketFeature, func() flows.Feature { return &slice{} }, flows.Const, flows.Const, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////
