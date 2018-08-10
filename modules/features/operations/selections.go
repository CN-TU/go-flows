package operations

import "github.com/CN-TU/go-flows/flows"

type selectF struct {
	flows.EmptyBaseFeature
	sel bool
}

func (f *selectF) Start(*flows.EventContext) { f.sel = false }

func (f *selectF) Event(new interface{}, context *flows.EventContext, src interface{}) {
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
	flows.EmptyBaseFeature
	start, stop, current int64
}

func (f *selectS) SetArguments(arguments []flows.Feature) {
	f.start = flows.ToInt(arguments[0].Value())
	f.stop = flows.ToInt(arguments[1].Value())
}
func (f *selectS) Start(*flows.EventContext) { f.current = 0 }

func (f *selectS) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.current >= f.start && f.current < f.stop {
		f.Emit(new, context, nil)
	}
	f.current++
}

func init() {
	flows.RegisterFunction("select", flows.Selection, func() flows.Feature { return &selectF{} }, flows.PacketFeature)
	flows.RegisterFunction("select_slice", flows.Selection, func() flows.Feature { return &selectS{} }, flows.Const, flows.Const)
	flows.RegisterFunction("select_slice", flows.Selection, func() flows.Feature { return &selectS{} }, flows.Const, flows.Const, flows.Selection)
}

////////////////////////////////////////////////////////////////////////////////
