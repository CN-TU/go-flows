package packet

import (
	"bytes"
	"fmt"

	"github.com/google/gopacket/layers"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

// src 4 dst 4 proto 1 src 2 dst 2
type fiveTuple4 [13]byte

func (t fiveTuple4) SrcIP() []byte   { return t[0:4] }
func (t fiveTuple4) DstIP() []byte   { return t[4:8] }
func (t fiveTuple4) Proto() byte     { return t[8] }
func (t fiveTuple4) SrcPort() []byte { return t[9:11] }
func (t fiveTuple4) DstPort() []byte { return t[11:13] }
func (t fiveTuple4) Hash() uint64    { return fnvHash(t[:]) }

// src 16 dst 16 proto 1 src 2 dst 2
type fiveTuple6 [37]byte

func (t fiveTuple6) SrcIP() []byte   { return t[0:16] }
func (t fiveTuple6) DstIP() []byte   { return t[16:32] }
func (t fiveTuple6) Proto() byte     { return t[32] }
func (t fiveTuple6) SrcPort() []byte { return t[33:35] }
func (t fiveTuple6) DstPort() []byte { return t[35:37] }
func (t fiveTuple6) Hash() uint64    { return fnvHash(t[:]) }

var emptyPort = make([]byte, 2)

func fivetuple(packet PacketBuffer) (flows.FlowKey, bool) {
	network := packet.NetworkLayer()
	if network == nil {
		return nil, false
	}
	transport := packet.TransportLayer()
	if transport == nil {
		return nil, false
	}
	srcPort, dstPort := transport.TransportFlow().Endpoints()
	srcPortR := srcPort.Raw()
	dstPortR := dstPort.Raw()
	proto := transport.LayerType()
	srcIP, dstIP := network.NetworkFlow().Endpoints()
	forward := true
	if dstIP.LessThan(srcIP) {
		forward = false
		srcIP, dstIP = dstIP, srcIP
		if !layers.LayerClassIPControl.Contains(proto) {
			srcPortR, dstPortR = dstPortR, srcPortR
		}
	} else if bytes.Compare(srcIP.Raw(), dstIP.Raw()) == 0 {
		if srcPort.LessThan(dstPort) {
			forward = false
			srcIP, dstIP = dstIP, srcIP
			if !layers.LayerClassIPControl.Contains(proto) {
				srcPortR, dstPortR = dstPortR, srcPortR
			}
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

/* According to curated data files we have:
- destinationIPv4Address
- sourceIPv4Address
- protocolIdentifier
- sourceTransportPort
- destinationTransportPort
- ipClassOfService
- ingressPhysicalInterface
- octetTotalCount

- flowStartSeconds <- this does not make sense
*/

func MakeDynamicKeySelector(key []string, bidirectional bool) (ret DynamicKeySelector) {
	for _, key := range key {
		switch key {
		case "sourceIPv4Address", "sourceIPv6Address", "sourceIPAddress":
			ret.srcIP = true
			ret.network = true
		case "destinationIPv4Address", "destinationIPv6Address", "destinationIPAddress":
			ret.dstIP = true
			ret.network = true
		case "protocolIdentifier":
			ret.protocolIdentifier = true
			ret.network = true
		case "sourceTransportPort":
			ret.srcPort = true
			ret.transport = true
		case "destinationTransportPort":
			ret.dstPort = true
			ret.transport = true
		default:
			panic(fmt.Sprintf("Unknown key_feature '%s'", key))
		}
	}
	ret.bidirectional = bidirectional
	ret.fivetuple = ret.srcIP && ret.dstIP && ret.protocolIdentifier && ret.srcPort && ret.dstPort && bidirectional
	ret.empty = !ret.network && !ret.transport
	return
}

type dynamicKey [37]byte

/*	srcIP              [0:16]
	srcPort            [16:18]
	dstIP              [18:34]
	dstPort            [34:36]
	protocolIdentifier [36]
*/

func (k dynamicKey) Hash() (h uint64) {
	return fnvHash(k[:])
}

type DynamicKeySelector struct {
	network,
	transport,
	srcIP,
	dstIP,
	protocolIdentifier,
	srcPort,
	dstPort,
	bidirectional,
	fivetuple,
	empty bool
}

func (selector *DynamicKeySelector) makeDynamicKey(packet PacketBuffer) (flows.FlowKey, bool) {
	ret := dynamicKey{}
	if selector.network {
		network := packet.NetworkLayer()
		if network == nil {
			return nil, false
		}
		flow := network.NetworkFlow()
		if selector.srcIP {
			copy(ret[0:16], flow.Src().Raw())
		}
		if selector.dstIP {
			copy(ret[18:34], flow.Dst().Raw())
		}
		if selector.protocolIdentifier {
			ret[36] = byte(packet.Proto())
		}
	}
	if selector.transport {
		transport := packet.TransportLayer()
		if transport == nil {
			return nil, false
		}
		flow := transport.TransportFlow()
		if selector.srcPort {
			copy(ret[16:18], flow.Src().Raw())
		}
		if selector.dstPort {
			copy(ret[34:36], flow.Dst().Raw())
		}
	}
	forward := true
	if selector.bidirectional {
		if ret[36] == byte(layers.IPProtocolICMPv4) || ret[36] == byte(layers.IPProtocolICMPv6) {
			// sort key so that srcIP < dstIP
			if bytes.Compare(ret[0:16], ret[18:34]) > 0 {
				var tmp [16]byte
				copy(tmp[:], ret[0:16])
				copy(ret[0:16], ret[18:34])
				copy(ret[18:34], tmp[:])
				forward = false
			}
		} else {
			// sort key so that srcIPsrcPort < dstIPdstPort
			if bytes.Compare(ret[0:18], ret[18:36]) > 0 {
				var tmp [18]byte
				copy(tmp[:], ret[0:18])
				copy(ret[0:18], ret[18:36])
				copy(ret[18:36], tmp[:])
				forward = false
			}
		}
	}
	return ret, forward
}

type emptyKey struct{}

func (k emptyKey) Hash() (h uint64) {
	return 0
}

func makeEmptyKey(packet PacketBuffer) (flows.FlowKey, bool) {
	return emptyKey{}, false
}
