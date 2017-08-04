package flows

// FlowCreator is responsible for creating new flows. Supplied values are event, the flowtable, a flow key, and the current time.
type FlowCreator func(Event, *FlowTable, FlowKey, Time) Flow

// FlowTable holds flows assigned to flow keys and handles expiry, events, and flow creation.
type FlowTable struct {
	FlowOptions
	flows    map[FlowKey]int
	flowlist []Flow
	freelist []int
	newflow  FlowCreator
	now      Time
	features FeatureListCreatorList
	eof      bool
}

// NewFlowTable returns a new flow table utilizing features, the newflow function called for unknown flows, and the active and idle timeout.
func NewFlowTable(features FeatureListCreatorList, newflow FlowCreator, options FlowOptions) *FlowTable {
	return &FlowTable{
		flows:       make(map[FlowKey]int, 1000000),
		flowlist:    make([]Flow, 0, 1000000),
		freelist:    make([]int, 0, 1000000),
		newflow:     newflow,
		FlowOptions: options,
		features:    features,
	}
}

// Expire expires all unhandled timer events. Can be called periodically to conserve memory.
func (tab *FlowTable) Expire() {
	when := tab.now
	for _, elem := range tab.flowlist {
		if elem == nil {
			continue
		}
		if when > elem.nextEvent() {
			elem.expire(when)
		}
	}
}

// Event needs to be called for every event (e.g., a received packet). Handles flow expiry if the event belongs to a flow, flow creation, and forwarding the event to the flow.
func (tab *FlowTable) Event(event Event) {
	when := event.Timestamp()
	key := event.Key()

	tab.now = when

	elem, ok := tab.flows[key]
	if ok {
		elem := tab.flowlist[elem]
		if elem != nil {
			if when > elem.nextEvent() {
				elem.expire(when)
				ok = elem.Active()
			}
			elem.Event(event, when)
		} else {
			ok = false
		}
	}
	if !ok {
		elem := tab.newflow(event, tab, key, when)
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
		elem.Event(event, when)
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
func (tab *FlowTable) EOF(now Time) {
	tab.eof = true
	for _, v := range tab.flowlist {
		if v == nil {
			continue
		}
		if now > v.nextEvent() {
			v.expire(now)
		}
		if v.Active() {
			v.EOF(now)
		}
	}
	tab.flows = make(map[FlowKey]int)
	tab.flowlist = nil
	tab.freelist = nil
	tab.eof = false
}
