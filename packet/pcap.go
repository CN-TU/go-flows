package packet

import (
	"io"
	"log"

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

func ReadFiles(fnames []string, plen int, flowtable EventTable) flows.Time {
	result := make(chan *multiPacketBuffer, fullBuffers)
	empty := make(chan *multiPacketBuffer, fullBuffers)

	prealloc := plen
	if plen == 0 {
		prealloc = 4096
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
