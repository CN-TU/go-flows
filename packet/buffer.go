package packet

import (
	"encoding/binary"
	"log"
	"sync/atomic"

	"github.com/CN-TU/go-flows/flows"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Buffer provides the packet interface from gopacket and some additional utility functions
//
// Never hold references to this buffer or modify it, since it will be reused!
// If you need to keep a packet around for a short time, it must be copied with Copy() and this
// copy later destroyed with Recycle()
type Buffer interface {
	gopacket.Packet
	//// Functions for querying additional packet attributes
	//// ------------------------------------------------------------------
	// Forward returns true if this packet is in the forward direction
	Forward() bool
	// Timestamp returns the point in time this packet was captured
	Timestamp() flows.DateTimeNanoseconds
	// Key returns the flow key of this packet
	Key() string
	// EtherType returns the EthernetType of the link layer
	EtherType() layers.EthernetType
	// Proto returns the protocol field
	Proto() uint8
	// Label returns the label of this packet, if one ones set
	Label() interface{}
	// PacketNr returns the the number of this packet
	PacketNr() uint64
	//// Convenience functions for packet size calculations
	//// ------------------------------------------------------------------
	// LinkLayerLength returns the length of the link layer (=header + payload) or 0 if there is no link layer
	LinkLayerLength() int
	// NetworkLayerLength returns the length of the network layer (=header + payload) or 0 if there is no network layer
	NetworkLayerLength() int
	// PayloadLength returns the length of the payload or 0 if there is no application layer
	PayloadLength() int
	//// Functions for holding on to packets
	//// ------------------------------------------------------------------
	// Copy reserves the buffer, creates a reference, and returns it. Use this if you need to hold on to a packet.
	Copy() Buffer
	// Recycle frees this buffer
	Recycle()
	//// Internal interface - don't use (necessa)
	//// ------------------------------------------------------------------
	// SetInfo sets the flowkey and the packet direction
	SetInfo(string, bool)

	decode() bool
}

type packetBuffer struct {
	inUse       int32
	owner       *multiPacketBuffer
	key         string
	time        flows.DateTimeNanoseconds
	buffer      []byte
	first       gopacket.LayerType
	sll         layers.LinuxSLL
	eth         layers.Ethernet
	ip4         layers.IPv4
	ip6         layers.IPv6
	ip6skipper  layers.IPv6ExtensionSkipper
	tcp         layers.TCP
	udp         layers.UDP
	icmpv4      icmpv4Flow
	icmpv6      icmpv6Flow
	link        gopacket.LinkLayer
	network     gopacket.NetworkLayer
	transport   gopacket.TransportLayer
	application gopacket.ApplicationLayer
	failure     gopacket.ErrorLayer
	ci          gopacket.PacketMetadata
	label       interface{}
	ip6headers  int
	refcnt      int
	packetnr    uint64
	ethertype   layers.EthernetType
	proto       uint8
	forward     bool
	resize      bool
}

// SerializableLayerType holds a packet layer, which can be serialized. This is needed for feature testing
type SerializableLayerType interface {
	gopacket.SerializableLayer
	LayerContents() []byte // gopacket.Layer
}

var layerSerializeLengthBufferScratch []byte

type layerSerializeLengthBuffer struct {
	len int
}

func (w *layerSerializeLengthBuffer) Bytes() []byte {
	return layerSerializeLengthBufferScratch
}

func (w *layerSerializeLengthBuffer) PrependBytes(num int) ([]byte, error) {
	w.len += num
	return layerSerializeLengthBufferScratch, nil
}

func (w *layerSerializeLengthBuffer) AppendBytes(num int) ([]byte, error) {
	w.len += num
	return layerSerializeLengthBufferScratch, nil
}

func (w *layerSerializeLengthBuffer) Clear() error {
	w.len = 0
	return nil
}

// BufferFromLayers creates a new Buffer for the given time and layers. Used for testing.
func BufferFromLayers(when flows.DateTimeNanoseconds, layerList ...SerializableLayerType) Buffer {
	pb := &packetBuffer{}
	pb.time = when
	for _, layer := range layerList {
		switch layer.LayerType() {
		case layers.LayerTypeEthernet:
			pb.first = layers.LayerTypeEthernet
			if pb.link != nil {
				log.Panic("Can only assign one Link Layer")
			}
			pb.link = layer.(gopacket.LinkLayer)
		case layers.LayerTypeLinuxSLL:
			pb.first = layers.LayerTypeLinuxSLL
			if pb.link != nil {
				log.Panic("Can only assign one Link Layer")
			}
			pb.link = layer.(gopacket.LinkLayer)
		case layers.LayerTypeIPv4:
			if pb.first != layers.LayerTypeEthernet {
				pb.first = layers.LayerTypeIPv4
			}
			if pb.network != nil {
				log.Panic("Can only assign one Network Layer")
			}
			pb.network = layer.(gopacket.NetworkLayer)
			pb.proto = uint8(pb.ip4.Protocol)
		case layers.LayerTypeIPv6:
			if pb.first != layers.LayerTypeEthernet {
				pb.first = layers.LayerTypeIPv6
			}
			if pb.network != nil {
				log.Panic("Can only assign one Network Layer")
			}
			pb.network = layer.(gopacket.NetworkLayer)
			pb.proto = uint8(pb.ip6.NextHeader)
		case layers.LayerTypeUDP:
			layer.(*layers.UDP).SetInternalPortsForTesting()
			if pb.first != layers.LayerTypeEthernet && pb.first != layers.LayerTypeIPv4 && pb.first != layers.LayerTypeIPv6 {
				pb.first = layers.LayerTypeUDP
			}
			if pb.transport != nil {
				log.Panic("Can only assign one Transport Layer")
			}
			pb.transport = layer.(gopacket.TransportLayer)
			if pb.proto == 0 {
				pb.proto = uint8(layers.IPProtocolUDP)
			}
		case layers.LayerTypeTCP:
			layer.(*layers.TCP).SetInternalPortsForTesting()
			if pb.first != layers.LayerTypeEthernet && pb.first != layers.LayerTypeIPv4 && pb.first != layers.LayerTypeIPv6 {
				pb.first = layers.LayerTypeTCP
			}
			if pb.transport != nil {
				log.Panic("Can only assign one Transport Layer")
			}
			pb.transport = layer.(gopacket.TransportLayer)
			if pb.proto == 0 {
				pb.proto = uint8(layers.IPProtocolTCP)
			}
		case layers.LayerTypeICMPv4:
			if pb.first != layers.LayerTypeEthernet && pb.first != layers.LayerTypeIPv4 && pb.first != layers.LayerTypeIPv6 {
				pb.first = layers.LayerTypeICMPv4
			}
			if pb.transport != nil {
				log.Panic("Can only assign one Transport Layer")
			}
			pb.transport = layer.(gopacket.TransportLayer)
			if pb.proto == 0 {
				pb.proto = uint8(layers.IPProtocolICMPv4)
			}
		case layers.LayerTypeICMPv6:
			if pb.first != layers.LayerTypeEthernet && pb.first != layers.LayerTypeIPv4 && pb.first != layers.LayerTypeIPv6 {
				pb.first = layers.LayerTypeICMPv6
			}
			if pb.transport != nil {
				log.Panic("Can only assign one Transport Layer")
			}
			pb.transport = layer.(gopacket.TransportLayer)
			if pb.proto == 0 {
				pb.proto = uint8(layers.IPProtocolICMPv6)
			}
		default:
			switch ip6e := layer.(type) {
			case *layers.IPv6Destination:
				pb.proto = uint8(ip6e.NextHeader)
				pb.ip6headers += len(layer.LayerContents())
			case *layers.IPv6HopByHop:
				pb.proto = uint8(ip6e.NextHeader)
				pb.ip6headers += len(layer.LayerContents())
			default:
				log.Panic("Protocol not supported")
			}
		}
	}
	if pb.network == nil {
		//add empty network layer (IPv4)
		if pb.first != layers.LayerTypeEthernet {
			pb.first = layers.LayerTypeIPv4
		}
		pb.ip4.SrcIP = []byte{0, 0, 0, 1}
		pb.ip4.DstIP = []byte{0, 0, 0, 2}
		pb.network = &pb.ip4
	}
	if pb.transport == nil {
		pb.transport = &pb.udp
		if pb.proto == 0 {
			pb.proto = uint8(layers.IPProtocolUDP)
		}
	}
	return pb
}

func (pb *packetBuffer) Proto() uint8 {
	return pb.proto
}

func (pb *packetBuffer) EtherType() layers.EthernetType {
	return pb.ethertype
}

func (pb *packetBuffer) PacketNr() uint64 {
	return pb.packetnr
}

func (pb *packetBuffer) Copy() Buffer {
	pb.refcnt++
	return pb
}

func (pb *packetBuffer) assign(data []byte, ci gopacket.CaptureInfo, lt gopacket.LayerType, packetnr uint64) flows.DateTimeNanoseconds {
	pb.link = nil
	pb.network = nil
	pb.transport = nil
	pb.application = nil
	pb.failure = nil
	pb.tcp.Payload = nil
	pb.proto = 0
	pb.ip6headers = 0
	pb.refcnt = 1
	dlen := len(data)
	if pb.resize && cap(pb.buffer) < dlen {
		pb.buffer = make([]byte, dlen)
	} else if dlen < cap(pb.buffer) {
		pb.buffer = pb.buffer[0:dlen]
	} else {
		pb.buffer = pb.buffer[0:cap(pb.buffer)]
	}
	clen := copy(pb.buffer, data)
	pb.time = flows.DateTimeNanoseconds(ci.Timestamp.UnixNano())
	pb.ci.CaptureInfo = ci
	pb.ci.Truncated = ci.CaptureLength < ci.Length || clen < dlen
	pb.first = lt
	pb.packetnr = packetnr
	return pb.time
}

func (pb *packetBuffer) canRecycle() bool {
	pb.refcnt--
	if pb.refcnt > 0 {
		return false
	}
	return true
}

func (pb *packetBuffer) Recycle() {
	if !pb.canRecycle() {
		return
	}
	atomic.StoreInt32(&pb.inUse, 0)
	pb.owner.free(1)
}

func (pb *packetBuffer) Key() string {
	return pb.key
}

func (pb *packetBuffer) Timestamp() flows.DateTimeNanoseconds {
	return pb.time
}

func (pb *packetBuffer) Forward() bool {
	return pb.forward
}

func (pb *packetBuffer) SetInfo(key string, forward bool) {
	pb.key = key
	pb.forward = forward
}

//DecodeFeedback
func (pb *packetBuffer) SetTruncated() { pb.ci.Truncated = true }

//gopacket.Packet
func (pb *packetBuffer) String() string { return "PacketBuffer" }
func (pb *packetBuffer) Dump() string   { return "" }
func (pb *packetBuffer) Layers() []gopacket.Layer {
	ret := make([]gopacket.Layer, 0, 3)
	if pb.link != nil {
		ret = append(ret, pb.link)
	}
	if pb.network != nil {
		ret = append(ret, pb.network)
	}
	if pb.transport != nil {
		ret = append(ret, pb.transport)
	}
	return ret
}
func (pb *packetBuffer) Layer(lt gopacket.LayerType) gopacket.Layer {
	if pb.link != nil && pb.link.LayerType() == lt {
		return pb.link
	}
	if pb.network != nil && pb.network.LayerType() == lt {
		return pb.network
	}
	if pb.transport != nil && pb.transport.LayerType() == lt {
		return pb.transport
	}
	return nil
}
func (pb *packetBuffer) LayerClass(lc gopacket.LayerClass) gopacket.Layer {
	if pb.link != nil && lc.Contains(pb.link.LayerType()) {
		return pb.link
	}
	if pb.network != nil && lc.Contains(pb.network.LayerType()) {
		return pb.network
	}
	if pb.transport != nil && lc.Contains(pb.transport.LayerType()) {
		return pb.transport
	}
	return nil
}
func (pb *packetBuffer) LinkLayer() gopacket.LinkLayer               { return pb.link }
func (pb *packetBuffer) NetworkLayer() gopacket.NetworkLayer         { return pb.network }
func (pb *packetBuffer) TransportLayer() gopacket.TransportLayer     { return pb.transport }
func (pb *packetBuffer) ApplicationLayer() gopacket.ApplicationLayer { return nil }
func (pb *packetBuffer) ErrorLayer() gopacket.ErrorLayer             { return nil }
func (pb *packetBuffer) Data() []byte                                { return pb.buffer }
func (pb *packetBuffer) Metadata() *gopacket.PacketMetadata          { return &pb.ci }
func (pb *packetBuffer) Label() interface{}                          { return pb.label }

func (pb *packetBuffer) LinkLayerLength() int {
	if eth, ok := pb.link.(*layers.Ethernet); ok && eth.Length != 0 {
		return int(eth.Length)
	}
	return pb.ci.CaptureLength
}

func (pb *packetBuffer) NetworkLayerLength() int {
	if ip, ok := pb.network.(*layers.IPv4); ok {
		return int(ip.Length)
	}
	if ip, ok := pb.network.(*layers.IPv6); ok {
		if ip.HopByHop != nil {
			var tlv *layers.IPv6HopByHopOption
			for _, t := range ip.HopByHop.Options {
				if t.OptionType == layers.IPv6HopByHopOptionJumbogram {
					tlv = t
					break
				}
			}
			if tlv != nil && len(tlv.OptionData) == 4 {
				l := binary.BigEndian.Uint32(tlv.OptionData)
				if l > 65535 {
					return int(l) + 40 // in ip6 the Length field is actually payload length...
				}
			}
		}
		return int(ip.Length) + 40 // in ip6 the Length field is actually payload length...
	}
	if pb.link != nil {
		return pb.LinkLayerLength() - len(pb.link.LayerContents())
	}
	return 0 // we don't know
}

func (pb *packetBuffer) PayloadLength() int {
	if pb.transport != nil {
		if pb.network != nil {
			return pb.NetworkLayerLength() - len(pb.network.LayerContents()) - len(pb.transport.LayerContents()) - pb.ip6headers
		}
	}
	return 0
}

//custom decoder for fun and speed. Borrowed from DecodingLayerParser
func (pb *packetBuffer) decode() (ret bool) {
	defer func() {
		if err := recover(); err != nil {
			ret = false
		}
	}()

	typ := pb.first
	data := pb.buffer

	// link layer
	if typ == layers.LayerTypeEthernet {
		if err := pb.eth.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.link = &pb.eth
		typ = pb.eth.NextLayerType()
		data = pb.eth.LayerPayload()
		pb.ethertype = pb.eth.EthernetType
		if typ == layers.LayerTypeLLC {
			var l layers.LLC
			if err := l.DecodeFromBytes(data, pb); err != nil {
				return false
			}
			typ = l.NextLayerType()
			data = l.LayerPayload()
			//SMELL: this might be bad
			pb.ethertype = layers.EthernetType(uint16(l.DSAP&0x7F)<<8 | uint16(l.SSAP&0x7F))
			if typ == layers.LayerTypeSNAP {
				var s layers.SNAP
				if err := s.DecodeFromBytes(data, pb); err != nil {
					return false
				}
				typ = s.NextLayerType()
				data = s.LayerPayload()
				pb.ethertype = s.Type
			}
		}
	} else if typ == layers.LayerTypeLinuxSLL {
		if err := pb.sll.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.link = &pb.sll
		typ = pb.sll.NextLayerType()
		data = pb.sll.LayerPayload()
	} else if typ == LayerTypeIPv46 {
		version := data[0] >> 4
		switch version {
		case 4:
			typ = layers.LayerTypeIPv4
		case 6:
			typ = layers.LayerTypeIPv6
		default:
			return false
		}
	}

	if len(data) == 0 {
		return true
	}

	// network layer
	if typ == layers.LayerTypeIPv4 {
		if err := pb.ip4.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.network = &pb.ip4
		pb.proto = uint8(pb.ip4.Protocol)
		if data[3] == 0 && data[2] == 0 && pb.ci.Truncated {
			// ip.Length == 0; e.g. windows TSO
			// fix length if packet is truncated...
			newlen := pb.ci.CaptureLength
			if pb.link != nil {
				newlen -= len(pb.link.LayerContents())
			}
			pb.ip4.Length = uint16(newlen)
		}
		typ = pb.ip4.NextLayerType()
		data = pb.ip4.LayerPayload()
	} else if typ == layers.LayerTypeIPv6 {
		if err := pb.ip6.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.network = &pb.ip6
		pb.proto = uint8(pb.ip6.NextHeader)
		if pb.proto == 0 { //fix hopbyhop
			pb.proto = uint8(pb.ip6.HopByHop.NextHeader)
		}
		typ = pb.ip6.NextLayerType()
		data = pb.ip6.LayerPayload()
		for layers.LayerClassIPv6Extension.Contains(typ) {
			if err := pb.ip6skipper.DecodeFromBytes(data, pb); err != nil {
				return false
			}
			pb.proto = uint8(pb.ip6skipper.NextHeader)
			pb.ip6headers += len(pb.ip6skipper.Contents)
			typ = pb.ip6skipper.NextLayerType()
			data = pb.ip6skipper.LayerPayload()
		}
	}

	if len(data) == 0 {
		return true
	}

	// transport layer
	switch typ {
	case layers.LayerTypeUDP:
		if err := pb.udp.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.transport = &pb.udp
		return true
	case layers.LayerTypeTCP:
		if err := pb.tcp.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.transport = &pb.tcp
		return true
	case layers.LayerTypeICMPv4:
		if err := pb.icmpv4.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.transport = &pb.icmpv4
		return true
	case layers.LayerTypeICMPv6:
		if err := pb.icmpv6.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		pb.transport = &pb.icmpv6
		return true
	}
	return true
}
