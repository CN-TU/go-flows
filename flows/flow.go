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
	AddTimer(TimerID, func(int64), int64)
	HasTimer(TimerID) bool
	EOF()
	NextEvent() int64
	Active() bool
}

type TimerID int

const (
	TimerIdle TimerID = iota
	TimerActive
)

type FuncEntry struct {
	Function func(int64)
	When     int64
	ID       TimerID
}

type FuncEntries []*FuncEntry

func (s FuncEntries) Len() int           { return len(s) }
func (s FuncEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s FuncEntries) Less(i, j int) bool { return s[i].When < s[j].When }

type BaseFlow struct {
	Key        FlowKey
	Table      *FlowTable
	Timers     map[TimerID]*FuncEntry
	ExpireNext int64
	active     bool
	Features   FeatureList
}

func (flow *BaseFlow) Stop() {
	flow.active = false
	flow.Table.Remove(flow.Key, flow)
}

func (flow *BaseFlow) NextEvent() int64 { return flow.ExpireNext }
func (flow *BaseFlow) Active() bool     { return flow.active }

func (flow *BaseFlow) Expire(when int64) {
	var values FuncEntries
	for _, v := range flow.Timers {
		values = append(values, v)
	}
	sort.Sort(values)
	for _, v := range values {
		if v.When <= when {
			v.Function(v.When)
			delete(flow.Timers, v.ID)
		} else {
			flow.ExpireNext = v.When
			break
		}
	}
}

func (flow *BaseFlow) AddTimer(id TimerID, f func(int64), when int64) {
	if entry, existing := flow.Timers[id]; existing {
		entry.Function = f
		entry.When = when
	} else {
		flow.Timers[id] = &FuncEntry{f, when, id}
	}
	if when < flow.ExpireNext || flow.ExpireNext == 0 {
		flow.ExpireNext = when
	}
}

func (flow *BaseFlow) HasTimer(id TimerID) bool {
	_, ret := flow.Timers[id]
	return ret
}

func (flow *BaseFlow) Export(reason string, when int64) {
	flow.Features.Stop()
	flow.Features.Export(reason, when)
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
	flow.Features.Event(packet, when)
}

func NewBaseFlow(table *FlowTable, key FlowKey) BaseFlow {
	ret := BaseFlow{Key: key, Table: table, Timers: make(map[TimerID]*FuncEntry, 2), active: true}
	ret.Features = table.features(&ret)
	ret.Features.Start()
	return ret
}
