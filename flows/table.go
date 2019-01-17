package flows

// FlowCreator is responsible for creating new flows. Supplied values are event, the flowtable, a flow key, and the current time.
type FlowCreator func(Event, *FlowTable, string, bool, *EventContext, uint64) Flow

// TableStats holds statistics for this table
type TableStats struct {
	// Packets is the number of packets processed
	Packets uint64
	// Flows is the number of flows processed
	Flows uint64
	// Maxflows is the maximum number of concurrent flows processed
	Maxflows uint64
}

// FlowTable holds flows assigned to flow keys and handles expiry, events, and flow creation.
type FlowTable struct {
	FlowOptions
	flows     map[string]int
	flowlist  []Flow
	freelist  []int
	newflow   FlowCreator
	records   RecordListMaker
	Stats     TableStats
	context   *EventContext
	flowID    uint64
	window    uint64
	id        uint8
	fivetuple bool
	eof       bool
}

// NewFlowTable returns a new flow table utilizing features, the newflow function called for unknown flows, and the active and idle timeout.
func NewFlowTable(records RecordListMaker, newflow FlowCreator, options FlowOptions, fivetuple bool, id uint8) *FlowTable {
	return &FlowTable{
		flows:       make(map[string]int),
		newflow:     newflow,
		FlowOptions: options,
		records:     records,
		fivetuple:   fivetuple,
		context:     &EventContext{},
		id:          id,
	}
}

// Expire expires all unhandled timer events. Can be called periodically to conserve memory.
func (tab *FlowTable) Expire() {
	when := tab.context.when
	for _, elem := range tab.flowlist {
		if elem == nil {
			continue
		}
		if when > elem.nextEvent() {
			elem.expire(tab.context)
		}
	}
}

// Event needs to be called for every event (e.g., a received packet). Handles flow expiry if the event belongs to a flow, flow creation, and forwarding the event to the flow.
func (tab *FlowTable) Event(event Event) {
	tab.Stats.Packets++
	when := event.Timestamp()
	key := event.Key()
	lowToHigh := event.LowToHigh()

	tab.context.when = when

	if tab.WindowExpiry {
		window := event.Window()
		if window != tab.window {
			tab.expireWindow()
			tab.window = window
		}
	}

	elem, ok := tab.flows[key]
	if ok {
		elem := tab.flowlist[elem]
		if elem != nil {
			if when > elem.nextEvent() {
				elem.expire(tab.context)
				ok = elem.Active()
			}
			if ok {
				tab.context.forward = lowToHigh == elem.firstLowToHigh()
				elem.Event(event, tab.context)
			}
		} else {
			ok = false
		}
	}
	if !ok {
		elem := tab.newflow(event, tab, key, lowToHigh, tab.context, tab.flowID)
		tab.flowID++
		tab.Stats.Flows++
		var new int
		freelen := len(tab.freelist)
		if freelen == 0 {
			new = len(tab.flowlist)
			tab.flowlist = append(tab.flowlist, elem)
		} else {
			new, tab.freelist = tab.freelist[freelen-1], tab.freelist[:freelen-1]
			tab.flowlist[new] = elem
		}
		tab.flows[key] = new
		nflows := uint64(len(tab.flows))
		if nflows > tab.Stats.Maxflows {
			tab.Stats.Maxflows = nflows
		}
		tab.context.forward = true
		elem.Event(event, tab.context)
	}
}

func (tab *FlowTable) remove(entry Flow) {
	if !tab.eof {
		old := tab.flows[entry.Key()]
		tab.flowlist[old] = nil
		tab.freelist = append(tab.freelist, old)
		delete(tab.flows, entry.Key())
	}
}

// EOF needs to be called upon end of file (e.g., program termination). All outstanding timers get expired, and the rest of the flows terminated with an eof event.
func (tab *FlowTable) EOF(now DateTimeNanoseconds) {
	tab.eof = true
	context := &EventContext{when: now}
	for _, v := range tab.flowlist {
		if v == nil {
			continue
		}
		if now > v.nextEvent() {
			v.expire(context)
		}
		if v.Active() {
			v.EOF(context)
		}
	}
	tab.flows = make(map[string]int)
	tab.flowlist = nil
	tab.freelist = nil
	tab.eof = false
}

// expireWindow expires all flows in the table due to end of the window. All outstanding timers get expired, and the rest of the flows terminated with a flowReaonEnd event.
func (tab *FlowTable) expireWindow() {
	tab.eof = true
	context := tab.context
	now := tab.context.when
	for _, v := range tab.flowlist {
		if v == nil {
			continue
		}
		if now > v.nextEvent() {
			v.expire(context)
		}
		if v.Active() {
			v.Export(FlowEndReasonEnd, context, now)
		}
	}
	for k := range tab.flows {
		delete(tab.flows, k)
	}
	for i := range tab.flowlist {
		tab.flowlist[i] = nil
	}
	tab.flowlist = tab.flowlist[:0]
	tab.freelist = tab.freelist[:0]
	tab.eof = false
}

// FiveTuple returns true if the key function is the fivetuple key
func (tab *FlowTable) FiveTuple() bool {
	return tab.fivetuple
}

// ID returns the table id
func (tab *FlowTable) ID() uint8 {
	return tab.id
}
