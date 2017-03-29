package flows

import (
	"sort"
	"sync"
)

type FlowEndReason byte

const (
	FlowEndReasonIdle            FlowEndReason = 1
	FlowEndReasonActive          FlowEndReason = 2
	FlowEndReasonEnd             FlowEndReason = 3
	FlowEndReasonForcedEnd       FlowEndReason = 4
	FlowEndReasonLackOfResources FlowEndReason = 5
)

func (fe FlowEndReason) String() string {
	switch fe {
	case FlowEndReasonIdle:
		return "IdleTimeout"
	case FlowEndReasonActive:
		return "ActiveTimeout"
	case FlowEndReasonEnd:
		return "EndOfFlow"
	case FlowEndReasonForcedEnd:
		return "ForcedEndOfFlow"
	case FlowEndReasonLackOfResources:
		return "LackOfResources"
	default:
		return "UnknownEndReason"
	}
}

type FlowKey interface {
	SrcIP() []byte
	DstIP() []byte
	Proto() byte
	SrcPort() []byte
	DstPort() []byte
	Hash() uint64
}

type Flow interface {
	Event(Event, Time)
	Expire(Time)
	AddTimer(TimerID, TimerCallback, Time)
	HasTimer(TimerID) bool
	EOF(Time)
	NextEvent() Time
	Active() bool
	Key() FlowKey
}

type funcEntry struct {
	function TimerCallback
	when     Time
	id       TimerID
}

type funcEntries []*funcEntry

func (s funcEntries) Len() int           { return len(s) }
func (s funcEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s funcEntries) Less(i, j int) bool { return s[i].when < s[j].when }

type BaseFlow struct {
	key        FlowKey
	table      *FlowTable
	timers     map[TimerID]*funcEntry
	expireNext Time
	features   FeatureList
	active     bool
}

func (flow *BaseFlow) Stop() {
	flow.table.Remove(flow)
	flow.active = false
}

func (flow *BaseFlow) NextEvent() Time { return flow.expireNext }
func (flow *BaseFlow) Active() bool    { return flow.active }
func (flow *BaseFlow) Key() FlowKey    { return flow.key }

var timerPool = sync.Pool{
	New: func() interface{} {
		return new(funcEntry)
	},
}

func (flow *BaseFlow) Expire(when Time) {
	values := make(funcEntries, 0, len(flow.timers))
	for _, v := range flow.timers {
		values = append(values, v)
	}
	sort.Sort(values)
	for _, v := range values {
		if v.when <= when {
			v.function(v.when)
			delete(flow.timers, v.id)
			timerPool.Put(v)
		} else {
			flow.expireNext = v.when
			break
		}
	}
}

func (flow *BaseFlow) AddTimer(id TimerID, f TimerCallback, when Time) {
	if entry, existing := flow.timers[id]; existing {
		entry.function = f
		entry.when = when
	} else {
		t := timerPool.Get().(*funcEntry)
		t.function = f
		t.when = when
		t.id = id
		flow.timers[id] = t
	}
	if when < flow.expireNext || flow.expireNext == 0 {
		flow.expireNext = when
	}
}

func (flow *BaseFlow) HasTimer(id TimerID) bool {
	_, ret := flow.timers[id]
	return ret
}

func (flow *BaseFlow) Export(reason FlowEndReason, when Time) {
	flow.features.Stop()
	flow.features.Export(reason, when)
	flow.Stop()
}

func (flow *BaseFlow) idleEvent(now Time)   { flow.Export(FlowEndReasonIdle, now) }
func (flow *BaseFlow) activeEvent(now Time) { flow.Export(FlowEndReasonActive, now) }
func (flow *BaseFlow) EOF(now Time)         { flow.Export(FlowEndReasonForcedEnd, now) }

func (flow *BaseFlow) Event(event Event, when Time) {
	flow.AddTimer(timerIdle, flow.idleEvent, when+flow.table.idleTimeout)
	if !flow.HasTimer(timerActive) {
		flow.AddTimer(timerActive, flow.activeEvent, when+flow.table.activeTimeout)
	}
	flow.features.Event(event, when)
}

func NewBaseFlow(table *FlowTable, key FlowKey) BaseFlow {
	ret := BaseFlow{
		key:    key,
		table:  table,
		timers: make(map[TimerID]*funcEntry, 2),
		active: true}
	ret.features = table.features(&ret)
	ret.features.Start()
	return ret
}
