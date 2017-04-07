package flows

import (
	"sort"
)

type FlowEndReason byte

const (
	FlowEndReasonIdle            FlowEndReason = 1
	FlowEndReasonActive          FlowEndReason = 2
	FlowEndReasonEnd             FlowEndReason = 3
	FlowEndReasonForcedEnd       FlowEndReason = 4
	FlowEndReasonLackOfResources FlowEndReason = 5
)

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
	Init(*FlowTable, FlowKey, Time)
	Recycle()
	Table() *FlowTable
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
	features   *FeatureList
	active     bool
}

func (flow *BaseFlow) Stop() {
	flow.table.Remove(flow)
	flow.active = false
}

func (flow *BaseFlow) NextEvent() Time   { return flow.expireNext }
func (flow *BaseFlow) Active() bool      { return flow.active }
func (flow *BaseFlow) Key() FlowKey      { return flow.key }
func (flow *BaseFlow) Recycle()          {}
func (flow *BaseFlow) Table() *FlowTable { return flow.table }

func (flow *BaseFlow) Expire(when Time) {
	values := make(funcEntries, 0, len(flow.timers))
	for _, v := range flow.timers {
		values = append(values, v)
	}
	sort.Sort(values)
	for _, v := range values {
		if v.when <= when {
			v.function(v.when, when)
			delete(flow.timers, v.id)
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
		t := new(funcEntry)
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

func (flow *BaseFlow) Export(reason FlowEndReason, when Time, now Time) {
	flow.features.Stop(reason, when)
	flow.features.Export(now)
	flow.Stop()
}

func (flow *BaseFlow) idleEvent(expired Time, now Time) { flow.Export(FlowEndReasonIdle, expired, now) }
func (flow *BaseFlow) activeEvent(expired Time, now Time) {
	flow.Export(FlowEndReasonActive, expired, now)
}
func (flow *BaseFlow) EOF(now Time) { flow.Export(FlowEndReasonForcedEnd, now, now) }

func (flow *BaseFlow) Event(event Event, when Time) {
	flow.AddTimer(timerIdle, flow.idleEvent, when+flow.table.idleTimeout)
	if !flow.HasTimer(timerActive) {
		flow.AddTimer(timerActive, flow.activeEvent, when+flow.table.activeTimeout)
	}
	flow.features.Event(event, when)
}

func (flow *BaseFlow) Init(table *FlowTable, key FlowKey, time Time) {
	flow.key = key
	flow.table = table
	flow.timers = make(map[TimerID]*funcEntry, 2)
	flow.active = true
	flow.features = table.features()
	flow.features.Init(flow)
	flow.features.Start(time)
}

/*
func NewBaseFlow(table *FlowTable, key FlowKey, time Time) BaseFlow {
	ret := BaseFlow{
		key:    key,
		table:  table,
		timers: make(map[TimerID]*funcEntry, 2),
		active: true}
	ret.features = table.features(&ret)
	ret.features.Start(time)
	return ret
}*/
