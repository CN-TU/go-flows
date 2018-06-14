package flows

type EventContext struct {
	when    DateTimeNanoseconds
	flow    Flow
	restart bool
	export  bool
}

func (ec *EventContext) FutureEventContext(offset DateTimeNanoseconds) *EventContext {
	return &EventContext{
		when: ec.when + offset,
		flow: ec.flow,
	}
}

func (ec *EventContext) When() DateTimeNanoseconds {
	return ec.when
}
