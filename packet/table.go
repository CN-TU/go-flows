package packet

import (
	"io"
	"log"
	"sync"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

type PacketBuffer struct {
	packet  gopacket.Packet
	key     flows.FlowKey
	empty   *chan *PacketBuffer
	time    flows.Time
	Forward bool
	buffer  []byte
}

func (pb *PacketBuffer) Recycle() {
	*pb.empty <- pb
}

func (pb *PacketBuffer) Key() flows.FlowKey {
	return pb.key
}

func (pb *PacketBuffer) Timestamp() flows.Time {
	return pb.time
}

func ReadFiles(fnames []string, plen int) <-chan *PacketBuffer {
	result := make(chan *PacketBuffer, 1000)
	empty := make(chan *PacketBuffer, 1000)

	prealloc := plen
	if plen == 0 {
		prealloc = 9000
	}

	for i := 0; i < 1000; i++ {
		empty <- &PacketBuffer{empty: &empty, buffer: make([]byte, prealloc)}
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
				plen := len(data)
				buffer := <-empty
				if cap(buffer.buffer) < plen {
					buffer.buffer = make([]byte, plen)
				}
				clen := copy(buffer.buffer, data)
				buffer.buffer = buffer.buffer[:clen]
				buffer.packet = gopacket.NewPacket(buffer.buffer, decoder, options)
				m := buffer.packet.Metadata()
				m.CaptureInfo = ci
				m.Truncated = m.Truncated || ci.CaptureLength < ci.Length || clen < plen
				result <- buffer
			}
			fhandle.Close()
		}
	}()

	return result
}

func safeDecode(buffer *PacketBuffer) {
	defer func() {
		recover() //fixme: handle fatal decoding errors somehow
	}()
	buffer.packet.TransportLayer()
	buffer.key, buffer.Forward = fivetuple(buffer.packet)
}

func ParsePacket(in <-chan *PacketBuffer, flowtable EventTable) flows.Time {
	c := make(chan flows.Time)
	go func() {
		var time flows.Time
		for buffer := range in {
			safeDecode(buffer)
			time = flows.Time(buffer.packet.Metadata().Timestamp.UnixNano())
			buffer.time = time
			if buffer.key != nil {
				flowtable.Event(buffer)
			} else {
				buffer.Recycle()
			}
		}
		c <- time
	}()
	return <-c
}

type EventTable interface {
	Event(buffer *PacketBuffer)
	EOF(flows.Time)
}

type ParallelFlowTable struct {
	tables []*flows.FlowTable
	chans  []chan *PacketBuffer
	wg     sync.WaitGroup
}

type SingleFlowTable struct {
	table *flows.FlowTable
	c     chan *PacketBuffer
	d     chan struct{}
}

func (sft *SingleFlowTable) Event(buffer *PacketBuffer) {
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
		ret.c = make(chan *PacketBuffer, 1000)
		ret.d = make(chan struct{})
		go func() {
			t := ret.table
			for buffer := range ret.c {
				t.Event(buffer)
				buffer.Recycle()
			}
			close(ret.d)
		}()
		return ret
	}
	ret := &ParallelFlowTable{tables: make([]*flows.FlowTable, num), chans: make([]chan *PacketBuffer, num)}
	for i := 0; i < num; i++ {
		c := make(chan *PacketBuffer, 100)
		ret.chans[i] = c
		t := flows.NewFlowTable(features, newflow, activeTimeout, idleTimeout, checkpoint)
		ret.tables[i] = t
		ret.wg.Add(1)
		go func() {
			defer ret.wg.Done()
			for buffer := range c {
				t.Event(buffer)
				buffer.Recycle()
			}
		}()
	}
	return ret
}

func (pft *ParallelFlowTable) Event(buffer *PacketBuffer) {
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
