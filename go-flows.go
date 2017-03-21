package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var format = flag.String("format", "text", "Output format")
var output = flag.String("output", "-", "Output filename")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")
var blockprofile = flag.String("blockprofile", "", "write block profile to file")

const ACTIVE_TIMEOUT int64 = 1800e9
const IDLE_TIMEOUT int64 = 300e9
const MAXLEN int = 9000

type TimerID int

const (
	TimerIdle TimerID = iota
	TimerActive
)

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
func (t FiveTuple6) Proto() []byte   { return t[32:33] }
func (t FiveTuple6) SrcPort() []byte { return t[33:35] }
func (t FiveTuple6) DstPort() []byte { return t[35:37] }

var emptyPort = make([]byte, 2)

func fivetuple(packet gopacket.Packet) (FlowKey, bool) {
	network := packet.NetworkLayer()
	if network == nil {
		return nil, false
	}
	transport := packet.TransportLayer()
	var srcPortR, dstPortR []byte
	var proto gopacket.LayerType
	isicmp := false
	if transport == nil {
		if icmp := packet.LayerClass(layers.LayerClassIPControl); icmp != nil {
			srcPortR = emptyPort
			dstPortR = icmp.LayerContents()[0:2]
			proto = icmp.LayerType()
			isicmp = true
		} else {
			return nil, false
		}
	} else {
		srcPort, dstPort := transport.TransportFlow().Endpoints()
		srcPortR = srcPort.Raw()
		dstPortR = dstPort.Raw()
		proto = transport.LayerType()
	}
	srcIP, dstIP := network.NetworkFlow().Endpoints()
	forward := true
	if dstIP.LessThan(srcIP) {
		forward = false
		srcIP, dstIP = dstIP, srcIP
		if !isicmp {
			srcPortR, dstPortR = dstPortR, srcPortR
		}
	}
	var protoB byte
	switch proto {
	case layers.LayerTypeTCP:
		protoB = byte(layers.IPProtocolTCP)
	case layers.LayerTypeUDP:
		protoB = byte(layers.IPProtocolUDP)
	case layers.LayerTypeICMPv4:
		protoB = byte(layers.IPProtocolICMPv4)
	case layers.LayerTypeICMPv6:
		protoB = byte(layers.IPProtocolICMPv6)
	}
	srcIPR := srcIP.Raw()
	dstIPR := dstIP.Raw()

	if len(srcIPR) == 4 {
		ret := FiveTuple4{}
		copy(ret[0:4], srcIPR)
		copy(ret[4:8], dstIPR)
		ret[8] = protoB
		copy(ret[9:11], srcPortR)
		copy(ret[11:13], dstPortR)
		return ret, forward
	}
	if len(srcIPR) == 16 {
		ret := FiveTuple6{}
		copy(ret[0:16], srcIPR)
		copy(ret[16:32], dstIPR)
		ret[32] = protoB
		copy(ret[33:35], srcPortR)
		copy(ret[35:37], dstPortR)
		return ret, forward
	}
	return nil, false
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
	flow      *BaseFlow
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
		f.SetValue(net.IP(f.flow.Key.SrcIP()), when)
	}
}

type DstIP struct {
	BaseFeature
}

func (f *DstIP) Event(new interface{}, when int64) {
	if f.value == nil {
		f.SetValue(net.IP(f.flow.Key.DstIP()), when)
	}
}

type Proto struct {
	BaseFeature
}

