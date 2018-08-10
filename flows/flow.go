package flows

// FlowEndReason holds the flowEndReason as specified by RFC5102
type FlowEndReason byte

const (
	// FlowEndReasonIdle idle timeout as specified by RFC5102
	FlowEndReasonIdle FlowEndReason = 1
	// FlowEndReasonActive active timeout as specified by RFC5102
	FlowEndReasonActive FlowEndReason = 2
	// FlowEndReasonEnd end of flow as specified by RFC5102
	FlowEndReasonEnd FlowEndReason = 3
	// FlowEndReasonForcedEnd forced end of flow as specified by RFC5102
	FlowEndReasonForcedEnd FlowEndReason = 4
	// FlowEndReasonLackOfResources lack of resources as specified by RFC5102
	FlowEndReasonLackOfResources FlowEndReason = 5
)

// FlowKey interface. Every flow key needs to implement this interface.
type FlowKey interface {
	// Hash returns a hash of the flow key.
	Hash() uint64
}

// Flow interface is the primary object for flows. This gets created in the flow table for non-existing
// flows and lives until forced end (EOF), timer expiry (active/idle timeout), or derived from the data
// (e.g. tcp fin/rst)
type Flow interface {
	//// Functions for timer handling
	//// ------------------------------------------------------------------
	// AddTimer adds a timer with the specific id and callback, which will be called at the given point in time
	AddTimer(TimerID, TimerCallback, DateTimeNanoseconds)
	// HasTimer returns true, if an active timer exists for the given timer id
	HasTimer(TimerID) bool
	// RemoveTimer cancels the timer with the given id
	RemoveTimer(TimerID)

	//// Functions for event handling
	//// ------------------------------------------------------------------
	// Event gets called by the flow table for every event that belongs to this flow
	Event(Event, *EventContext)
	// EOF gets called from the main program after the last event was read from the input
	EOF(*EventContext)

	//// Functions for querying flow status
	//// ------------------------------------------------------------------
	// Active returns true if this flow is still active
	Active() bool
	// Key returns the flow key
	Key() FlowKey
	// ID returns the flow id
	ID() uint64
	// Table returns the flow table this flow belongs to
	Table() *FlowTable

	//// Functions for flow initialization
	//// ------------------------------------------------------------------
	// Init gets called by the flow table to provide the flow table, a key, and a flow id
	Init(*FlowTable, FlowKey, *EventContext, uint64)

	// internal function which returns the point in time the next event will happen
	nextEvent() DateTimeNanoseconds
	// expire gets called by the flow table for handling timer expiry
	expire(*EventContext)
}

//FlowOptions applying to each flow
type FlowOptions struct {
	// ActiveTimeout is the active timeout in nanoseconds
	ActiveTimeout DateTimeNanoseconds
	// IdleTimeout is the idle timeout in nanoseconds
	IdleTimeout DateTimeNanoseconds
}

// BaseFlow holds the base information a flow needs. Needs to be embedded into every flow.
type BaseFlow struct {
	key        FlowKey
	table      *FlowTable
	timers     funcEntries
	expireNext DateTimeNanoseconds
	records    Record
	id         uint64
	active     bool
}

// Stop destroys the resources associated with this flow. Call this to cancel the flow without exporting it or notifying the features.
func (flow *BaseFlow) Stop() {
	flow.table.remove(flow)
	flow.active = false
}

func (flow *BaseFlow) nextEvent() DateTimeNanoseconds { return flow.expireNext }

// Active returns if the flow is still active.
func (flow *BaseFlow) Active() bool { return flow.active }

// Key returns the flow key belonging to this flow.
func (flow *BaseFlow) Key() FlowKey { return flow.key }

// Table returns the flow table belonging to this flow.
func (flow *BaseFlow) Table() *FlowTable { return flow.table }

func (flow *BaseFlow) expire(context *EventContext) {
	if flow.expireNext == 0 {
		return
	}
	flow.expireNext = flow.timers.expire(context.when)
}

// AddTimer adds a new timer with the associated id, callback, at the time when. If the timerid already exists, then the old timer will be overwritten.
func (flow *BaseFlow) AddTimer(id TimerID, f TimerCallback, when DateTimeNanoseconds) {
	flow.timers.addTimer(id, f, when)
	if when < flow.expireNext || flow.expireNext == 0 {
		flow.expireNext = when
	}
}

// HasTimer returns true if the timer with id is active.
func (flow *BaseFlow) HasTimer(id TimerID) bool {
	return flow.timers.hasTimer(id)
}

// RemoveTimer deletes the timer with the given id.
func (flow *BaseFlow) RemoveTimer(id TimerID) {
	flow.timers.removeTimer(id)
}

// Export exports the features of the flow with reason as FlowEndReason, at time when, with current time now. Afterwards the flow is removed from the table.
func (flow *BaseFlow) Export(reason FlowEndReason, context *EventContext, now DateTimeNanoseconds) {
	if !flow.active {
		return //WTF, this should not happen
	}
	context.hard = true
	flow.records.Export(reason, context, now)
	flow.Stop()
}

// ExportWithoutContext exports the features of the flow (see Export). This function can be used, when no context is available.
func (flow *BaseFlow) ExportWithoutContext(reason FlowEndReason, expire, now DateTimeNanoseconds) {
	context := &EventContext{
		when: expire,
	}
	context.initFlow(flow)
	flow.Export(reason, context, now)
}

func (flow *BaseFlow) idleEvent(expires, now DateTimeNanoseconds) {
	flow.ExportWithoutContext(FlowEndReasonIdle, expires, now)
}
func (flow *BaseFlow) activeEvent(expires, now DateTimeNanoseconds) {
	flow.ExportWithoutContext(FlowEndReasonActive, expires, now)
}

// EOF stops the flow with forced end reason.
func (flow *BaseFlow) EOF(context *EventContext) {
	flow.Export(FlowEndReasonForcedEnd, context, context.when)
}

// Event handles the given event and the active and idle timers.
func (flow *BaseFlow) Event(event Event, context *EventContext) {
	context.initFlow(flow)
	if flow.table.ActiveTimeout != 0 {
		flow.AddTimer(TimerIdle, flow.idleEvent, context.when+flow.table.IdleTimeout)
	}
	flow.records.Event(event, context)
	if !flow.records.Active() {
		flow.Stop()
		return
	}
	if flow.table.ActiveTimeout == 0 {
		flow.Export(FlowEndReasonEnd, context, context.when)
	}
}

// ID returns the flow ID within the flow table
func (flow *BaseFlow) ID() uint64 {
	return flow.id
}

// Init initializes the flow and correspoding features. The associated table, key, and current time need to be provided.
func (flow *BaseFlow) Init(table *FlowTable, key FlowKey, context *EventContext, id uint64) {
	flow.key = key
	flow.table = table
	flow.timers = makeFuncEntries()
	flow.active = true
	flow.records = table.records.make()
	flow.id = id
	context.initFlow(flow)
	if flow.table.ActiveTimeout != 0 {
		flow.AddTimer(TimerActive, flow.activeEvent, context.when+flow.table.ActiveTimeout)
	}
}
