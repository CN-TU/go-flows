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

// Flow interface which needs to be implemented by every flow.
type Flow interface {
	Event(Event, DateTimeNanoseconds)
	AddTimer(TimerID, TimerCallback, EventContext)
	HasTimer(TimerID) bool
	EOF(DateTimeNanoseconds)
	Active() bool
	Key() FlowKey
	Init(*FlowTable, FlowKey, DateTimeNanoseconds)
	Table() *FlowTable
	nextEvent() DateTimeNanoseconds
	expire(DateTimeNanoseconds)
}

//FlowOptions applying to each flow
type FlowOptions struct {
	ActiveTimeout DateTimeNanoseconds
	IdleTimeout   DateTimeNanoseconds
	PerPacket     bool
}

type EventContext struct {
	When   DateTimeNanoseconds
	Flow   Flow
	record *record
}

func (ec EventContext) FutureEventContext(offset DateTimeNanoseconds) (ret EventContext) {
	ret = ec
	ret.When += offset
	return
}

// BaseFlow holds the base information a flow needs. Needs to be embedded into every flow.
type BaseFlow struct {
	key        FlowKey
	table      *FlowTable
	timers     funcEntries
	expireNext DateTimeNanoseconds
	records    Record
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

func (flow *BaseFlow) expire(when DateTimeNanoseconds) {
	if flow.expireNext == 0 {
		return
	}
	flow.expireNext = flow.timers.expire(when)
}

// AddTimer adds a new timer with the associated id, callback, at the time when. If the timerid already exists, then the old timer will be overwritten.
func (flow *BaseFlow) AddTimer(id TimerID, f TimerCallback, context EventContext) {
	flow.timers.addTimer(id, f, context)
	if context.When < flow.expireNext || flow.expireNext == 0 {
		flow.expireNext = context.When
	}
}

// HasTimer returns true if the timer with id is active.
func (flow *BaseFlow) HasTimer(id TimerID) bool {
	return flow.timers.hasTimer(id)
}

func (flow *BaseFlow) ExportWithoutContext(reason FlowEndReason, now DateTimeNanoseconds) {
	context := EventContext{
		When: now,
		Flow: flow,
	}
	flow.Export(reason, context, now)
}

// Export exports the features of the flow with reason as FlowEndReason, at time when, with current time now. Afterwards the flow is removed from the table.
func (flow *BaseFlow) Export(reason FlowEndReason, context EventContext, now DateTimeNanoseconds) {
	if !flow.active {
		return //WTF, this should not happen
	}
	flow.records.Stop(reason, context)
	flow.records.Export(context)
	flow.Stop()
}

func (flow *BaseFlow) idleEvent(context EventContext, now DateTimeNanoseconds) {
	flow.Export(FlowEndReasonIdle, context, now)
}
func (flow *BaseFlow) activeEvent(context EventContext, now DateTimeNanoseconds) {
	flow.Export(FlowEndReasonActive, context, now)
}

// EOF stops the flow with forced end reason.
func (flow *BaseFlow) EOF(now DateTimeNanoseconds) {
	flow.ExportWithoutContext(FlowEndReasonForcedEnd, now)
}

// Event handles the given event and the active and idle timers.
func (flow *BaseFlow) Event(event Event, when DateTimeNanoseconds) {
	context := EventContext{
		When: when,
		Flow: flow,
	}
	if !flow.table.PerPacket {
		flow.AddTimer(timerIdle, flow.idleEvent, context.FutureEventContext(flow.table.IdleTimeout))
		if !flow.HasTimer(timerActive) {
			flow.AddTimer(timerActive, flow.activeEvent, context.FutureEventContext(flow.table.ActiveTimeout))
		}
	}
	flow.records.Event(event, context)
	if !flow.records.Active() {
		flow.Stop()
		return
	}
	if flow.table.PerPacket {
		flow.Export(FlowEndReasonEnd, context, when)
	}
}

// Init initializes the flow and correspoding features. The associated table, key, and current time need to be provided.
func (flow *BaseFlow) Init(table *FlowTable, key FlowKey, time DateTimeNanoseconds) {
	flow.key = key
	flow.table = table
	flow.timers = makeFuncEntries()
	flow.active = true
	flow.records = table.records.make()
	context := EventContext{
		When: time,
		Flow: flow,
	}
	flow.records.Start(context)
}
