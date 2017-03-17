package main

import (
	"container/heap"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var fname = flag.String("format", "text", "Output format")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")

const ACTIVE_TIMEOUT int64 = 1800e9
const IDLE_TIMEOUT int64 = 300e9

type FlowKey interface {
}

type FiveTuple struct {
	SrcIP, DstIP     gopacket.Endpoint
	proto            gopacket.LayerType
	SrcPort, DstPort gopacket.Endpoint
	ICMPTypeCode     uint16
}

func fivetuple(packet gopacket.Packet) FlowKey {
	network := packet.NetworkLayer()
	if network == nil {
		return nil
	}
	transport := packet.TransportLayer()
	var srcPort, dstPort gopacket.Endpoint
	var icmpCode uint16
	var proto gopacket.LayerType
	if transport == nil {
		if icmp := packet.Layer(layers.LayerTypeICMPv4); icmp != nil {
			icmpCode = uint16(icmp.(*layers.ICMPv4).TypeCode)
			proto = layers.LayerTypeICMPv4
		} else if icmp := packet.Layer(layers.LayerTypeICMPv6); icmp != nil {
			icmpCode = uint16(icmp.(*layers.ICMPv6).TypeCode)
			proto = layers.LayerTypeICMPv6
		} else {
			return nil
		}
	} else {
		srcPort = transport.TransportFlow().Src()
		dstPort = transport.TransportFlow().Dst()
	}
	srcip, dstip := network.NetworkFlow().Endpoints()
	if dstip.LessThan(srcip) {
		srcip, dstip = dstip, srcip
		srcPort, dstPort = dstPort, srcPort
	}
	ret := FiveTuple{}
	//copy(ret.SrcIP[:], srcip.Raw())
	//copy(ret.DstIP[:], dstip.Raw())
	ret.SrcIP = srcip
	ret.DstIP = dstip
	if proto == layers.LayerTypeICMPv4 || proto == layers.LayerTypeICMPv6 {
		ret.ICMPTypeCode = icmpCode
		ret.proto = proto
	} else {
		ret.SrcPort = srcPort
		ret.DstPort = dstPort
		ret.proto = transport.LayerType()
	}

	return ret
}

type Feature interface {
	Event(interface{}, int64)
	Value() interface{}
	SetValue(interface{}, int64)
	Start()
	Stop()
}

type BaseFeature struct {
	value     interface{}
	dependent []Feature
}

func (f *BaseFeature) Event(interface{}, int64) {

}

func (f *BaseFeature) Value() interface{} {
	return f.value
}

func (f *BaseFeature) SetValue(new interface{}, when int64) {
	f.value = new
	if new != nil {
		for _, v := range f.dependent {
			v.Event(new, when)
		}
	}
}

func (f *BaseFeature) Start() {

}

func (f *BaseFeature) Stop() {

}

type SrcIP struct {
	BaseFeature
}

func (f *SrcIP) Event(new interface{}, when int64) {
	if f.value == nil {
		f.SetValue(new.(gopacket.Packet).NetworkLayer().NetworkFlow().Src(), when)
	}
}

type DstIP struct {
	BaseFeature
}

func (f *DstIP) Event(new interface{}, when int64) {
	if f.value == nil {
		f.SetValue(new.(gopacket.Packet).NetworkLayer().NetworkFlow().Dst(), when)
	}
}

type Flow interface {
	Event(gopacket.Packet, int64)
	Expire(*TimerItem)
	AddTimer(string, func(int64), int64)
	HasTimer(string) bool
	EOF()
}

type FuncEntry struct {
	Function func(int64)
	When     int64
	Name     string
}

type BaseFlow struct {
	Key        FlowKey
	Table      *FlowTable
	Timers     map[string]*FuncEntry
	ExpireNext int64
	Expires    map[*TimerItem]struct{}
	Features   []Feature
}

type TCPFlow struct {
	BaseFlow
	srcFIN, dstFIN, dstACK, srcACK bool
}

type UniFlow struct {
	BaseFlow
}

func (flow *BaseFlow) Stop() {
	flow.Table.Remove(flow.Key, flow)
}

type FuncEntries []*FuncEntry

func (s FuncEntries) Len() int           { return len(s) }
func (s FuncEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s FuncEntries) Less(i, j int) bool { return s[i].When < s[j].When }

func (flow *BaseFlow) Expire(which *TimerItem) {
	delete(flow.Expires, which)
	var values FuncEntries
	for _, v := range flow.Timers {
		values = append(values, v)
	}
	sort.Sort(values)
	for _, v := range values {
		if v.When <= which.when {
			v.Function(v.When)
			delete(flow.Timers, v.Name)
		}
	}
}

func (flow *BaseFlow) AddTimer(name string, f func(int64), when int64) {
	if entry, existing := flow.Timers[name]; existing {
		entry.Function = f
		entry.When = when
	} else {
		flow.Timers[name] = &FuncEntry{f, when, name}
	}
	if when < flow.ExpireNext || flow.ExpireNext == 0 {
		flow.Expires[flow.Table.AddTimer(flow, when)] = struct{}{}
		flow.ExpireNext = when
	}
}

func (flow *BaseFlow) HasTimer(name string) bool {
	_, ret := flow.Timers[name]
	return ret
}

func (flow *BaseFlow) Export(reason string) {
	for _, feature := range flow.Features {
		fmt.Print(feature.Value(), ", ")
	}
	fmt.Println(reason)
	flow.Stop()
}

func (flow *BaseFlow) Idle(now int64) {
	for k := range flow.Expires {
		k.Flow = nil
	}
	flow.Export("IDLE")
}

func (flow *BaseFlow) EOF() {
	flow.Export("EOF")
}

func (flow *BaseFlow) Event(packet gopacket.Packet, when int64) {
	flow.AddTimer("IDLE", flow.Idle, when+IDLE_TIMEOUT)
	if !flow.HasTimer("ACTIVE") {
		flow.AddTimer("ACTIVE", flow.Idle, when+ACTIVE_TIMEOUT)
	}
	for _, feature := range flow.Features {
		feature.Event(packet, when)
	}
}

func (flow *TCPFlow) Event(packet gopacket.Packet, when int64) {
	flow.BaseFlow.Event(packet, when)
	tcp := packet.Layer(layers.LayerTypeTCP).(*layers.TCP)
	srcip := packet.NetworkLayer().NetworkFlow().Src()
	if tcp.RST {
		flow.Export("RST")
	}
	if srcip == flow.Key.(FiveTuple).SrcIP {
		if tcp.FIN {
			flow.srcFIN = true
		}
		if flow.dstFIN && tcp.ACK {
			flow.dstACK = true
		}
	} else {
		if tcp.FIN {
			flow.dstFIN = true
		}
		if flow.srcFIN && tcp.ACK {
			flow.srcACK = true
		}
	}

	if flow.srcFIN && flow.srcACK && flow.dstFIN && flow.dstACK {
		flow.Export("FIN")
	}
}

type TimerItem struct {
	Flow  Flow
	when  int64
	index int
}

func (it *TimerItem) Reset() {
	it.Flow = nil
	it.index = 0
}

type TimerQueue []*TimerItem

func (tq TimerQueue) Len() int { return len(tq) }

func (tq TimerQueue) Less(i, j int) bool {
	return tq[i].when < tq[j].when
}

func (tq TimerQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].index = i
	tq[j].index = j
}

