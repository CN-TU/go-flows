package packet

import (
	"runtime"
	"sync"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type EventTable interface {
	Event(buffer *shallowMultiPacketBuffer)
	Expire()
	EOF(flows.Time)
}

type ParallelFlowTable struct {
	tables     []*flows.FlowTable
	full       []chan *shallowMultiPacketBuffer
	expire     []chan struct{}
	expirewg   sync.WaitGroup
	empty      []chan *shallowMultiPacketBuffer
	tmp        []*shallowMultiPacketBuffer
	wg         sync.WaitGroup
	expireTime flows.Time
	nextExpire flows.Time
}

type SingleFlowTable struct {
	table      *flows.FlowTable
	full       chan *shallowMultiPacketBuffer
	expire     chan struct{}
	empty      chan *shallowMultiPacketBuffer
	done       chan struct{}
	expireTime flows.Time
	nextExpire flows.Time
}

func (sft *SingleFlowTable) Expire() {
	sft.expire <- struct{}{}
	go runtime.GC()
}

func (sft *SingleFlowTable) Event(buffer *shallowMultiPacketBuffer) {
	current := buffer.buffers[0].Timestamp()
	if current > sft.nextExpire {
		sft.Expire()
		sft.nextExpire = current + sft.expireTime
	}
	tmp := <-sft.empty
	tmp.copy(buffer)
	sft.full <- tmp
}

func (sft *SingleFlowTable) EOF(now flows.Time) {
	close(sft.full)
	<-sft.done
	sft.table.EOF(now)
}

func NewParallelFlowTable(num int, features flows.FeatureListCreator, newflow flows.FlowCreator, activeTimeout, idleTimeout, expire flows.Time) EventTable {
	if num == 1 {
		ret := &SingleFlowTable{
			table:      flows.NewFlowTable(features, newflow, activeTimeout, idleTimeout),
			expireTime: expire,
		}
		ret.full = make(chan *shallowMultiPacketBuffer, shallowBuffers)
		ret.expire = make(chan struct{}, 1)
		ret.empty = make(chan *shallowMultiPacketBuffer, shallowBuffers)
		for i := 0; i < shallowBuffers; i++ {
			ret.empty <- newShallowMultiPacketBuffer(batchSize)
		}
		ret.done = make(chan struct{})
		go func() {
			t := ret.table
			defer close(ret.done)
			for {
				select {
				case <-ret.expire:
					t.Expire()
				case buffer, ok := <-ret.full:
					if !ok {
						return
					}
					for _, b := range buffer.buffers {
						t.Event(b)
					}
					buffer.recycle()
					ret.empty <- buffer
				}
			}
		}()
		return ret
	}
	ret := &ParallelFlowTable{
		tables:     make([]*flows.FlowTable, num),
		full:       make([]chan *shallowMultiPacketBuffer, num),
		expire:     make([]chan struct{}, num),
		empty:      make([]chan *shallowMultiPacketBuffer, num),
		tmp:        make([]*shallowMultiPacketBuffer, num),
		expireTime: expire,
	}
	for i := 0; i < num; i++ {
		c := make(chan *shallowMultiPacketBuffer, shallowBuffers)
		expire := make(chan struct{}, 1)
		e := make(chan *shallowMultiPacketBuffer, shallowBuffers)
		ret.full[i] = c
		ret.expire[i] = expire
		ret.empty[i] = e
		for j := 0; j < shallowBuffers; j++ {
			ret.empty[i] <- newShallowMultiPacketBuffer(batchSize)
		}
		t := flows.NewFlowTable(features, newflow, activeTimeout, idleTimeout)
		ret.tables[i] = t
		ret.wg.Add(1)
		go func() {
			defer ret.wg.Done()
			for {
				select {
				case <-expire:
					t.Expire()
					ret.expirewg.Done()
				case buffer, ok := <-c:
					if !ok {
						return
					}
					for _, b := range buffer.buffers {
						t.Event(b)
					}
					buffer.recycle()
					e <- buffer
				}
			}
		}()
	}
	return ret
}

func (pft *ParallelFlowTable) Expire() {
	for _, e := range pft.expire {
		pft.expirewg.Add(1)
		e <- struct{}{}
	}
	pft.expirewg.Wait()
	go runtime.GC()
}

func (pft *ParallelFlowTable) Event(buffer *shallowMultiPacketBuffer) {
	current := buffer.buffers[0].Timestamp()
	if current > pft.nextExpire {
		pft.Expire()
		pft.nextExpire = current + pft.expireTime
	}
	num := len(pft.tables)
	for i := 0; i < num; i++ {
		pft.tmp[i] = <-pft.empty[i]
		pft.tmp[i].multiBuffer = buffer.multiBuffer
	}
	for _, packet := range buffer.buffers {
		h := packet.key.Hash() % uint64(num)
		pft.tmp[h].add(packet)
	}
	for i := 0; i < num; i++ {
		pft.full[i] <- pft.tmp[i]
	}
}

func (pft *ParallelFlowTable) EOF(now flows.Time) {
	for _, c := range pft.full {
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
