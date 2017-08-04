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
	expire     []chan struct{}
	expirewg   sync.WaitGroup
	buffers    []*shallowMultiPacketBufferRing
	tmp        []*shallowMultiPacketBuffer
	wg         sync.WaitGroup
	expireTime flows.Time
	nextExpire flows.Time
}

type SingleFlowTable struct {
	table      *flows.FlowTable
	buffer     *shallowMultiPacketBufferRing
	expire     chan struct{}
	done       chan struct{}
	expireTime flows.Time
	nextExpire flows.Time
}

func (sft *SingleFlowTable) Expire() {
	sft.expire <- struct{}{}
	go runtime.GC()
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

func (sft *SingleFlowTable) EOF(now flows.Time) {
	close(sft.buffer.full)
	<-sft.done
	sft.table.EOF(now)
}

func NewParallelFlowTable(num int, features flows.FeatureListCreatorList, newflow flows.FlowCreator, options flows.FlowOptions, expire flows.Time) EventTable {
	if num == 1 {
		ret := &SingleFlowTable{
			table:      flows.NewFlowTable(features, newflow, options),
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
	ret := &ParallelFlowTable{
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
		t := flows.NewFlowTable(features, newflow, options)
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

func (pft *ParallelFlowTable) Expire() {
	for _, e := range pft.expire {
		pft.expirewg.Add(1)
		e <- struct{}{}
	}
	pft.expirewg.Wait()
	go runtime.GC()
}

func (pft *ParallelFlowTable) Event(buffer *shallowMultiPacketBuffer) {
	current := buffer.Timestamp()
	if current > pft.nextExpire {
		pft.Expire()
		pft.nextExpire = current + pft.expireTime
	}
	num := len(pft.tables)

	for i := 0; i < num; i++ {
		pft.tmp[i], _ = pft.buffers[i].popEmpty()
	}
	for {
		b := buffer.read()
		if b == nil {
			break
		}
		h := b.Key().Hash() % uint64(num)
		pft.tmp[h].push(b)
	}
	for i := 0; i < num; i++ {
		pft.tmp[i].finalize()
	}
}

func (pft *ParallelFlowTable) EOF(now flows.Time) {
	for _, c := range pft.buffers {
		close(c.full)
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
