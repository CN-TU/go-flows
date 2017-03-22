package flows

type FlowTable struct {
	flows     map[FlowKey]Flow
	features  func(*BaseFlow) FeatureList
	eof       bool
	lastEvent Time
	newflow   func(Event, *FlowTable, FlowKey) Flow
}

func NewFlowTable(features func(*BaseFlow) FeatureList, newflow func(Event, *FlowTable, FlowKey) Flow) *FlowTable {
	return &FlowTable{flows: make(map[FlowKey]Flow, 1000000), features: features, newflow: newflow}
}

func (tab *FlowTable) Event(event Event) {
	when := event.Timestamp()
	key := event.Key()

	if tab.lastEvent < when {
		for _, elem := range tab.flows {
			if when > elem.NextEvent() {
				elem.Expire(when)
			}
		}
		tab.lastEvent = when + 100*Seconds
	}
	// event every n seconds
	elem, ok := tab.flows[key]
	if ok {
		if when > elem.NextEvent() {
			elem.Expire(when)
			ok = elem.Active()
		}
	}
	if !ok {
		elem = tab.newflow(event, tab, key)
		tab.flows[key] = elem
	}
	elem.Event(event, when)
}

func (tab *FlowTable) Remove(key FlowKey, entry *BaseFlow) {
	if !tab.eof {
		delete(tab.flows, key)
	}
}

func (tab *FlowTable) EOF(now Time) {
	tab.eof = true
	for _, v := range tab.flows {
		// check for timeout!!
		v.EOF(now)
	}
	tab.flows = make(map[FlowKey]Flow)
	tab.eof = false
}
