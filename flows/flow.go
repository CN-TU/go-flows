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
	// SrcIP returns the source ip.
	SrcIP() []byte
	// DstIP returns the destination ip.
	DstIP() []byte
	// Proto returns the protocol.
	Proto() byte
	// SrcPort returns the source port.
	SrcPort() []byte
	// DstPort returns the destination port.
	DstPort() []byte
	// Hash returns a hash of the flow key.
	Hash() uint64
}

// Flow interface which needs to be implemented by every flow.
type Flow interface {
	Event(Event, Time)
	AddTimer(TimerID, TimerCallback, Time)
	HasTimer(TimerID) bool
	EOF(Time)
	Active() bool
	Key() FlowKey
	Init(*FlowTable, FlowKey, Time)
	Table() *FlowTable
	nextEvent() Time
	expire(Time)
}

// BaseFlow holds the base information a flow needs. Needs to be embedded into every flow.
type BaseFlow struct {
	key        FlowKey
	table      *FlowTable
	timers     funcEntries
	expireNext Time
	features   *featureList
	active     bool
}

// Stop destroys the resources associated with this flow. Call this to cancel the flow without exporting it or notifying the features.
func (flow *BaseFlow) Stop() {
	flow.table.remove(flow)
	flow.active = false
}

func (flow *BaseFlow) nextEvent() Time { return flow.expireNext }

// Active returns if the flow is still active.
func (flow *BaseFlow) Active() bool { return flow.active }

// Key returns the flow key belonging to this flow.
func (flow *BaseFlow) Key() FlowKey { return flow.key }

// Table returns the flow table belonging to this flow.
func (flow *BaseFlow) Table() *FlowTable { return flow.table }

func (flow *BaseFlow) expire(when Time) {
	if flow.expireNext == 0 {
		return
	}
	flow.expireNext = flow.timers.expire(when)
}

// AddTimer adds a new timer with the associated id, callback, at the time when. If the timerid already exists, then the old timer will be overwritten.
func (flow *BaseFlow) AddTimer(id TimerID, f TimerCallback, when Time) {
	flow.timers.addTimer(id, f, when)
	if when < flow.expireNext || flow.expireNext == 0 {
		flow.expireNext = when
	}
}

// HasTimer returns true if the timer with id is active.
func (flow *BaseFlow) HasTimer(id TimerID) bool {
	return flow.timers.hasTimer(id)
}

// Export exports the features of the flow with reason as FlowEndReason, at time when, with current time now. Afterwards the flow is removed from the table.
func (flow *BaseFlow) Export(reason FlowEndReason, when Time, now Time) {
	if !flow.active {
		return //WTF, this should not happen
	}
	flow.features.Stop(reason, when)
	flow.features.Export(now)
	flow.Stop()
}

func (flow *BaseFlow) idleEvent(expired Time, now Time) { flow.Export(FlowEndReasonIdle, expired, now) }
func (flow *BaseFlow) activeEvent(expired Time, now Time) {
	flow.Export(FlowEndReasonActive, expired, now)
}

// EOF stops the flow with forced end reason.
func (flow *BaseFlow) EOF(now Time) { flow.Export(FlowEndReasonForcedEnd, now, now) }

// Event handles the given event and the active and idle timers.
func (flow *BaseFlow) Event(event Event, when Time) {
	flow.AddTimer(timerIdle, flow.idleEvent, when+flow.table.idleTimeout)
	if !flow.HasTimer(timerActive) {
		flow.AddTimer(timerActive, flow.activeEvent, when+flow.table.activeTimeout)
	}
	flow.features.Event(event, when)
}

// Init initializes the flow and correspoding features. The associated table, key, and current time need to be provided.
func (flow *BaseFlow) Init(table *FlowTable, key FlowKey, time Time) {
	flow.key = key
	flow.table = table
	flow.timers = makeFuncEntries()
	flow.active = true
	flow.features = table.features.creator()
	flow.features.Init(flow)
	flow.features.Start(time)
}
