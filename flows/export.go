package flows

/*
Output sorting supports 4 different modes:
	- none
	- start
	- stop
	- expiry

The following diagrams use the abbreviations:
	r: record
	p: ExportPipeline
	e: Exporter
	t: FlowTable
	(): function
	n,m: table ids
	a,b: exporter ids
	x,y: recordList ids
	channel otherwise

none:
	r.Export() creates an *exportRecord and directly pushes it into the output channel, which is multiplexed unto the export channels
	next and prev of the *exportRecord are always nil.

	tab[n]: r.Export() -> p.export() -> p.sorted -> p.exportSplicer() -> p.out[a] -> exportWorker() -> e.Export()
	tab[m]: r.Export() -> p.export() /                                \> p.out[b] -> exportWorker() -> e.Export()
	....

start/stop/expiry:
	t.exports[x] is the head of a doubly linked list (empty: prev, next point to t.exports[x])
	record.start() creates an *exportRecord, fills in packetID, recordID and pushes it into the front of t.exports[x]
	stop, expiry:
		record.filteredEvent() updates packetID and moves the exportRecord to the front
	record.Export() fills out features, template, exportTime, expiryTime and removes *exportRecord from the record
		expiry only:
			exportRecord is moved to the front of t.exports[x]

	after every event; expire* (start/stop only):
		*exportRecords that contain features are popped from the back of the list and forwarded to the per-table-queue
	after expire* (expiry only):
		all *exportRecords containing features are forwarded to the per-table-queue

	*exportRecords moved to per-table-queue (and the following queues) are single linked lists, where prev of the first element points to the tail

	per-table-queue (expiry only):
		received linked lists are sorted with natural merge sort

	per-table-queue (start/stop) or sorted per-table-queues (expiry):
		these sorted lists are merged in sortorder unto p.sorted, which is then multiplexed to the exporters

	less function for sorting compares the following fields in order:
		start/stop: packetID, recordID
		expiry: expiryTime, packetID, recordID

	if a flow is removed it calles r.Destroy(), which unlinks the *exportRecord in case it is still there (= was never exported)


	start/stop:
	tab[n]: t.flush*() -> p.in[n] -> mergeTreeWorker() -> p.sorted -> p.exportSplicer() -> p.out[a] -> exportWorker() -> e.Export()
	tab[m]: t.flush*() -> p.in[m] /                                                     \> p.out[b] -> exportWorker() -> e.Export()
	....

	expiry:
	tab[n]: t.flush*() -> p.in[n] -> sortWorker() -> unmerged[n] -> mergeTreeWorker() -> p.sorted -> p.exportSplicer() -> p.out[a] -> exportWorker() -> e.Export()
	tab[m]: t.flush*() -> p.in[m] -> sortWorker() -> unmerged[m] /                                                     \> p.out[b] -> exportWorker() -> e.Export()
	....
*/

type exportKey struct {
	packetID   uint64
	expiryTime DateTimeNanoseconds
	recordID   int
}

// exportRecord contains a list of records to export
type exportRecord struct {
	exportKey
	exportTime DateTimeNanoseconds
	next       *exportRecord
	prev       *exportRecord
	template   Template
	features   []interface{}
}

func (e *exportRecord) lessPacket(b *exportRecord) bool {
	if e.packetID == b.packetID {
		if e.recordID < b.recordID {
			return true
		}
		return false
	}
	if e.packetID < b.packetID {
		return true
	}
	return false
}

func (e *exportRecord) lessExpiry(b *exportRecord) bool {
	if e.expiryTime == b.expiryTime {
		if e.packetID == b.packetID {
			if e.recordID < b.recordID {
				return true
			}
			return false
		}
		if e.packetID < b.packetID {
			return true
		}
		return false
	}
	if e.expiryTime < b.expiryTime {
		return true
	}
	return false
}

func makeExportHead() *exportRecord {
	ret := &exportRecord{}
	ret.next = ret
	ret.prev = ret
	return ret
}

func (e *exportRecord) unlink() {
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	e.prev = nil
	e.next = nil
}

func (e *exportRecord) insert(after *exportRecord) {
	next := after.next

	after.next = e
	e.prev = after

	next.prev = e
	e.next = next
}

func (e *exportRecord) exported() bool {
	return len(e.features) != 0
}