func (tq *TimerQueue) Push(x interface{}) {
	n := len(*tq)
	item := x.(*TimerItem)
	item.index = n
	*tq = append(*tq, item)
}

func (tq *TimerQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*tq = old[0 : n-1]
	return item
}

func (tq *TimerQueue) update(item *TimerItem, when int64) {
	item.when = when
	heap.Fix(tq, item.index)
}

type FlowTable struct {
	flows      map[FlowKey]Flow
	key        func(gopacket.Packet) FlowKey
	timers     TimerQueue
	freeTimers chan *TimerItem
}

func NewFlowTable(f func(gopacket.Packet) FlowKey) *FlowTable {
	return &FlowTable{flows: make(map[FlowKey]Flow), key: f, freeTimers: make(chan *TimerItem, 10000)}
}

func NewBaseFlow(table *FlowTable, key FlowKey) BaseFlow {
	features := []Feature{
		&SrcIP{},
		&DstIP{},
	}
	return BaseFlow{Key: key, Table: table, Expires: make(map[*TimerItem]struct{}, 1), Timers: make(map[string]*FuncEntry, 2), Features: features}
}

func NewFlow(packet gopacket.Packet, table *FlowTable, key FlowKey) Flow {
	if packet.Layer(layers.LayerTypeTCP) != nil {
		return &TCPFlow{BaseFlow: NewBaseFlow(table, key)}
	}
	return &UniFlow{NewBaseFlow(table, key)}
}

func (tab *FlowTable) Expire(now int64) {
	for len(tab.timers) > 0 && tab.timers[0].when < now {
		timer := heap.Pop(&tab.timers).(*TimerItem)
		timer.Flow.Expire(timer)
		timer.Reset()
		select {
		case tab.freeTimers <- timer:
		default:
		}
	}
}

func (tab *FlowTable) AddTimer(flow Flow, when int64) *TimerItem {
	var ret *TimerItem
	select {
	case ret = <-tab.freeTimers:
		ret.Flow = flow
		ret.when = when
	default:
		ret = &TimerItem{Flow: flow, when: when}
	}
	heap.Push(&tab.timers, ret)
	return ret
}

func (tab *FlowTable) Event(packet gopacket.Packet) {
	when := packet.Metadata().Timestamp.UnixNano()
	tab.Expire(when)
	if key := tab.key(packet); key != nil {
		elem, ok := tab.flows[key]
		if !ok {
			elem = NewFlow(packet, tab, key)
			tab.flows[key] = elem
		}
		elem.Event(packet, when)
	}
}

func (tab *FlowTable) Remove(key FlowKey, entry *BaseFlow) {
	for timer := range entry.Expires {
		if timer.Flow != nil { //already recycled!
			heap.Remove(&tab.timers, timer.index)
			timer.Reset()
			select {
			case tab.freeTimers <- timer:
			default:
			}
		}
	}
	delete(tab.flows, key)
}

func (tab *FlowTable) EOF() {
	for _, v := range tab.flows {
		v.EOF()
	}
}

func main() {
	flag.Parse()
	if *fname != "text" {
		log.Fatal("Only text output supported for now!")
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	flowtable := NewFlowTable(fivetuple)

	options := gopacket.DecodeOptions{Lazy: true, NoCopy: true}

	for _, fname := range flag.Args() {
		fhandle, err := pcap.OpenOffline(fname)
		if err != nil {
			log.Fatalf("Couldn't open file %s", fname)
		}
		decoder := fhandle.LinkType()

		for {
			data, ci, err := fhandle.ZeroCopyReadPacketData()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Println("Error:", err)
				continue
			}
			packet := gopacket.NewPacket(data, decoder, options)
			m := packet.Metadata()
			m.CaptureInfo = ci
			m.Truncated = m.Truncated || ci.CaptureLength < ci.Length
			flowtable.Event(packet)
		}
		fhandle.Close()
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}

	flowtable.EOF()
}
