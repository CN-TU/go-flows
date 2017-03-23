package packet

import (
	"io"
	"log"
	"sync"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

type packetBuffer struct {
	packet  gopacket.Packet
	key     flows.FlowKey
	empty   *chan *packetBuffer
	time    flows.Time
	Forward bool
	buffer  []byte
}

func (pb *packetBuffer) recycle() {
	*pb.empty <- pb
}

func (pb *packetBuffer) Key() flows.FlowKey {
	return pb.key
}

func (pb *packetBuffer) Timestamp() flows.Time {
	return pb.time
}

func ReadFiles(fnames []string, plen int) <-chan *packetBuffer {
	result := make(chan *packetBuffer, 1000)
	empty := make(chan *packetBuffer, 1000)

	prealloc := plen
	if plen == 0 {
		prealloc = 9000
	}

	for i := 0; i < 1000; i++ {
		empty <- &packetBuffer{empty: &empty, buffer: make([]byte, prealloc)}
	}

	go func() {
		defer close(result)
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
				dlen := len(data)
				buffer := <-empty
				if plen == 0 && cap(buffer.buffer) < dlen {
					buffer.buffer = make([]byte, dlen)
				} else if dlen < cap(buffer.buffer) {
					buffer.buffer = buffer.buffer[:dlen]
				} else {
					buffer.buffer = buffer.buffer[:cap(buffer.buffer)]
				}
				clen := copy(buffer.buffer, data)
				buffer.packet = gopacket.NewPacket(buffer.buffer, decoder, options)
				m := buffer.packet.Metadata()
				m.CaptureInfo = ci
				m.Truncated = m.Truncated || ci.CaptureLength < ci.Length || clen < dlen
				result <- buffer
			}
			fhandle.Close()
		}
	}()

	return result
}

func ParsePacket(in <-chan *packetBuffer, flowtable EventTable) flows.Time {
	c := make(chan flows.Time)
	go func() {
		var time flows.Time
		for buffer := range in {
			buffer.packet.TransportLayer()
			buffer.key, buffer.Forward = fivetuple(buffer.packet)
			time = flows.Time(buffer.packet.Metadata().Timestamp.UnixNano())
			buffer.time = time
			if buffer.key != nil {
				flowtable.Event(buffer)
			} else {
				buffer.recycle()
			}
		}
		c <- time
	}()
	return <-c
}

type EventTable interface {
	Event(buffer *packetBuffer)
	EOF(flows.Time)
}

type ParallelFlowTable struct {
	tables []*flows.FlowTable
	chans  []chan *packetBuffer
	wg     sync.WaitGroup
}

type SingleFlowTable struct {
	table *flows.FlowTable
	c     chan *packetBuffer
	d     chan struct{}
}

func (sft *SingleFlowTable) Event(buffer *packetBuffer) {
	sft.c <- buffer
}

func (sft *SingleFlowTable) EOF(now flows.Time) {
	close(sft.c)
	<-sft.d
	sft.table.EOF(now)
}

func NewParallelFlowTable(num int, features flows.FeatureCreator, newflow flows.FlowCreator, activeTimeout, idleTimeout, checkpoint flows.Time) EventTable {
	if num == 1 {
		ret := &SingleFlowTable{table: flows.NewFlowTable(features, newflow, activeTimeout, idleTimeout, checkpoint)}
		ret.c = make(chan *packetBuffer, 1000)
		ret.d = make(chan struct{})
		go func() {
			t := ret.table
			for buffer := range ret.c {
				t.Event(buffer)
				buffer.recycle()
			}
			close(ret.d)
		}()
		return ret
	}
	ret := &ParallelFlowTable{tables: make([]*flows.FlowTable, num), chans: make([]chan *packetBuffer, num)}
	for i := 0; i < num; i++ {
		c := make(chan *packetBuffer, 100)
		ret.chans[i] = c
		t := flows.NewFlowTable(features, newflow, activeTimeout, idleTimeout, checkpoint)
		ret.tables[i] = t
		ret.wg.Add(1)
		go func() {
			defer ret.wg.Done()
			for buffer := range c {
				t.Event(buffer)
				buffer.recycle()
			}
		}()
	}
	return ret
}

func (pft *ParallelFlowTable) Event(buffer *packetBuffer) {
	h := buffer.key.Hash() % uint64(len(pft.tables))
	pft.chans[h] <- buffer
}

func (pft *ParallelFlowTable) EOF(now flows.Time) {
	for _, c := range pft.chans {
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
