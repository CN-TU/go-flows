package packet

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type multiPacketBuffer struct {
	buffers      []PacketBuffer
	empty        *chan *multiPacketBuffer
	recycleMutex sync.Mutex
	recycled     int
	pos          int
}

type shallowMultiPacketBuffer struct {
	buffers     []PacketBuffer
	multiBuffer *multiPacketBuffer
}

func newShallowMultiPacketBuffer(size int) *shallowMultiPacketBuffer {
	return &shallowMultiPacketBuffer{buffers: make([]PacketBuffer, 0, size)}
}

func (b *shallowMultiPacketBuffer) add(p PacketBuffer) {
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
		b.buffers = b.buffers[:cap(b.buffers)]
		*b.empty <- b
	}
	b.recycleMutex.Unlock()
}

func (b *multiPacketBuffer) current() (ret PacketBuffer) {
	ret = b.buffers[b.pos]
	b.pos++
	return
}

func (b *multiPacketBuffer) finished() bool {
	return b.pos == batchSize
}

func (b *multiPacketBuffer) reset() {
	b.pos = 0
}

func (b *multiPacketBuffer) halfFull() (ret bool) {
	ret = false
	if b.pos != 0 {
		ret = true
		b.buffers = b.buffers[:b.pos]
	}
	return
}

type PacketBuffer interface {
	gopacket.Packet
	Forward() bool
	Timestamp() flows.Time
	Key() flows.FlowKey
	setInfo(flows.FlowKey, bool)
	recycle()
	decode() bool
}

type pcapPacketBuffer struct {
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
	forward     bool
	resize      bool
}

func (pb *pcapPacketBuffer) assign(data []byte, ci gopacket.CaptureInfo, lt gopacket.LayerType) {
	dlen := len(data)
	if pb.resize && cap(pb.buffer) < dlen {
		pb.buffer = make([]byte, dlen)
	} else if dlen < cap(pb.buffer) {
		pb.buffer = pb.buffer[0:dlen]
	} else {
		pb.buffer = pb.buffer[0:cap(pb.buffer)]
	}
	clen := copy(pb.buffer, data)
	pb.time = flows.Time(ci.Timestamp.UnixNano())
	pb.ci.CaptureInfo = ci
	pb.ci.Truncated = ci.CaptureLength < ci.Length || clen < dlen
	pb.first = lt
}

func (pb *pcapPacketBuffer) recycle() {
	pb.link = nil
	pb.network = nil
	pb.transport = nil
	pb.application = nil
	pb.failure = nil
	pb.tcp.Payload = nil
}

func (pb *pcapPacketBuffer) Key() flows.FlowKey {
	return pb.key
}

func (pb *pcapPacketBuffer) Timestamp() flows.Time {
	return pb.time
}

func (pb *pcapPacketBuffer) Forward() bool {
	return pb.forward
}

func (pb *pcapPacketBuffer) setInfo(key flows.FlowKey, forward bool) {
	pb.key = key
	pb.forward = forward
}

//DecodeFeedback
func (pb *pcapPacketBuffer) SetTruncated() { pb.ci.Truncated = true }

//gopacket.Packet
func (pb *pcapPacketBuffer) String() string { return "PacketBuffer" }
func (pb *pcapPacketBuffer) Dump() string   { return "" }
func (pb *pcapPacketBuffer) Layers() []gopacket.Layer {
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
func (pb *pcapPacketBuffer) Layer(lt gopacket.LayerType) gopacket.Layer {
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
func (pb *pcapPacketBuffer) LayerClass(lc gopacket.LayerClass) gopacket.Layer {
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
func (pb *pcapPacketBuffer) LinkLayer() gopacket.LinkLayer               { return pb.link }
func (pb *pcapPacketBuffer) NetworkLayer() gopacket.NetworkLayer         { return pb.network }
func (pb *pcapPacketBuffer) TransportLayer() gopacket.TransportLayer     { return pb.transport }
func (pb *pcapPacketBuffer) ApplicationLayer() gopacket.ApplicationLayer { return nil }
func (pb *pcapPacketBuffer) ErrorLayer() gopacket.ErrorLayer             { return nil }
func (pb *pcapPacketBuffer) Data() []byte                                { return pb.buffer }
func (pb *pcapPacketBuffer) Metadata() *gopacket.PacketMetadata          { return &pb.ci }

//custom decoder for fun and speed. Borrowed from DecodingLayerParser
func (pb *pcapPacketBuffer) decode() (ret bool) {
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