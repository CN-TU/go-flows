package packet

import (
	"runtime"
	"sync"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type EventTable interface {
	Event(buffer PacketBuffer)
	Expire()
	EOF(flows.Time)
}

type ParallelFlowTable struct {
	tables     []*flows.FlowTable
	full       []chan PacketBuffer
	expire     []chan struct{}
	expirewg   sync.WaitGroup
	wg         sync.WaitGroup
	expireTime flows.Time
	nextExpire flows.Time
}

type SingleFlowTable struct {
	table      *flows.FlowTable
	full       chan PacketBuffer
	expire     chan struct{}
	done       chan struct{}
	expireTime flows.Time
	nextExpire flows.Time
}

func (sft *SingleFlowTable) Expire() {
	sft.expire <- struct{}{}
	go runtime.GC()
}

func (sft *SingleFlowTable) Event(buffer PacketBuffer) {
	current := buffer.Timestamp()
	if current > sft.nextExpire {
		sft.Expire()
		sft.nextExpire = current + sft.expireTime
	}
	sft.full <- buffer
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
		ret.full = make(chan PacketBuffer, shallowBuffers)
		ret.expire = make(chan struct{}, 1)
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
					t.Event(buffer)
					buffer.recycle()
				}
			}
		}()
		return ret
	}
	ret := &ParallelFlowTable{
		tables:     make([]*flows.FlowTable, num),
		full:       make([]chan PacketBuffer, num),
		expire:     make([]chan struct{}, num),
		expireTime: expire,
	}
	for i := 0; i < num; i++ {
		c := make(chan PacketBuffer, shallowBuffers)
		expire := make(chan struct{}, 1)
		ret.full[i] = c
		ret.expire[i] = expire
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
					t.Event(buffer)
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

func (pft *ParallelFlowTable) Event(buffer PacketBuffer) {
	current := buffer.Timestamp()
	if current > pft.nextExpire {
		pft.Expire()
		pft.nextExpire = current + pft.expireTime
	}
	num := len(pft.tables)
	h := buffer.Key().Hash() % uint64(num)
	pft.full[h] <- buffer
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
