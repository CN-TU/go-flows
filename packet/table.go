package packet

import (
	"io"
	"log"
	"runtime"
	"sync"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"

	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var icmpEndpointType = gopacket.RegisterEndpointType(1000, gopacket.EndpointTypeMetadata{Name: "ICMP", Formatter: func(b []byte) string {
	return fmt.Sprintf("%d:%d", b[0], b[1])
}})

type ICMPv4Flow struct {
	layers.ICMPv4
}

func (i *ICMPv4Flow) TransportFlow() gopacket.Flow {
	return gopacket.NewFlow(icmpEndpointType, emptyPort, i.Contents[0:2])
}

type ICMPv6Flow struct {
	layers.ICMPv6
}

func (i *ICMPv6Flow) TransportFlow() gopacket.Flow {
	return gopacket.NewFlow(icmpEndpointType, emptyPort, i.Contents[0:2])
}

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

func (t *shallowMultiPacketBuffer) copy(src *shallowMultiPacketBuffer) {
	t.buffers = t.buffers[:len(src.buffers)]
	copy(t.buffers, src.buffers)
	t.multiBuffer = src.multiBuffer
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

var layerTypeIPv46 = gopacket.RegisterLayerType(1000, gopacket.LayerTypeMetadata{Name: "IPv4 or IPv6"})

const (
	batchSize      = 1000
	fullBuffers    = 10
	shallowBuffers = 10
)

func ReadFiles(fnames []string, plen int, flowtable EventTable) flows.Time {
	result := make(chan *multiPacketBuffer, fullBuffers)
	empty := make(chan *multiPacketBuffer, fullBuffers)

	prealloc := plen
	if plen == 0 {
		prealloc = 9000
	}

	for i := 0; i < fullBuffers; i++ {
		buf := &multiPacketBuffer{
			buffers: make([]*packetBuffer, batchSize),
			empty:   &empty,
		}
		for j := 0; j < batchSize; j++ {
			buf.buffers[j] = &packetBuffer{buffer: make([]byte, prealloc), multibuffer: buf}
		}
		empty <- buf
	}

	go func() {
		multiBuffer := <-empty
		pos := 0
		defer func() {
			if pos != 0 {
				multiBuffer.buffers = multiBuffer.buffers[:pos]
				result <- multiBuffer
			} else {
				empty <- multiBuffer
			}
			close(result)
			// consume empty buffers -> let every go routine finish
			for i := 0; i < fullBuffers; i++ {
				<-empty
			}
		}()
		for _, fname := range fnames {
			fhandle, err := pcap.OpenOffline(fname)
			if err != nil {
				log.Fatalf("Couldn't open file %s", fname)
			}
			var lt gopacket.LayerType
			switch fhandle.LinkType() {
			case layers.LinkTypeEthernet:
				lt = layers.LayerTypeEthernet
			case layers.LinkTypeRaw, layers.LinkType(12):
				lt = layerTypeIPv46
			default:
				log.Fatalf("File format not implemented")
			}
			for {
				data, ci, err := fhandle.ZeroCopyReadPacketData()
				if err == io.EOF {
					break
				} else if err != nil {
					log.Println("Error:", err)
					continue
				}
				dlen := len(data)
				buffer := multiBuffer.buffers[pos]
				pos++
				if plen == 0 && cap(buffer.buffer) < dlen {
					buffer.buffer = make([]byte, dlen)
				} else if dlen < cap(buffer.buffer) {
					buffer.buffer = buffer.buffer[0:dlen]
				} else {
					buffer.buffer = buffer.buffer[0:cap(buffer.buffer)]
				}
				clen := copy(buffer.buffer, data)
				buffer.time = flows.Time(ci.Timestamp.UnixNano())
				buffer.ci.CaptureInfo = ci
				buffer.ci.Truncated = ci.CaptureLength < ci.Length || clen < dlen
				buffer.first = lt
				if pos == batchSize {
					pos = 0
					result <- multiBuffer
					multiBuffer = <-empty
				}
			}
			fhandle.Close()
		}
	}()

	c := make(chan flows.Time)
	go func() {
		var time flows.Time
		discard := newShallowMultiPacketBuffer(batchSize)
		forward := newShallowMultiPacketBuffer(batchSize)
		for multibuffer := range result {
			discard.multiBuffer = multibuffer
			forward.multiBuffer = multibuffer
			for _, buffer := range multibuffer.buffers {
				if !buffer.decode() {
					//count non interesting packets?
					discard.add(buffer)
				} else {
					buffer.key, buffer.Forward = fivetuple(buffer)
					time = buffer.time
					if buffer.key != nil {
						forward.add(buffer)
					} else {
						discard.add(buffer)
					}
				}
			}
			flowtable.Event(forward)
			forward.reset()
			discard.recycle()
		}
		c <- time
	}()

	return <-c
}

type EventTable interface {
	Event(buffer *shallowMultiPacketBuffer)
	Expire()
	EOF(flows.Time)
}

type ParallelFlowTable struct {
	tables     []*flows.FlowTable
	full       []chan *shallowMultiPacketBuffer
	expire     []chan struct{}
	expirewg   sync.WaitGroup
	empty      []chan *shallowMultiPacketBuffer
	tmp        []*shallowMultiPacketBuffer
	wg         sync.WaitGroup
	expireTime flows.Time
	nextExpire flows.Time
}

type SingleFlowTable struct {
	table      *flows.FlowTable
	full       chan *shallowMultiPacketBuffer
	expire     chan struct{}
	empty      chan *shallowMultiPacketBuffer
	done       chan struct{}
	expireTime flows.Time
	nextExpire flows.Time
}

func (sft *SingleFlowTable) Expire() {
	sft.expire <- struct{}{}
	runtime.GC()
}

func (sft *SingleFlowTable) Event(buffer *shallowMultiPacketBuffer) {
	current := buffer.buffers[0].Timestamp()
	if current > sft.nextExpire {
		sft.Expire()
		sft.nextExpire = current + sft.expireTime
	}
	tmp := <-sft.empty
	tmp.copy(buffer)
	sft.full <- tmp
}

func (sft *SingleFlowTable) EOF(now flows.Time) {
	close(sft.full)
	<-sft.done
	sft.table.EOF(now)
}

func NewParallelFlowTable(num int, features flows.FeatureListCreator, newflow flows.FlowCreator, activeTimeout, idleTimeout, expire flows.Time) EventTable {
	if num == 1 {
		ret := &SingleFlowTable{
			table:      flows.NewFlowTable(features, newflow, activeTimeout, idleTimeout),
			expireTime: expire,
		}
		ret.full = make(chan *shallowMultiPacketBuffer, shallowBuffers)
		ret.expire = make(chan struct{}, 1)
		ret.empty = make(chan *shallowMultiPacketBuffer, shallowBuffers)
		for i := 0; i < shallowBuffers; i++ {
			ret.empty <- newShallowMultiPacketBuffer(batchSize)
		}
		ret.done = make(chan struct{})
		go func() {
			t := ret.table
			defer close(ret.done)
			for {
				select {
				case <-ret.expire:
					t.Expire()
				case buffer, ok := <-ret.full:
					if !ok {
						return
					}
					for _, b := range buffer.buffers {
						t.Event(b)
					}
					buffer.recycle()
					ret.empty <- buffer
				}
			}
		}()
		return ret
	}
	ret := &ParallelFlowTable{
		tables:     make([]*flows.FlowTable, num),
		full:       make([]chan *shallowMultiPacketBuffer, num),
		expire:     make([]chan struct{}, num),
		empty:      make([]chan *shallowMultiPacketBuffer, num),
		tmp:        make([]*shallowMultiPacketBuffer, num),
		expireTime: expire,
	}
	for i := 0; i < num; i++ {
		c := make(chan *shallowMultiPacketBuffer, shallowBuffers)
		expire := make(chan struct{}, 1)
		e := make(chan *shallowMultiPacketBuffer, shallowBuffers)
		ret.full[i] = c
		ret.expire[i] = expire
		ret.empty[i] = e
		for j := 0; j < shallowBuffers; j++ {
			ret.empty[i] <- newShallowMultiPacketBuffer(batchSize)
		}
		t := flows.NewFlowTable(features, newflow, activeTimeout, idleTimeout)
		ret.tables[i] = t
		ret.wg.Add(1)
		go func() {
			defer ret.wg.Done()
			for {
				select {
				case <-expire:
					t.Expire()
					ret.expirewg.Done()
				case buffer, ok := <-c:
					if !ok {
						return
					}
					for _, b := range buffer.buffers {
						t.Event(b)
					}
					buffer.recycle()
					e <- buffer
				}
			}
		}()
	}
	return ret
}

func (pft *ParallelFlowTable) Expire() {
	for _, e := range pft.expire {
		pft.expirewg.Add(1)
		e <- struct{}{}
	}
	pft.expirewg.Wait()
	runtime.GC()
}

func (pft *ParallelFlowTable) Event(buffer *shallowMultiPacketBuffer) {
	current := buffer.buffers[0].Timestamp()
	if current > pft.nextExpire {
		pft.Expire()
		pft.nextExpire = current + pft.expireTime
	}
	num := len(pft.tables)
	for i := 0; i < num; i++ {
		pft.tmp[i] = <-pft.empty[i]
		pft.tmp[i].multiBuffer = buffer.multiBuffer
	}
	for _, packet := range buffer.buffers {
		h := packet.key.Hash() % uint64(num)
		pft.tmp[h].add(packet)
	}
	for i := 0; i < num; i++ {
		pft.full[i] <- pft.tmp[i]
	}
}

func (pft *ParallelFlowTable) EOF(now flows.Time) {
	for _, c := range pft.full {
		close(c)
	}
	pft.wg.Wait()
	for _, t := range pft.tables {
		pft.wg.Add(1)
		go func(table *flows.FlowTable) {
			defer pft.wg.Done()
			table.EOF(now)
		}(t)
	}
	pft.wg.Wait()
}
