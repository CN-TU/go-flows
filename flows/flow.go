package flows

import (
	"sort"
)

type FlowKey interface {
	SrcIP() []byte
	DstIP() []byte
	Proto() []byte
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
}

type BaseFlow struct {
	key        FlowKey
	table      *FlowTable
	timers     map[TimerID]*funcEntry
	expireNext Time
	active     bool
	features   FeatureList
}

func (flow *BaseFlow) Stop() {
	flow.active = false
	flow.table.Remove(flow.key, flow)
}

func (flow *BaseFlow) NextEvent() Time { return flow.expireNext }
func (flow *BaseFlow) Active() bool    { return flow.active }

func (flow *BaseFlow) Expire(when Time) {
	var values funcEntries
	for _, v := range flow.timers {
		values = append(values, v)
	}
	sort.Sort(values)
	for _, v := range values {
		if v.when <= when {
			v.function(v.when)
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
		flow.timers[id] = &funcEntry{f, when, id}
	}
	if when < flow.expireNext || flow.expireNext == 0 {
		flow.expireNext = when
	}
}

func (flow *BaseFlow) HasTimer(id TimerID) bool {
	_, ret := flow.timers[id]
	return ret
}

func (flow *BaseFlow) Export(reason string, when Time) {
	flow.features.Stop()
	flow.features.Export(reason, when)
	flow.Stop()
}

func (flow *BaseFlow) Idle(now Time) {
	flow.Export("IDLE", now)
}

func (flow *BaseFlow) EOF(now Time) {
	flow.Export("EOF", now)
}

const ACTIVE_TIMEOUT = 1800 * Seconds //FIXME
const IDLE_TIMEOUT = 300 * Seconds    //FIXME

func (flow *BaseFlow) Event(event Event, when Time) {
	flow.AddTimer(TimerIdle, flow.Idle, when+IDLE_TIMEOUT)
	if !flow.HasTimer(TimerActive) {
		flow.AddTimer(TimerActive, flow.Idle, when+ACTIVE_TIMEOUT)
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
