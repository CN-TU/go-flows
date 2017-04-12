package flows

type FlowCreator func(Event, *FlowTable, FlowKey, Time) Flow

type FlowTable struct {
	flows         map[FlowKey]int
	flowlist      []Flow
	freelist      []int
	newflow       FlowCreator
	activeTimeout Time
	idleTimeout   Time
	now           Time
	features      FeatureListCreator
	eof           bool
}

func NewFlowTable(features FeatureListCreator, newflow FlowCreator, activeTimeout, idleTimeout Time) *FlowTable {
	return &FlowTable{
		flows:         make(map[FlowKey]int, 1000000),
		flowlist:      make([]Flow, 0, 1000000),
		freelist:      make([]int, 0, 1000000),
		newflow:       newflow,
		activeTimeout: activeTimeout,
		idleTimeout:   idleTimeout,
		features:      features,
	}
}

func (tab *FlowTable) Expire() {
	when := tab.now
	for _, elem := range tab.flowlist {
		if elem == nil {
			continue
		}
		if when > elem.NextEvent() {
			elem.Expire(when)
		}
	}
}

func (tab *FlowTable) Event(event Event) {
	when := event.Timestamp()
	key := event.Key()

	tab.now = when

	elem, ok := tab.flows[key]
	if ok {
		elem := tab.flowlist[elem]
		if elem != nil {
			if when > elem.NextEvent() {
				elem.Expire(when)
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

func (tab *FlowTable) Remove(entry Flow) {
	if !tab.eof {
		old := tab.flows[entry.Key()]
		tab.flowlist[old] = nil
		tab.freelist = append(tab.freelist, old)
		delete(tab.flows, entry.Key())
	}
}

func (tab *FlowTable) EOF(now Time) {
	tab.eof = true
	for _, v := range tab.flowlist {
		if v == nil {
			continue
		}
		if now > v.NextEvent() {
			v.Expire(now)
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
