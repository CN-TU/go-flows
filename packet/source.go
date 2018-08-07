package packet

import (
	"fmt"
	"io"
	"log"

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
		stats := flowtable.getDecodeStats()
		labels := ret.labels
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
					buffer.label = labels.GetLabel(buffer)
					key, fw := flowtable.key(buffer)
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
				flowtable.event(forward)
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

	input.flowtable.flush()
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

		if input.current.empty() {
			input.empty.Pop(input.current)
		}
		buffer := input.current.read()
		time = buffer.assign(data, ci, lt, npackets)
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
