package packet

import (
	"sync"
	"sync/atomic"

	"github.com/CN-TU/go-flows/flows"
)

type multiPacketBuffer struct {
	numFree int32
	buffers []*packetBuffer
	cond    *sync.Cond
}

func newMultiPacketBuffer(buffers int32, prealloc int, resize bool) *multiPacketBuffer {
	buf := &multiPacketBuffer{
		numFree: buffers,
		buffers: make([]*packetBuffer, buffers),
	}
	buf.cond = sync.NewCond(&sync.Mutex{})
	for j := range buf.buffers {
		buf.buffers[j] = &packetBuffer{buffer: make([]byte, prealloc), owner: buf, resize: resize}
	}
	return buf
}

func (mpb *multiPacketBuffer) free(num int32) {
	if atomic.AddInt32(&mpb.numFree, num) > batchSize {
		mpb.cond.Signal()
	}
}

func (mpb *multiPacketBuffer) Pop(buffer *shallowMultiPacketBuffer) {
	var num int32
	buffer.reset()
	for num == 0 { //return a buffer with at least one element
		if atomic.LoadInt32(&mpb.numFree) < batchSize {
			mpb.cond.L.Lock()
			for atomic.LoadInt32(&mpb.numFree) < batchSize {
				mpb.cond.Wait()
			}
			mpb.cond.L.Unlock()
		}

		for _, b := range mpb.buffers {
			if atomic.LoadInt32(&b.inUse) == 0 {
				if !buffer.push(b) {
					break
				}
				atomic.StoreInt32(&b.inUse, 1)
				num++
			}
		}
	}
	atomic.AddInt32(&mpb.numFree, -num)
}

type shallowMultiPacketBuffer struct {
	buffers []*packetBuffer
	owner   *shallowMultiPacketBufferRing
	rindex  int
	windex  int
}

func newShallowMultiPacketBuffer(size int, owner *shallowMultiPacketBufferRing) *shallowMultiPacketBuffer {
	return &shallowMultiPacketBuffer{
		buffers: make([]*packetBuffer, size),
		owner:   owner,
	}
}

func (smpb *shallowMultiPacketBuffer) empty() bool {
	return smpb.windex == 0
}

func (smpb *shallowMultiPacketBuffer) full() bool {
	return smpb.windex != 0 && smpb.rindex == smpb.windex
}

func (smpb *shallowMultiPacketBuffer) reset() {
	smpb.rindex = 0
	smpb.windex = 0
}

func (smpb *shallowMultiPacketBuffer) push(buffer *packetBuffer) bool {
	if smpb.windex >= len(smpb.buffers) || smpb.windex < 0 {
		return false
	}
	smpb.buffers[smpb.windex] = buffer
	smpb.windex++
	return true
}

func (smpb *shallowMultiPacketBuffer) read() (ret *packetBuffer) {
	if smpb.rindex >= len(smpb.buffers) || smpb.rindex >= smpb.windex || smpb.rindex < 0 {
		return nil
	}
	ret = smpb.buffers[smpb.rindex]
	smpb.rindex++
	return
}

func (smpb *shallowMultiPacketBuffer) finalize() {
	smpb.rindex = 0
	if smpb.owner != nil {
		smpb.owner.full <- smpb
	}
}

func (smpb *shallowMultiPacketBuffer) finalizeWritten() {
	rec := smpb.buffers[smpb.rindex:smpb.windex]
	for _, buf := range rec {
		buf.Recycle()
	}
	smpb.windex = smpb.rindex
	smpb.finalize()
}

func (smpb *shallowMultiPacketBuffer) recycleEmpty() {
	smpb.reset()
	if smpb.owner != nil {
		smpb.owner.empty <- smpb
	}
}

func (smpb *shallowMultiPacketBuffer) recycle() {
	if !smpb.empty() {
		var num int32
		mpb := smpb.buffers[0].owner
		buf := smpb.buffers[:smpb.windex]
		for i, b := range buf {
			if b.canRecycle() {
				atomic.StoreInt32(&buf[i].inUse, 0)
				num++
			}
		}
		mpb.free(num)
	}
	smpb.reset()
	if smpb.owner != nil {
		smpb.owner.empty <- smpb
	}
}

func (smpb *shallowMultiPacketBuffer) Timestamp() flows.DateTimeNanoseconds {
	if !smpb.empty() {
		return smpb.buffers[0].Timestamp()
	}
	return 0
}

func (smpb *shallowMultiPacketBuffer) Copy(other *shallowMultiPacketBuffer) {
	src := smpb.buffers[:smpb.windex]
	target := other.buffers[:len(src)]
	copy(target, src)
	other.rindex = 0
	other.windex = smpb.windex
}

type shallowMultiPacketBufferRing struct {
	empty chan *shallowMultiPacketBuffer
	full  chan *shallowMultiPacketBuffer
}

func newShallowMultiPacketBufferRing(buffers, batch int) (ret *shallowMultiPacketBufferRing) {
	ret = &shallowMultiPacketBufferRing{
		empty: make(chan *shallowMultiPacketBuffer, buffers),
		full:  make(chan *shallowMultiPacketBuffer, buffers),
	}
	for i := 0; i < buffers; i++ {
		ret.empty <- newShallowMultiPacketBuffer(batch, ret)
	}
	return
}

func (smpbr *shallowMultiPacketBufferRing) popEmpty() (ret *shallowMultiPacketBuffer, ok bool) {
	ret, ok = <-smpbr.empty
	return
}

func (smpbr *shallowMultiPacketBufferRing) popFull() (ret *shallowMultiPacketBuffer, ok bool) {
	ret, ok = <-smpbr.full
	return
}

func (smpbr *shallowMultiPacketBufferRing) close() {
	close(smpbr.full)
}
