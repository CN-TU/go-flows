package packet

import (
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/CN-TU/go-flows/flows"
)

type DecodeStats struct {
	decodeError uint64
	keyError    uint64
}

type EventTable interface {
	Event(buffer *shallowMultiPacketBuffer)
	Expire()
	Flush()
	EOF(flows.DateTimeNanoseconds)
	Key(Buffer) (flows.FlowKey, bool)
	DecodeStats() *DecodeStats
	PrintStats(io.Writer)
}

type baseTable struct {
	key       func(Buffer, bool) (flows.FlowKey, bool)
	allowZero bool
	autoGC    bool
}

func (bt baseTable) Key(pb Buffer) (flows.FlowKey, bool) {
	return bt.key(pb, bt.allowZero)
}

type ParallelFlowTable struct {
	baseTable
	tables      []*flows.FlowTable
	expire      []chan struct{}
	expirewg    sync.WaitGroup
	buffers     []*shallowMultiPacketBufferRing
	tmp         []*shallowMultiPacketBuffer
	wg          sync.WaitGroup
	decodeStats DecodeStats
	expireTime  flows.DateTimeNanoseconds
	nextExpire  flows.DateTimeNanoseconds
}

type SingleFlowTable struct {
	baseTable
	table       *flows.FlowTable
	buffer      *shallowMultiPacketBufferRing
	expire      chan struct{}
	done        chan struct{}
	decodeStats DecodeStats
	expireTime  flows.DateTimeNanoseconds
	nextExpire  flows.DateTimeNanoseconds
}

func (sft *SingleFlowTable) PrintStats(w io.Writer) {
	fmt.Fprintf(w,
		`Decode statistics:
	decode errors: %d
	key function rejects: %d
`, sft.decodeStats.decodeError, sft.decodeStats.keyError)
	fmt.Fprintf(w,
		`Table statistics:
	flows: %d
	peak flows: %d
`, sft.table.Stats.Flows, sft.table.Stats.Maxflows)
}

func (sft *SingleFlowTable) DecodeStats() *DecodeStats {
	return &sft.decodeStats
}

func (sft *SingleFlowTable) Expire() {
	sft.expire <- struct{}{}
	if !sft.autoGC {
		go runtime.GC()
	}
}

func (sft *SingleFlowTable) Event(buffer *shallowMultiPacketBuffer) {
	current := buffer.Timestamp()
	if current > sft.nextExpire {
		sft.Expire()
		sft.nextExpire = current + sft.expireTime
	}
	b, _ := sft.buffer.popEmpty()
	buffer.Copy(b)
	b.finalize()
}

func (sft *SingleFlowTable) Flush() {
	close(sft.buffer.full)
	<-sft.done
}

func (sft *SingleFlowTable) EOF(now flows.DateTimeNanoseconds) {
	sft.table.EOF(now)
}

