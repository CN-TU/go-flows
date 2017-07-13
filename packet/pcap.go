package packet

import (
	"io"
	"log"
	"os"
	"os/signal"
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
	result  chan *multiPacketBuffer
	empty   chan *multiPacketBuffer
	current *multiPacketBuffer
	filter  string
	plen    int
}

func NewPcapBuffer(plen int, flowtable EventTable) *PcapBuffer {
	ret := &PcapBuffer{
		result: make(chan *multiPacketBuffer, fullBuffers),
		empty:  make(chan *multiPacketBuffer, fullBuffers),
		plen:   plen}
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
			buf.buffers[j] = &pcapPacketBuffer{buffer: make([]byte, prealloc), multibuffer: buf, resize: plen == 0}
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

	ret.current = <-ret.empty

	return ret
}

func (input *PcapBuffer) SetFilter(filter string) (old string) {
	old = input.filter
	input.filter = filter
	return
}

func (input *PcapBuffer) Finish() {
	if input.current != nil {
		input.current.halfFull()
		input.result <- input.current
	}

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
	t, _ := input.readHandle(fhandle, filter)
	return t
}

func (input *PcapBuffer) ReadInterface(dname string) (t flows.Time) {
	inactive, err := pcap.NewInactiveHandle(dname)
	if err != nil {
		log.Fatal(err)
	}

	if err = inactive.SetTimeout(100 * time.Millisecond); err != nil {
		log.Fatal(err)
	}
	// FIXME: set other options here

	handle, err := inactive.Activate()
	if err != nil {
		log.Fatal(err)
	}

	if err = handle.SetBPFFilter(input.filter); err != nil {
		log.Fatal(err)
	}

	t, stop := input.readHandle(handle, nil)

	if !stop {
		inactive.CleanUp()
		handle.Close()
	}

	return
}

func (input *PcapBuffer) readHandle(fhandle *pcap.Handle, filter *pcap.BPF) (time flows.Time, stop bool) {
	cancel := make(chan os.Signal, 1)
	finished := make(chan interface{}, 1)
	signal.Notify(cancel, os.Interrupt)
	defer func() {
		signal.Stop(cancel)
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
	go func(time *flows.Time, stop *bool) {
		for {
			data, ci, err := fhandle.ZeroCopyReadPacketData()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Println("Error:", err)
				continue
			}
			if filter != nil && !filter.Matches(ci, data) {
				continue
			}
			if *stop {
				return
			}
			input.current.current().(*pcapPacketBuffer).assign(data, ci, lt)
			if input.current.finished() {
				input.result <- input.current
				input.current = <-input.empty
				input.current.reset()
			}
		}
		finished <- nil
	}(&time, &stop)
	select {
	case <-finished:
		stop = false
	case <-cancel:
		stop = true
	}
	return
}
