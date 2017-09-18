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
	batchSize   = 1000
	fullBuffers = 10
)

type PcapBuffer struct {
	empty    *multiPacketBuffer
	todecode *shallowMultiPacketBufferRing
	current  *shallowMultiPacketBuffer
	filter   string
	plen     int
}

func NewPcapBuffer(plen int, flowtable EventTable) *PcapBuffer {
	prealloc := plen
	if plen == 0 {
		prealloc = 4096
	}
	ret := &PcapBuffer{
		empty:    newMultiPacketBuffer(batchSize*fullBuffers, prealloc, plen == 0),
		todecode: newShallowMultiPacketBufferRing(fullBuffers, batchSize),
		plen:     plen}

	go func() {
		discard := newShallowMultiPacketBuffer(batchSize, nil)
		forward := newShallowMultiPacketBuffer(batchSize, nil)
		keyFunc := flowtable.KeyFunc()
		for {
			multibuffer, ok := ret.todecode.popFull()
			if !ok {
				return
			}
			for {
				buffer := multibuffer.read()
				if buffer == nil {
					break
				}
				if !buffer.decode() {
					//count non interesting packets?
					discard.push(buffer)
				} else {
					key, fw := keyFunc(buffer)
					if key != nil {
						buffer.setInfo(key, fw)
						forward.push(buffer)
					} else {
						discard.push(buffer)
					}
				}
			}
			multibuffer.recycleEmpty()
			if !forward.empty() {
				flowtable.Event(forward)
			}
			forward.reset()
			discard.recycle()
		}
	}()

	ret.current, _ = ret.todecode.popEmpty()

	return ret
}

func (input *PcapBuffer) SetFilter(filter string) (old string) {
	old = input.filter
	input.filter = filter
	return
}

func (input *PcapBuffer) Finish() {
	if !input.current.empty() {
		input.current.finalizeWritten()
	}
	input.todecode.close()

	// consume empty buffers -> let every go routine finish
	input.empty.close()
}

func (input *PcapBuffer) ReadFile(fname string) flows.DateTimeNanoSeconds {
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

func (input *PcapBuffer) ReadInterface(dname string) (t flows.DateTimeNanoSeconds) {
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

func (input *PcapBuffer) readHandle(fhandle *pcap.Handle, filter *pcap.BPF) (time flows.DateTimeNanoSeconds, stop bool) {
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
	go func(time *flows.DateTimeNanoSeconds, stop *bool) {
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
			if input.current.empty() {
				input.empty.Pop(input.current)
			}
			buffer := input.current.read()
			*time = buffer.assign(data, ci, lt)
			if input.current.full() {
				input.current.finalize()
				var ok bool
				if input.current, ok = input.todecode.popEmpty(); !ok {
					return
				}
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
