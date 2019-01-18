package flows

type exportKey struct {
	packetID   uint64
	exportTime DateTimeNanoseconds
	recordID   int
}

// exportRecord contains a list of records to export
type exportRecord struct {
	exportKey
	next     *exportRecord
	prev     *exportRecord
	template Template
	features []interface{}
	head     bool
}

func makeExportHead() *exportRecord {
	ret := &exportRecord{head: true}
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