func NewParallelFlowTable(num int, features flows.RecordListMaker, newflow flows.FlowCreator, options flows.FlowOptions, expire flows.DateTimeNanoseconds, selector DynamicKeySelector, allowZero bool, autoGC bool) EventTable {
	bt := baseTable{
		allowZero: allowZero,
		autoGC:    autoGC,
	}
	switch {
	case selector.fivetuple:
		bt.key = fivetuple
	case selector.empty:
		bt.key = makeEmptyKey
	default:
		bt.key = selector.makeDynamicKey
	}
	if num == 1 {
		ret := &SingleFlowTable{
			baseTable:  bt,
			table:      flows.NewFlowTable(features, newflow, options, selector.fivetuple, 0),
			expireTime: expire,
		}
		ret.buffer = newShallowMultiPacketBufferRing(fullBuffers, batchSize)
		ret.expire = make(chan struct{}, 1)
		ret.done = make(chan struct{})
		go func() {
			t := ret.table
			defer close(ret.done)
			for {
				select {
				case <-ret.expire:
					t.Expire()
				case buffer, ok := <-ret.buffer.full:
					if !ok {
						return
					}
					for {
						b := buffer.read()
						if b == nil {
							break
						}
						t.Event(b)
					}
					buffer.recycle()
				}
			}
		}()
		return ret
	}
	if num > 256 {
		panic("Maximum of 256 tables allowed")
	}
	ret := &ParallelFlowTable{
		baseTable:  bt,
		tables:     make([]*flows.FlowTable, num),
		buffers:    make([]*shallowMultiPacketBufferRing, num),
		tmp:        make([]*shallowMultiPacketBuffer, num),
		expire:     make([]chan struct{}, num),
		expireTime: expire,
	}
	for i := 0; i < num; i++ {
		c := newShallowMultiPacketBufferRing(fullBuffers, batchSize)
		expire := make(chan struct{}, 1)
		ret.expire[i] = expire
		ret.buffers[i] = c
		t := flows.NewFlowTable(features, newflow, options, selector.fivetuple, uint8(i))
		ret.tables[i] = t
		ret.wg.Add(1)
		go func() {
			defer ret.wg.Done()
			for {
				select {
				case <-expire:
					t.Expire()
					ret.expirewg.Done()
				case buffer, ok := <-c.full:
					if !ok {
						return
					}
					for {
						b := buffer.read()
						if b == nil {
							break
						}
						t.Event(b)
					}
					buffer.recycle()
				}
			}
		}()
	}
	return ret
}

func (pft *ParallelFlowTable) PrintStats(w io.Writer) {
	fmt.Fprintf(w,
		`Decode statistics:
	decode errors: %d
	key function rejects: %d
`, pft.decodeStats.decodeError, pft.decodeStats.keyError)
	fmt.Fprintln(w, "Table statistics:")
	var sumPackets, sumFlows uint64
	for _, table := range pft.tables {
		sumPackets += table.Stats.Packets
		sumFlows += table.Stats.Flows
	}
	for i, table := range pft.tables {
		fmt.Fprintf(w, `	Table #%d:
		packets: %d (%2.2f)
		flows: %d (%2.2f)
		peak flows: %d
`, i+1, table.Stats.Packets, float64(table.Stats.Packets)/float64(sumPackets)*100, table.Stats.Flows, float64(table.Stats.Flows)/float64(sumFlows)*100, table.Stats.Maxflows)
	}
}

func (pft *ParallelFlowTable) DecodeStats() *DecodeStats {
	return &pft.decodeStats
}

func (pft *ParallelFlowTable) Expire() {
	for _, e := range pft.expire {
		pft.expirewg.Add(1)
		e <- struct{}{}
	}
	pft.expirewg.Wait()
	if !pft.autoGC {
		go runtime.GC()
	}
}

func (pft *ParallelFlowTable) Event(buffer *shallowMultiPacketBuffer) {
	current := buffer.Timestamp()
	if current > pft.nextExpire {
		pft.Expire()
		pft.nextExpire = current + pft.expireTime
	}

	tmp := pft.tmp[:len(pft.buffers)]
	for i, buf := range pft.buffers {
		tmp[i], _ = buf.popEmpty()
	}
	for {
		b := buffer.read()
		if b == nil {
			break
		}
		h := b.Key().Hash() % uint64(len(tmp))
		tmp[h].push(b)
	}
	for _, buf := range pft.tmp {
		buf.finalize()
	}
}

func (pft *ParallelFlowTable) Flush() {
	for _, c := range pft.buffers {
		close(c.full)
	}
	pft.wg.Wait()
}

func (pft *ParallelFlowTable) EOF(now flows.DateTimeNanoseconds) {
	for _, t := range pft.tables {
		pft.wg.Add(1)
		go func(table *flows.FlowTable) {
			defer pft.wg.Done()
			table.EOF(now)
		}(t)
	}
	pft.wg.Wait()
}
