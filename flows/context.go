package flows

type EventContext struct {
	when    DateTimeNanoseconds
	flow    Flow
	reason  FlowEndReason
	event   func(interface{}, *EventContext, interface{})
	now     bool
	stop    bool
	export  bool
	restart bool
	keep    bool
	hard    bool
}

func (ec *EventContext) initFlow(flow Flow) {
	ec.flow = flow
}

func (ec *EventContext) clear() {
	ec.now = false
	ec.stop = false
	ec.export = false
	ec.restart = false
	ec.keep = false
	ec.hard = false
}

func (ec *EventContext) When() DateTimeNanoseconds {
	return ec.when
}

func (ec *EventContext) Event(new interface{}, context *EventContext, data interface{}) {
	ec.event(new, context, data)
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

// Keep keeps this record alive for filters
func (ec *EventContext) Keep() {
	ec.keep = true
}

func (ec *EventContext) IsHard() bool {
	return ec.hard
}
