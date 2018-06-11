package packet

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/CN-TU/go-flows/flows"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var layerTypeIPv46 = gopacket.RegisterLayerType(1000, gopacket.LayerTypeMetadata{Name: "IPv4 or IPv6"})

const (
	batchSize   = 1000
	fullBuffers = 10
)

type PacketStats struct {
	packets  uint64
	filtered uint64
}

type labelProvider struct {
	labels     []string
	file       io.Closer
	csv        *csv.Reader
	nextData   interface{}
	currentPos int
	nextPos    int
}

type PcapBuffer struct {
	empty       *multiPacketBuffer
	todecode    *shallowMultiPacketBufferRing
	current     *shallowMultiPacketBuffer
	packetStats PacketStats
	label       *labelProvider
	filter      string
	plen        int
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
		stats := flowtable.DecodeStats()
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
					stats.decodeError++
					discard.push(buffer)
				} else {
					key, fw := keyFunc(buffer)
					if key != nil {
						buffer.setInfo(key, fw)
						forward.push(buffer)
					} else {
						stats.keyError++
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

func (input *PcapBuffer) PrintStats(w io.Writer) {
	fmt.Fprintf(w,
		`Packet statistics:
	overall: %d
	filtered: %d
`, input.packetStats.packets, input.packetStats.filtered)
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

func (input *PcapBuffer) SetLabel(fnames []string) {
	input.label = newLabelProvider(fnames)
}

func (input *PcapBuffer) ReadFile(fname string) flows.DateTimeNanoseconds {
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

func (input *PcapBuffer) ReadInterface(dname string) (t flows.DateTimeNanoseconds) {
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

func newLabelProvider(fnames []string) *labelProvider {
	return &labelProvider{labels: fnames}
}

func (label *labelProvider) open() {
	if label.csv != nil {
		label.close()
	}
	if len(label.labels) == 0 {
		return
	}
	var f string
	f, label.labels = label.labels[0], label.labels[1:]
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	label.file = r
	label.csv = csv.NewReader(r)
	_, err = label.csv.Read() // Read title line
	if err != nil {
		panic(err)
	}
}

func (label *labelProvider) close() {
	if label.csv != nil {
		label.csv = nil
		label.file.Close()
	}
}

func (label *labelProvider) pop() interface{} {
	if label == nil {
		return nil
	}
	label.currentPos++
	if label.nextPos == label.currentPos {
		return label.nextData
	}
	if label.nextPos > label.currentPos {
		return nil
	}
	if label.csv == nil {
		if len(label.labels) == 0 {
			return nil
		}
		label.open()
	}
	record, err := label.csv.Read()
	if err == io.EOF {
		label.open()
		if label.csv == nil {
			return nil
		}
		record, err = label.csv.Read()
	}
	if record == nil && err != nil {
		panic(err)
	}
	if len(record) == 1 {
		return record
	}
	label.nextPos, err = strconv.Atoi(record[0])
	if err != nil {
		panic(err)
	}
	if label.nextPos <= 0 {
		panic("Label packet position must be >= 0")
	}
	if label.nextPos == label.currentPos {
		return record[1:]
	}
	label.nextData = record[1:]
	return nil
}

func (input *PcapBuffer) readHandle(fhandle *pcap.Handle, filter *pcap.BPF) (time flows.DateTimeNanoseconds, stop bool) {
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
	case layers.LinkTypeLinuxSLL:
		lt = layers.LayerTypeLinuxSLL

	default:
		log.Fatalf("File format not implemented")
	}
	go func(time *flows.DateTimeNanoseconds, stop *bool) {
		npackets := input.packetStats.packets - 1
		nfiltered := input.packetStats.filtered
		defer func() {
			input.packetStats.packets = npackets + 1
			input.packetStats.filtered = nfiltered
		}()
		for {
			data, ci, err := fhandle.ZeroCopyReadPacketData()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Println("Read error in pcap file:", err)
				break
			}
			npackets++
			label := input.label.pop()
			if filter != nil && !filter.Matches(ci, data) {
				nfiltered++
				continue
			}
			if *stop {
				return
			}
			if input.current.empty() {
				input.empty.Pop(input.current)
			}
			buffer := input.current.read()
			*time = buffer.assign(data, ci, lt, npackets, label)
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
