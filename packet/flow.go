package packet

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

func fnvHash(s []byte) (h uint64) {
	h = fnvBasis
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime
	}
	return
}

const fnvBasis = 14695981039346656037
const fnvPrime = 1099511628211

// src 4 dst 4 proto 1 src 2 dst 2
type fiveTuple4 [13]byte

func (t fiveTuple4) SrcIP() []byte   { return t[0:4] }
func (t fiveTuple4) DstIP() []byte   { return t[4:8] }
func (t fiveTuple4) Proto() []byte   { return t[8:9] }
func (t fiveTuple4) SrcPort() []byte { return t[9:11] }
func (t fiveTuple4) DstPort() []byte { return t[11:13] }
func (t fiveTuple4) Hash() uint64    { return fnvHash(t[:]) }

// src 16 dst 16 proto 1 src 2 dst 2
type fiveTuple6 [37]byte

func (t fiveTuple6) SrcIP() []byte   { return t[0:16] }
func (t fiveTuple6) DstIP() []byte   { return t[16:32] }
func (t fiveTuple6) Proto() []byte   { return t[32:33] }
func (t fiveTuple6) SrcPort() []byte { return t[33:35] }
func (t fiveTuple6) DstPort() []byte { return t[35:37] }
func (t fiveTuple6) Hash() uint64    { return fnvHash(t[:]) }

var emptyPort = make([]byte, 2)

func fivetuple(packet gopacket.Packet) (flows.FlowKey, bool) {
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
		ret := fiveTuple4{}
		copy(ret[0:4], srcIPR)
		copy(ret[4:8], dstIPR)
		ret[8] = protoB
		copy(ret[9:11], srcPortR)
		copy(ret[11:13], dstPortR)
		return ret, forward
	}
	if len(srcIPR) == 16 {
		ret := fiveTuple6{}
		copy(ret[0:16], srcIPR)
		copy(ret[16:32], dstIPR)
		ret[32] = protoB
		copy(ret[33:35], srcPortR)
		copy(ret[35:37], dstPortR)
		return ret, forward
	}
	return nil, false
}

type TCPFlow struct {
	flows.BaseFlow
	srcFIN, dstFIN, dstACK, srcACK bool
}

type UniFlow struct {
	flows.BaseFlow
}

func NewFlow(event flows.Event, table *flows.FlowTable, key flows.FlowKey) flows.Flow {
	tp := event.(*PacketBuffer).packet.TransportLayer()
	if tp != nil && tp.LayerType() == layers.LayerTypeTCP {
		return &TCPFlow{BaseFlow: flows.NewBaseFlow(table, key)}
	}
	return &UniFlow{flows.NewBaseFlow(table, key)}
}

func (flow *TCPFlow) Event(event flows.Event, when flows.Time) {
	flow.BaseFlow.Event(event, when)
	buffer := event.(*PacketBuffer)
	tcp := buffer.packet.TransportLayer().(*layers.TCP)
	if tcp.RST {
		flow.Export(flows.FlowEndReasonEnd, when)
	}
	if buffer.Forward {
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
		flow.Export(flows.FlowEndReasonEnd, when)
	}
}
