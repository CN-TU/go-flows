package packet

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/CN-TU/go-flows/flows"
	"github.com/google/gopacket"
)

// LayerTypeIPv46 holds either a raw IPv4 or raw IPv6 packet
var LayerTypeIPv46 = gopacket.RegisterLayerType(1000, gopacket.LayerTypeMetadata{Name: "IPv4 or IPv6"})

const (
	batchSize   = 1000
	fullBuffers = 10
)

// Stats holds number of packets, skipped packets, and filtered packets
type Stats struct {
	packets  uint64
	skipped  uint64
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

// Engine holds and manages buffers, sources, filters and forwards packets to the flowtable
type Engine struct {
	empty       *multiPacketBuffer
	todecode    *shallowMultiPacketBufferRing
	current     *shallowMultiPacketBuffer
	packetStats Stats
	plen        int
	flowtable   EventTable
	done        chan struct{}
	sources     Sources
	filters     Filters
	labels      Labels
}

// NewEngine initializes a new packet handling engine.
// Packets of plen size are handled (0 means automatic). Packets are read from sources, filtered with filter, and forwarded to flowtable. Labels are assigned to the packets from the labels provider.
func NewEngine(plen int, flowtable EventTable, filters Filters, sources Sources, labels Labels) *Engine {
	prealloc := plen
	if plen == 0 {
		prealloc = 4096
	}
	ret := &Engine{
		empty:     newMultiPacketBuffer(batchSize*fullBuffers, prealloc, plen == 0),
		todecode:  newShallowMultiPacketBufferRing(fullBuffers, batchSize),
		plen:      plen,
		flowtable: flowtable,
		done:      make(chan struct{}),
		sources:   sources,
		filters:   filters,
		labels:    labels,
	}

	go func() {
		defer close(ret.done)
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

// PrintStats writes the packet statistics to w
func (input *Engine) PrintStats(w io.Writer) {
	fmt.Fprintf(w,
		`Packet statistics:
	overall: %d
	skipped: %d
	filtered: %d
`, input.packetStats.packets, input.packetStats.skipped, input.packetStats.filtered)
}

// Finish submits eventual partially filled buffers, flushes the packet handling pipeline and waits for everything to finish.
func (input *Engine) Finish() {
	if !input.current.empty() {
		input.current.finalizeWritten()
	}
	input.todecode.close()
	<-input.done

	input.flowtable.Flush()
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

// Run reads all the packets from the sources and forwards those to the flowtable
func (input *Engine) Run() (time flows.DateTimeNanoseconds) {
	var npackets, nskipped, nfiltered uint64
	for {
		lt, data, ci, skipped, filtered, err := input.sources.ReadPacket()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("Error reading packet: ", err)
		}
		npackets += skipped + filtered + 1
		nskipped += skipped
		nfiltered += filtered

		if input.filters.Matches(ci, data) {
			nfiltered++
			continue
		}

		// fixme: reintroduce labels
		// label := input.label.pop()

		if input.current.empty() {
			input.empty.Pop(input.current)
		}
		buffer := input.current.read()
		time = buffer.assign(data, ci, lt, npackets, nil /* label */)
		if input.current.full() {
			input.current.finalize()
			var ok bool
			if input.current, ok = input.todecode.popEmpty(); !ok {
				break
			}
		}
	}
	input.packetStats.packets = npackets
	input.packetStats.filtered = nfiltered
	input.packetStats.skipped = nskipped
	return
}

// Stop cancels the whole process and stops packet input
func (input *Engine) Stop() {
	input.sources.Stop()
}
