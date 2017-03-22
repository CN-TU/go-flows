package flows

type FlowTable struct {
	flows     map[FlowKey]Flow
	features  func(*BaseFlow) FeatureList
	eof       bool
	lastEvent Time
	newflow   func(FlowPacket, *FlowTable, FlowKey) Flow
}

func NewFlowTable(features func(*BaseFlow) FeatureList, newflow func(FlowPacket, *FlowTable, FlowKey) Flow) *FlowTable {
	return &FlowTable{flows: make(map[FlowKey]Flow, 1000000), features: features, newflow: newflow}
}

func (tab *FlowTable) Event(packet FlowPacket, key FlowKey) {
	when := Time(packet.Metadata().Timestamp.UnixNano())

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
		elem = tab.newflow(packet, tab, key)
		tab.flows[key] = elem
	}
	elem.Event(packet, when)
}

func (tab *FlowTable) Remove(key FlowKey, entry *BaseFlow) {
	if !tab.eof {
		delete(tab.flows, key)
	}
}

func (tab *FlowTable) EOF() {
	tab.eof = true
	for _, v := range tab.flows {
		// check for timeout!!
		v.EOF()
	}
	tab.flows = make(map[FlowKey]Flow)
	tab.eof = false
}
