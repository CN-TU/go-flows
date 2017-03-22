package packet

import (
	"io"
	"log"
	"sync"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const MAXLEN int = 9000 //FIXME

type PacketBuffer struct {
	buffer  [MAXLEN]byte
	packet  gopacket.Packet
	key     flows.FlowKey
	empty   *chan *PacketBuffer
	Forward bool
}

func (pb *PacketBuffer) Recycle() {
	*pb.empty <- pb
}

func (pb *PacketBuffer) Key() flows.FlowKey {
	return pb.key
}

func (pb *PacketBuffer) Timestamp() flows.Time {
	return flows.Time(pb.packet.Metadata().Timestamp.UnixNano())
}

func ReadFiles(fnames []string) <-chan *PacketBuffer {
	result := make(chan *PacketBuffer, 1000)
	empty := make(chan *PacketBuffer, 1000)

	go func() {
		defer close(result)
		for i := 0; i < 1000; i++ {
			empty <- &PacketBuffer{empty: &empty}
		}
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
				buffer := <-empty
				copy(buffer.buffer[:], data)
				plen := len(data)
				if plen > MAXLEN {
					plen = MAXLEN
				}
				buffer.packet = gopacket.NewPacket(buffer.buffer[:plen], decoder, options)
				m := buffer.packet.Metadata()
				m.CaptureInfo = ci
				m.Truncated = m.Truncated || ci.CaptureLength < ci.Length || plen < len(data)
				result <- buffer
			}
			fhandle.Close()
		}
	}()

	return result
}

func ParsePacket(in <-chan *PacketBuffer, flowtable EventTable) {
	c := make(chan struct{})
	go func() {
		for packet := range in {
			packet.packet.TransportLayer()
			packet.key, packet.Forward = fivetuple(packet.packet)
			if packet.key != nil {
				flowtable.Event(packet)
			} else {
				packet.Recycle()
			}
		}
		close(c)
	}()
	<-c
}

type EventTable interface {
	Event(buffer *PacketBuffer)
	EOF()
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

func (sft *SingleFlowTable) EOF() {
	close(sft.c)
	<-sft.d
	sft.table.EOF()
}

func NewParallelFlowTable(num int, features func(*flows.BaseFlow) flows.FeatureList, newflow func(flows.Event, *flows.FlowTable, flows.FlowKey) flows.Flow) EventTable {
	if num == 1 {
		ret := &SingleFlowTable{table: flows.NewFlowTable(features, newflow)}
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
		t := flows.NewFlowTable(features, newflow)
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

func (pft *ParallelFlowTable) EOF() {
	for _, c := range pft.chans {
		close(c)
	}
	pft.wg.Wait()
	for _, t := range pft.tables {
		pft.wg.Add(1)
		go func(table *flows.FlowTable) {
			defer pft.wg.Done()
			table.EOF()
		}(t)
	}
	pft.wg.Wait()
}
