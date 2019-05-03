package packet

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/CN-TU/go-flows/flows"
	"github.com/google/gopacket"
)

// LayerTypeIPv46 holds either a raw IPv4 or raw IPv6 packet
var LayerTypeIPv46 = gopacket.RegisterLayerType(1000, gopacket.LayerTypeMetadata{Name: "IPv4 or IPv6"})

// ErrTimeout should be returned by sources, if no packet has been observed for some timeout (e.g., 1 second).
// In this case ci MUST hold the current timestamp
var ErrTimeout = errors.New("Timeout")

const (
	batchSize    = 1000
	fullBuffers  = 5
	releaseMark  = 10
	freeMark     = 3000
	lowMark      = 1
	highMark     = 2
	debugBuffers = false
)

// Stats holds number of packets, skipped packets, and filtered packets
type Stats struct {
	packets          uint64
	skipped          uint64
	filtered         uint64
	maxBuffers       int
	buffersAllocated int
	buffersReleased  int
}

// Engine holds and manages buffers, sources, filters and forwards packets to the flowtable
type Engine struct {
	empty       *multiPacketBuffer
	todecode    *shallowMultiPacketBufferRing
	current     *shallowMultiPacketBuffer
	packetStats Stats
	full        int
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
		prealloc = 1500
	}
	ret := &Engine{
		empty:     newMultiPacketBuffer(batchSize, prealloc, plen == 0),
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
		selector := flowtable.getSelector()
		labels := ret.labels
		for {
			multibuffer, ok := ret.todecode.popFull()
			if !ok {
				return
			}
			forward.setTimestamp(multibuffer.Timestamp())
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
					key, fw, ok := selector.Key(buffer)
					if ok {
						buffer.SetInfo(key, fw)
						forward.push(buffer)
					} else {
						stats.keyError++
						discard.push(buffer)
					}
				}
			}
			multibuffer.recycleEmpty()
			flowtable.event(forward)
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
Buffer statistics:
	peak: %d
	allocated: %d
	freed: %d
`, input.packetStats.packets, input.packetStats.skipped, input.packetStats.filtered, input.packetStats.maxBuffers, input.packetStats.buffersAllocated, input.packetStats.buffersReleased)
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

func (input *Engine) starved(have int, max int) {
	todecode := input.todecode.usage()
	table := input.flowtable.usage()
	alloc := false
	stop := false
	if todecode.buffers < lowMark {
		alloc = true
	}
	if todecode.buffers > highMark {
		stop = true
	}
	for _, usage := range table {
		if usage.buffers < lowMark {
			alloc = true
		}
		if usage.buffers > highMark {
			stop = true
		}
	}
	if stop {
		alloc = false
	}
	if debugBuffers {
		fmt.Println("too small ", have, max, todecode.buffers, todecode.packets, table, alloc)
	}
	if alloc {
		input.packetStats.buffersAllocated += batchSize
		input.empty.replenish()
	}
}

func (input *Engine) ok(have int, max int) {
	if max > input.packetStats.maxBuffers {
		input.packetStats.maxBuffers = max
	}
	if have > freeMark {
		input.full++
		if debugBuffers {
			todecode := input.todecode.usage()
			table := input.flowtable.usage()
			fmt.Println("     high ", have, max, todecode.buffers, todecode.packets, table, input.full > releaseMark)
		}
		if input.full > releaseMark {
			input.packetStats.buffersReleased += batchSize
			input.empty.release()
			input.full = 0
		}
	} else {
		input.full = 0
	}
}

// Run reads all the packets from the sources and forwards those to the flowtable
func (input *Engine) Run() (time flows.DateTimeNanoseconds) {
	var npackets, nskipped, nfiltered uint64
	var lastTime flows.DateTimeNanoseconds
	warned := false

	input.sources.Init()

	for {
		lt, data, ci, skipped, filtered, err := input.sources.ReadPacket()
		if err != nil {
			if err == ErrTimeout {
				input.current.setTimestamp(flows.DateTimeNanoseconds(ci.Timestamp.UnixNano()))
				input.current.finalizeWritten()
				var ok bool
				if input.current, ok = input.todecode.popEmpty(); !ok {
					break
				}
				continue
			}
			if err == io.EOF {
				break
			}
			log.Fatal("Error reading packet: ", err)
		}
		npackets += skipped + filtered + 1
		nskipped += skipped
		nfiltered += filtered

		if input.filters.Matches(lt, data, ci, npackets) {
			nfiltered++
			continue
		}

		if input.current.empty() {
			input.empty.Pop(input.current, input.starved, input.ok)
		}
		buffer := input.current.read()
		time = buffer.assign(data, ci, lt, npackets)
		if !warned && time < lastTime {
			log.Printf("Warning: Jump back in time (from %d to %d)\n", lastTime, time)
			warned = true
		}
		lastTime = time
		if input.current.full() {
			input.current.setTimestamp(time)
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
