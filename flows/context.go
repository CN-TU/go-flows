package flows

// EventContext holds additional data for an event (e.g. time) and allows features to modify flow behaviour
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

// When returns the time, the event happened
func (ec *EventContext) When() DateTimeNanoseconds {
	return ec.when
}

// Event must be called by filter-features to forward an Event down the processing chain
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

// IsHard returns true, if the current Stop event is non-cancelable (e.g. EOF)
func (ec *EventContext) IsHard() bool {
	return ec.hard
}

// Flow returns the current flow
func (ec *EventContext) Flow() Flow {
	return ec.flow
}
