package operations

import (
	"errors"

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

func resolveSlice(args []ipfix.InformationElement) (ipfix.InformationElement, error) {
	if len(args) != 3 {
		return ipfix.InformationElement{}, errors.New("slice must have exactly 3 arguments")
	}
	return args[2], nil
}

func init() {
	flows.RegisterCustomFunction("slice", "gets third_argument[first_argument, second_argument]; indexing is like in Python", resolveSlice, flows.PacketFeature, func() flows.Feature { return &slice{} }, flows.Const, flows.Const, flows.PacketFeature)
}

////////////////////////////////////////////////////////////////////////////////
