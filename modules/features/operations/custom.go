package operations

import "github.com/CN-TU/go-flows/flows"

type join struct {
	flows.MultiBaseFeature
}

func (f *join) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	f.SetValue(values, context, f)
}

func init() {
	flows.RegisterFunction("join", flows.MatchType, func() flows.Feature { return &join{} }, flows.MatchType, flows.Ellipsis)
}

////////////////////////////////////////////////////////////////////////////////
