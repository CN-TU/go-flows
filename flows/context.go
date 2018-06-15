package flows

type EventContext struct {
	when    DateTimeNanoseconds
	flow    Flow
	restart bool
	export  bool
}

func (ec *EventContext) initFlow(flow Flow) {
	ec.flow = flow
	ec.restart = false
	ec.export = false
}

func (ec *EventContext) When() DateTimeNanoseconds {
	return ec.when
}
