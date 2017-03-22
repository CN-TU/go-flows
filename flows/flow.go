package flows

import (
	"sort"

	"github.com/google/gopacket"
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
	Event(FlowPacket, int64)
	Expire(int64)
	AddTimer(TimerID, TimerCallback, int64)
	HasTimer(TimerID) bool
	EOF()
	NextEvent() int64
	Active() bool
}

type BaseFlow struct {
	key        FlowKey
	table      *FlowTable
	timers     map[TimerID]*funcEntry
	expireNext int64
	active     bool
	features   FeatureList
}

func (flow *BaseFlow) Stop() {
	flow.active = false
	flow.table.Remove(flow.key, flow)
}

func (flow *BaseFlow) NextEvent() int64 { return flow.expireNext }
func (flow *BaseFlow) Active() bool     { return flow.active }

func (flow *BaseFlow) Expire(when int64) {
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

func (flow *BaseFlow) AddTimer(id TimerID, f TimerCallback, when int64) {
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

func (flow *BaseFlow) Export(reason string, when int64) {
	flow.features.Stop()
	flow.features.Export(reason, when)
	flow.Stop()
}

func (flow *BaseFlow) Idle(now int64) {
	flow.Export("IDLE", now)
}

func (flow *BaseFlow) EOF() {
	flow.Export("EOF", -1)
}

const ACTIVE_TIMEOUT int64 = 1800e9 //FIXME
const IDLE_TIMEOUT int64 = 300e9    //FIXME

type FlowPacket struct { //FIXME
	gopacket.Packet
	Forward bool
}

func (flow *BaseFlow) Event(packet FlowPacket, when int64) {
	flow.AddTimer(TimerIdle, flow.Idle, when+IDLE_TIMEOUT)
	if !flow.HasTimer(TimerActive) {
		flow.AddTimer(TimerActive, flow.Idle, when+ACTIVE_TIMEOUT)
	}
	flow.features.Event(packet, when)
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
