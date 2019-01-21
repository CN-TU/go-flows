package flows

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

func (e *exportRecord) lessExport(b *exportRecord) bool {
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
