package flows

type FlowCreator func(Event, *FlowTable, FlowKey) Flow
type FeatureCreator func(Flow) FeatureList

type FlowTable struct {
	flows         map[FlowKey]Flow
	features      FeatureCreator
	lastEvent     Time
	newflow       FlowCreator
	activeTimeout Time
	idleTimeout   Time
	checkpoint    Time
	eof           bool
}

func NewFlowTable(features FeatureCreator, newflow FlowCreator, activeTimeout, idleTimeout, checkpoint Time) *FlowTable {
	return &FlowTable{
		flows:         make(map[FlowKey]Flow, 1000000),
		features:      features,
		newflow:       newflow,
		activeTimeout: activeTimeout,
		idleTimeout:   idleTimeout,
		checkpoint:    checkpoint}
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
		tab.lastEvent = when + tab.checkpoint
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

func (tab *FlowTable) Remove(entry Flow) {
	if !tab.eof {
		delete(tab.flows, entry.Key())
	}
}

func (tab *FlowTable) EOF(now Time) {
	tab.eof = true
	for _, v := range tab.flows {
		if now > v.NextEvent() {
			v.Expire(now)
		}
		if v.Active() {
			v.EOF(now)
		}
	}
	tab.flows = make(map[FlowKey]Flow)
	tab.eof = false
}