func (f *Proto) Event(new interface{}, when int64) {
	if f.value == nil {
		f.SetValue(f.flow.Key.Proto(), when)
	}
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

type FuncEntry struct {
	Function func(int64)
	When     int64
	ID       TimerID
}

type BaseFlow struct {
	Key        FlowKey
	Table      *FlowTable
	Timers     map[TimerID]*FuncEntry
	ExpireNext int64
	active     bool
	Features   FeatureList
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

func (flow *BaseFlow) Event(packet FlowPacket, when int64) {
	flow.AddTimer(TimerIdle, flow.Idle, when+IDLE_TIMEOUT)
	if !flow.HasTimer(TimerActive) {
		flow.AddTimer(TimerActive, flow.Idle, when+ACTIVE_TIMEOUT)
	}
	flow.Features.Event(packet, when)
}

func (flow *TCPFlow) Event(packet FlowPacket, when int64) {
	flow.BaseFlow.Event(packet, when)
	tcp := packet.Layer(layers.LayerTypeTCP).(*layers.TCP)
	if tcp.RST {
		flow.Export("RST", when)
	}
	if packet.Forward {
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
		flow.Export("FIN", when)
	}
}

type FlowTable struct {
	flows     map[FlowKey]Flow
	features  func(*BaseFlow) FeatureList
	eof       bool
	lastEvent int64
}

func NewFlowTable(features func(*BaseFlow) FeatureList) *FlowTable {
	return &FlowTable{flows: make(map[FlowKey]Flow, 1000000), features: features}
}

func NewBaseFlow(table *FlowTable, key FlowKey) BaseFlow {
	ret := BaseFlow{Key: key, Table: table, Timers: make(map[TimerID]*FuncEntry, 2), active: true}
	ret.Features = table.features(&ret)
	ret.Features.Start()
	return ret
}

func NewFlow(packet gopacket.Packet, table *FlowTable, key FlowKey) Flow {
	if packet.Layer(layers.LayerTypeTCP) != nil {
		return &TCPFlow{BaseFlow: NewBaseFlow(table, key)}
	}
	return &UniFlow{NewBaseFlow(table, key)}
}

func (tab *FlowTable) Event(packet FlowPacket, key FlowKey) {
	when := packet.Metadata().Timestamp.UnixNano()

	if tab.lastEvent < when {
		for _, elem := range tab.flows {
			if when > elem.NextEvent() {
				elem.Expire(when)
			}
		}
		tab.lastEvent = when + 100e9
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

type FlowPacket struct {
	gopacket.Packet
	Forward bool
}

type PacketBuffer struct {
	buffer [MAXLEN]byte
	packet FlowPacket
	key    FlowKey
}

type Exporter interface {
	Export([]Feature, string, int64)
	Finish()
}

type PrintExporter struct {
	exportlist chan []interface{}
	finished   chan struct{}
	stop       chan struct{}
}

func (pe *PrintExporter) Export(features []Feature, reason string, when int64) {
	n := len(features)
	var list = make([]interface{}, n+2)
	for i, elem := range features {
		list[i] = elem.Value()
	}
	list[n] = reason
	list[n+1] = when
	pe.exportlist <- list
}

func (pe *PrintExporter) Finish() {
	close(pe.stop)
	<-pe.finished
}

func NewPrintExporter(filename string) Exporter {
	ret := &PrintExporter{make(chan []interface{}, 1000), make(chan struct{}), make(chan struct{})}
	var outfile io.Writer
	if filename == "-" {
		outfile = os.Stdout
	} else {
		var err error
		outfile, err = os.Create(filename)
		if err != nil {
			log.Fatal("Couldn't open file ", filename, err)
		}
	}
	go func() {
		defer close(ret.finished)
		for {
			select {
			case data := <-ret.exportlist:
				fmt.Fprintln(outfile, data...)
			case <-ret.stop:
				return
			}
		}
	}()
	return ret
}

type FeatureList struct {
	event    []Feature
	export   []Feature
	startup  []Feature
	exporter Exporter
}

func (list *FeatureList) Start() {
	for _, feature := range list.startup {
		feature.Start()
	}
}

func (list *FeatureList) Stop() {
	for _, feature := range list.startup {
		feature.Stop()
	}
}

func (list *FeatureList) Event(data interface{}, when int64) {
	for _, feature := range list.event {
		feature.Event(data, when)
	}
}

func (list *FeatureList) Export(why string, when int64) {
	list.exporter.Export(list.export, why, when)
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
				buffer.packet.Packet = gopacket.NewPacket(buffer.buffer[:plen], decoder, options)
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
			packet.key, packet.packet.Forward = fivetuple(packet.packet)
			out <- packet
		}
		close(out)
	}()

	return out
}

func main() {
	flag.Parse()
	var exporter Exporter
	if *format == "text" {
		exporter = NewPrintExporter(*output)
	} else {
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
	if *blockprofile != "" {
		f, err := os.Create(*blockprofile)
		if err != nil {
			log.Fatal(err)
		}
		runtime.SetBlockProfileRate(1)
		defer pprof.Lookup("block").WriteTo(f, 0)
	}

	packets, empty := readFiles(flag.Args())
	parsed := parsePacket(packets)

	defer exporter.Finish()
	flowtable := NewFlowTable(func(flow *BaseFlow) FeatureList {
		a := &SrcIP{}
		a.flow = flow
		b := &DstIP{}
		b.flow = flow
		//		c := &Proto{}
		//		c.flow = flow
		features := []Feature{
			//			c,
			a,
			b}
		ret := FeatureList{
			event:    features,
			export:   features,
			startup:  features,
			exporter: exporter}

		return ret
	})
	defer flowtable.EOF()

	for buffer := range parsed {
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
}
