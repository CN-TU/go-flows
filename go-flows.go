package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"

	"bytes"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var fname = flag.String("format", "text", "Output format")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")

const ACTIVE_TIMEOUT int64 = 1800e9
const IDLE_TIMEOUT int64 = 300e9
const MAXLEN int = 9000

type FlowKey interface {
	SrcIP() []byte
	DstIP() []byte
	Proto() []byte
	SrcPort() []byte
	DstPort() []byte
}

// src 4 dst 4 proto 1 src 2 dst 2
type FiveTuple4 [13]byte

func (t FiveTuple4) SrcIP() []byte   { return t[0:4] }
func (t FiveTuple4) DstIP() []byte   { return t[4:8] }
func (t FiveTuple4) Proto() []byte   { return t[8:9] }
func (t FiveTuple4) SrcPort() []byte { return t[9:11] }
func (t FiveTuple4) DstPort() []byte { return t[11:13] }

// src 16 dst 16 proto 1 src 2 dst 2
type FiveTuple6 [37]byte

func (t FiveTuple6) SrcIP() []byte   { return t[0:16] }
func (t FiveTuple6) DstIP() []byte   { return t[16:32] }
func (t FiveTuple6) Proto() []byte   { return t[32:32] }
func (t FiveTuple6) SrcPort() []byte { return t[33:35] }
func (t FiveTuple6) DstPort() []byte { return t[35:37] }

func fivetuple(packet gopacket.Packet) FlowKey {
	network := packet.NetworkLayer()
	if network == nil {
		return nil
	}
	transport := packet.TransportLayer()
	var srcPort, dstPort []byte
	var icmpCode []byte
	var proto gopacket.LayerType
	if transport == nil {
		if icmp := packet.Layer(layers.LayerTypeICMPv4); icmp != nil {
			icmpCode = icmp.(*layers.ICMPv4).Contents[0:2]
			proto = layers.LayerTypeICMPv4
		} else if icmp := packet.Layer(layers.LayerTypeICMPv6); icmp != nil {
			icmpCode = icmp.(*layers.ICMPv6).Contents[0:2]
			proto = layers.LayerTypeICMPv6
		} else {
			return nil
		}
	} else {
		proto = transport.LayerType()
		srcPort = transport.TransportFlow().Src().Raw()
		dstPort = transport.TransportFlow().Dst().Raw()
	}
	srcip := network.NetworkFlow().Src().Raw()
	dstip := network.NetworkFlow().Dst().Raw()
	if bytes.Compare(srcip, dstip) > 0 {
		srcip, dstip = dstip, srcip
		srcPort, dstPort = dstPort, srcPort
	}
	if len(srcip) == 4 {
		ret := FiveTuple4{}
		copy(ret[0:4], srcip)
		copy(ret[4:8], dstip)
		if proto == layers.LayerTypeICMPv4 {
			copy(ret[11:13], icmpCode)
		} else {
			copy(ret[9:11], srcPort)
			copy(ret[11:13], dstPort)
		}
		ret[8] = byte(proto & 0xFF)
		return ret
	}
	if len(srcip) == 16 {
		ret := FiveTuple6{}
		copy(ret[0:4], srcip)
		copy(ret[4:8], dstip)
		if proto == layers.LayerTypeICMPv6 {
			copy(ret[11:13], icmpCode)
		} else {
			copy(ret[9:11], srcPort)
			copy(ret[11:13], dstPort)
		}
		ret[8] = byte(proto & 0xFF)
		return ret
	}
	return nil
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
	Expire(int64)
	AddTimer(string, func(int64), int64)
	HasTimer(string) bool
	EOF()
	NextEvent() int64
	Active() bool
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
	active     bool
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
	flow.active = false
	flow.Table.Remove(flow.Key, flow)
}

func (flow *BaseFlow) NextEvent() int64 { return flow.ExpireNext }
func (flow *BaseFlow) Active() bool     { return flow.active }

type FuncEntries []*FuncEntry

func (s FuncEntries) Len() int           { return len(s) }
func (s FuncEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s FuncEntries) Less(i, j int) bool { return s[i].When < s[j].When }

