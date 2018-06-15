package flows

type EventContext struct {
	when    DateTimeNanoseconds
	flow    Flow
	reason  FlowEndReason
	now     bool
	stop    bool
	export  bool
	restart bool
}

func (ec *EventContext) initFlow(flow Flow) {
	ec.flow = flow
}

func (ec *EventContext) clear() {
	ec.now = false
	ec.stop = false
	ec.export = false
	ec.restart = false
}

func (ec *EventContext) When() DateTimeNanoseconds {
	return ec.when
}

// Stop removes the current record and discards the current event
func (ec *EventContext) Stop() {
	ec.stop = true
}

// Export exports the current record now or after the event. WARNING: If now is true, the current event happens again after this!
func (ec *EventContext) Export(now bool, reason FlowEndReason) {
	ec.now = now
	ec.export = true
	ec.reason = reason
}

// Restart restarts the current record now or after the event WARNING: If now is true, the current event happens again after this!
func (ec *EventContext) Restart(now bool) {
	ec.now = now
	ec.restart = true
}
