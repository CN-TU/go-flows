package packet

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type multiPacketBuffer struct {
	buffers      []*packetBuffer
	empty        *chan *multiPacketBuffer
	recycled     int
	recycleMutex sync.Mutex
}

type shallowMultiPacketBuffer struct {
	buffers     []*packetBuffer
	multiBuffer *multiPacketBuffer
}

func newShallowMultiPacketBuffer(size int) *shallowMultiPacketBuffer {
	return &shallowMultiPacketBuffer{buffers: make([]*packetBuffer, 0, size)}
}

func (b *shallowMultiPacketBuffer) add(p *packetBuffer) {
	b.buffers = append(b.buffers, p)
}

func (b *shallowMultiPacketBuffer) copy(src *shallowMultiPacketBuffer) {
	b.buffers = b.buffers[:len(src.buffers)]
	copy(b.buffers, src.buffers)
	b.multiBuffer = src.multiBuffer
}

func (b *shallowMultiPacketBuffer) reset() {
	b.buffers = b.buffers[:0]
	b.multiBuffer = nil
}

func (b *shallowMultiPacketBuffer) recycle() {
	for _, buffer := range b.buffers {
		buffer.recycle()
	}
	b.multiBuffer.recycle(len(b.buffers))
	b.buffers = b.buffers[:0]
	b.multiBuffer = nil
}

func (b *multiPacketBuffer) recycle(num int) {
	b.recycleMutex.Lock()
	b.recycled += num
	if b.recycled == len(b.buffers) {
		b.recycled = 0
		*b.empty <- b
	}
	b.recycleMutex.Unlock()
}

type packetBuffer struct {
	key         flows.FlowKey
	multibuffer *multiPacketBuffer
	time        flows.Time
	buffer      []byte
	first       gopacket.LayerType
	eth         layers.Ethernet
	ip4         layers.IPv4
	ip6         layers.IPv6
	tcp         layers.TCP
	udp         layers.UDP
	icmpv4      ICMPv4Flow
	icmpv6      ICMPv6Flow
	link        gopacket.LinkLayer
	network     gopacket.NetworkLayer
	transport   gopacket.TransportLayer
	application gopacket.ApplicationLayer
	failure     gopacket.ErrorLayer
	ci          gopacket.PacketMetadata
	Forward     bool
}

func (pb *packetBuffer) recycle() {
	pb.link = nil
	pb.network = nil
	pb.transport = nil
	pb.application = nil
	pb.failure = nil
	pb.tcp.Payload = nil
}

func (pb *packetBuffer) Key() flows.FlowKey {
	return pb.key
}

func (pb *packetBuffer) Timestamp() flows.Time {
	return pb.time
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
	if pb.link.LayerType() == lt {
		return pb.link
	}
	if pb.network.LayerType() == lt {
		return pb.network
	}
	if pb.transport.LayerType() == lt {
		return pb.transport
	}
	return nil
}
func (pb *packetBuffer) LayerClass(lc gopacket.LayerClass) gopacket.Layer {
	if lc.Contains(pb.link.LayerType()) {
		return pb.link
	}
	if lc.Contains(pb.network.LayerType()) {
		return pb.network
	}
	if lc.Contains(pb.transport.LayerType()) {
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

//custom decoder for fun and speed. Borrowed from DecodingLayerParser
func (pb *packetBuffer) decode() (ret bool) {
	defer func(r *bool) {
		if err := recover(); err != nil {
			if pb.tcp.Payload != nil {
				*r = true //fully decoded tcp packet except for options; should we count that as valid?
			} else {
				*r = false
			}
			//count decoding errors?
		}
	}(&ret)
	typ := pb.first
	var decoder gopacket.DecodingLayer
	data := pb.buffer
	for len(data) > 0 {
		switch typ {
		case layers.LayerTypeEthernet:
			decoder = &pb.eth
		case layers.LayerTypeIPv4:
			decoder = &pb.ip4
		case layers.LayerTypeIPv6:
			decoder = &pb.ip6
		case layers.LayerTypeUDP:
			decoder = &pb.udp
		case layers.LayerTypeTCP:
			decoder = &pb.tcp
		case layers.LayerTypeICMPv4:
			decoder = &pb.icmpv4
		case layers.LayerTypeICMPv6:
			decoder = &pb.icmpv6
		case layerTypeIPv46:
			version := data[0] >> 4
			switch version {
			case 4:
				decoder = &pb.ip4
				typ = layers.LayerTypeIPv4
			case 6:
				decoder = &pb.ip6
				typ = layers.LayerTypeIPv6
			default:
				return false
			}
		default:
			return false
		}
		if err := decoder.DecodeFromBytes(data, pb); err != nil {
			return false
		}
		switch typ {
		case layers.LayerTypeEthernet:
			pb.link = &pb.eth
		case layers.LayerTypeIPv4:
			pb.network = &pb.ip4
		case layers.LayerTypeIPv6:
			pb.network = &pb.ip6
		case layers.LayerTypeUDP:
			pb.transport = &pb.udp
			return true
		case layers.LayerTypeTCP:
			pb.transport = &pb.tcp
			return true
		case layers.LayerTypeICMPv4:
			pb.transport = &pb.icmpv4
			return true
		case layers.LayerTypeICMPv6:
			pb.transport = &pb.icmpv6
			return true
		}
		typ = decoder.NextLayerType()
		data = decoder.LayerPayload()
	}
	return false
}