func (flow *BaseFlow) Expire(when int64) {
	var values FuncEntries
	for _, v := range flow.Timers {
		values = append(values, v)
	}
	sort.Sort(values)
	for _, v := range values {
		if v.When <= when {
			v.Function(v.When)
			delete(flow.Timers, v.Name)
		} else {
			flow.ExpireNext = v.When
			break
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
	if tcp.RST {
		flow.Export("RST")
	}
	if bytes.Compare(packet.NetworkLayer().NetworkFlow().Src().Raw(), flow.Key.SrcIP()) == 0 {
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

type FlowTable struct {
	flows     map[FlowKey]Flow
	eof       bool
	lastEvent int64
}

func NewFlowTable() *FlowTable {
	return &FlowTable{flows: make(map[FlowKey]Flow, 1000000)}
}

func NewBaseFlow(table *FlowTable, key FlowKey) BaseFlow {
	features := []Feature{
		&SrcIP{},
		&DstIP{},
	}
	return BaseFlow{Key: key, Table: table, Timers: make(map[string]*FuncEntry, 2), Features: features, active: true}
}

func NewFlow(packet gopacket.Packet, table *FlowTable, key FlowKey) Flow {
	if packet.Layer(layers.LayerTypeTCP) != nil {
		return &TCPFlow{BaseFlow: NewBaseFlow(table, key)}
	}
	return &UniFlow{NewBaseFlow(table, key)}
}

func (tab *FlowTable) Event(packet gopacket.Packet, key FlowKey) {
	when := packet.Metadata().Timestamp.UnixNano()

	if tab.lastEvent < when {
		for _, elem := range tab.flows {
			if when > elem.NextEvent() {
				elem.Expire(when)
			}
		}
		tab.lastEvent = when + 300e9
	}
	// event every n seconds
	elem, ok := tab.flows[key]
	if ok {
		if when > elem.NextEvent() {
			elem.Expire(when)
			ok = elem.Active()
		}
	}
	if !ok {
		elem = NewFlow(packet, tab, key)
		tab.flows[key] = elem
	}
	elem.Event(packet, when)
}

func (tab *FlowTable) Remove(key FlowKey, entry *BaseFlow) {
	if !tab.eof {
		delete(tab.flows, key)
	}
}

func (tab *FlowTable) EOF() {
	tab.eof = true
	for _, v := range tab.flows {
		v.EOF()
	}
	tab.flows = make(map[FlowKey]Flow)
	tab.eof = false
}

type PacketBuffer struct {
	buffer [MAXLEN]byte
	packet gopacket.Packet
	key    FlowKey
}

func readFiles(fnames []string) (<-chan *PacketBuffer, chan<- *PacketBuffer) {
	result := make(chan *PacketBuffer, 1000)
	empty := make(chan *PacketBuffer, 1000)

	go func() {
		defer close(result)
		for i := 0; i < 1000; i++ {
			empty <- &PacketBuffer{}
		}
		options := gopacket.DecodeOptions{Lazy: true, NoCopy: true}
		for _, fname := range fnames {
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
				buffer := <-empty
				copy(buffer.buffer[:], data)
				plen := len(data)
				if plen > MAXLEN {
					plen = MAXLEN
				}
				buffer.packet = gopacket.NewPacket(buffer.buffer[:plen], decoder, options)
				m := buffer.packet.Metadata()
				m.CaptureInfo = ci
				m.Truncated = m.Truncated || ci.CaptureLength < ci.Length || plen < len(data)
				result <- buffer
			}
			fhandle.Close()
		}
	}()

	return result, empty
}

func parsePacket(in <-chan *PacketBuffer) <-chan *PacketBuffer {
	out := make(chan *PacketBuffer, 1000)

	go func() {
		for packet := range in {
			packet.packet.TransportLayer()
			out <- packet
		}
		close(out)
	}()

	return out
}

func parseKey(in <-chan *PacketBuffer) <-chan *PacketBuffer {
	out := make(chan *PacketBuffer, 1000)

	go func() {
		for packet := range in {
			packet.key = fivetuple(packet.packet)
			out <- packet
		}
		close(out)
	}()

	return out
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

	packets, empty := readFiles(flag.Args())
	parsed := parsePacket(packets)
	keyed := parseKey(parsed)

	flowtable := NewFlowTable()
	for buffer := range keyed {
		if buffer.key != nil {
			flowtable.Event(buffer.packet, buffer.key)
		}
		empty <- buffer
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
