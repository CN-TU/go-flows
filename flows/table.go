package flows

type FlowCreator func(Event, *FlowTable, FlowKey, Time) Flow
type FeatureListCreator func() *FeatureList

type FlowTable struct {
	flows         map[FlowKey]int
	flowLRU       []Flow
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
		flowLRU:       make([]Flow, 0, 1000000),
		newflow:       newflow,
		activeTimeout: activeTimeout,
		idleTimeout:   idleTimeout,
		features:      features,
	}
}

func (tab *FlowTable) Expire() {
	when := tab.now
	for _, elem := range tab.flowLRU {
		if elem == nil {
			break
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
		elem := tab.flowLRU[elem]
		if when > elem.NextEvent() {
			elem.Expire(when)
			ok = elem.Active()
		}
		elem.Event(event, when)
	}
	if !ok {
		elem := tab.newflow(event, tab, key, when)
		tab.flows[key] = len(tab.flowLRU)
		elem.setPosition(len(tab.flowLRU))
		tab.flowLRU = append(tab.flowLRU, elem)
		elem.Event(event, when)
	}
}

func (tab *FlowTable) Remove(entry Flow) {
	if !tab.eof {
		old := tab.flows[entry.Key()]
		end := tab.flowLRU[len(tab.flowLRU)-1]
		end.setPosition(old)
		tab.flows[end.Key()] = old
		tab.flowLRU[old] = end
		tab.flowLRU[len(tab.flowLRU)-1] = nil
		tab.flowLRU = tab.flowLRU[:len(tab.flowLRU)-1]
		delete(tab.flows, entry.Key())
	}
}

func (tab *FlowTable) EOF(now Time) {
	tab.eof = true
	for _, v := range tab.flowLRU {
		if now > v.NextEvent() {
			v.Expire(now)
		}
		if v.Active() {
			v.EOF(now)
		}
	}
	tab.flows = make(map[FlowKey]int)
	tab.flowLRU = nil
	tab.eof = false
}
