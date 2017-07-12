package packet

import (
	"io"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

var layerTypeIPv46 = gopacket.RegisterLayerType(1000, gopacket.LayerTypeMetadata{Name: "IPv4 or IPv6"})

const (
	batchSize      = 1000
	fullBuffers    = 10
	shallowBuffers = 10
)

type PcapBuffer struct {
	result chan *multiPacketBuffer
	empty  chan *multiPacketBuffer
	filter string
	plen   int
}

func NewPcapBuffer(plen int, flowtable EventTable) *PcapBuffer {
	ret := &PcapBuffer{make(chan *multiPacketBuffer, fullBuffers), make(chan *multiPacketBuffer, fullBuffers), "", plen}
	prealloc := plen
	if plen == 0 {
		prealloc = 4096
	}

	for i := 0; i < fullBuffers; i++ {
		buf := &multiPacketBuffer{
			buffers: make([]PacketBuffer, batchSize),
			empty:   &ret.empty,
		}
		for j := 0; j < batchSize; j++ {
			buf.buffers[j] = &pcapPacketBuffer{buffer: make([]byte, prealloc), multibuffer: buf}
		}
		ret.empty <- buf
	}

	go func() {
		discard := newShallowMultiPacketBuffer(batchSize)
		forward := newShallowMultiPacketBuffer(batchSize)
		for multibuffer := range ret.result {
			discard.multiBuffer = multibuffer
			forward.multiBuffer = multibuffer
			for _, buffer := range multibuffer.buffers {
				if !buffer.decode() {
					//count non interesting packets?
					discard.add(buffer)
				} else {
					key, fw := fivetuple(buffer)
					if key != nil {
						buffer.setInfo(key, fw)
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
	}()

	return ret
}

func (input *PcapBuffer) SetFilter(filter string) (old string) {
	old = input.filter
	input.filter = filter
	return
}

func (input *PcapBuffer) Finish() {
	close(input.result)
	// consume empty buffers -> let every go routine finish
	for i := 0; i < fullBuffers; i++ {
		<-input.empty
	}
}

func (input *PcapBuffer) ReadFile(fname string) flows.Time {
	fhandle, err := pcap.OpenOffline(fname)
	defer fhandle.Close()
	if err != nil {
		log.Fatalf("Couldn't open file %s", fname)
	}
	var filter *pcap.BPF
	if input.filter != "" {
		filter, err = fhandle.NewBPF(input.filter)
		if err != nil {
			log.Fatal(err)
		}
	}
	return input.readHandle(fhandle, filter)
}

func (input *PcapBuffer) ReadInterface(dname string) flows.Time {
	inactive, err := pcap.NewInactiveHandle(dname)
	if err != nil {
		log.Fatal(err)
	}
	defer inactive.CleanUp()

	if err = inactive.SetTimeout(100 * time.Millisecond); err != nil {
		log.Fatal(err)
	}

	handle, err := inactive.Activate()
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	if err = handle.SetBPFFilter(input.filter); err != nil {
		log.Fatal(err)
	}

	return input.readHandle(handle, nil)
}

func (input *PcapBuffer) readHandle(fhandle *pcap.Handle, filter *pcap.BPF) flows.Time {
	var time flows.Time
	multiBuffer := <-input.empty
	pos := 0
	defer func() {
		if pos != 0 {
			multiBuffer.buffers = multiBuffer.buffers[:pos]
			input.result <- multiBuffer
		} else {
			input.empty <- multiBuffer
		}
	}()

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
		log.Println("test")
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Error:", err)
			continue
		}
		if filter != nil && !filter.Matches(ci, data) {
			continue
		}
		dlen := len(data)
		buffer := multiBuffer.buffers[pos].(*pcapPacketBuffer)
		pos++
		if input.plen == 0 && cap(buffer.buffer) < dlen {
			buffer.buffer = make([]byte, dlen)
		} else if dlen < cap(buffer.buffer) {
			buffer.buffer = buffer.buffer[0:dlen]
		} else {
			buffer.buffer = buffer.buffer[0:cap(buffer.buffer)]
		}
		clen := copy(buffer.buffer, data)
		time = flows.Time(ci.Timestamp.UnixNano())
		buffer.time = time
		buffer.ci.CaptureInfo = ci
		buffer.ci.Truncated = ci.CaptureLength < ci.Length || clen < dlen
		buffer.first = lt
		if pos == batchSize {
			pos = 0
			input.result <- multiBuffer
			multiBuffer = <-input.empty
		}
	}
	return time
}
