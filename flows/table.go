package flows

type FlowCreator func(Event, *FlowTable, FlowKey) Flow
type FeatureCreator func(Flow) FeatureList

type FlowTable struct {
	flows         *tree
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
		flows:         newTree(),
		features:      features,
		newflow:       newflow,
		activeTimeout: activeTimeout,
		idleTimeout:   idleTimeout,
		checkpoint:    checkpoint}
}

func (tab *FlowTable) Event(event Event) {
	when := event.Timestamp()
	key := event.Key()
	keyb := key.Bytes()

	if tab.lastEvent < when {
		tab.flows.walk(func(elem Flow) {
			if when > elem.NextEvent() {
				elem.Expire(when)
			}
		})
		tab.lastEvent = when + tab.checkpoint
	}
	// event every n seconds
	elem, ok := tab.flows.get(keyb)
	if ok {
		if when > elem.NextEvent() {
			elem.Expire(when)
			ok = elem.Active()
		}
	}
	if !ok {
		elem = tab.newflow(event, tab, key)
		tab.flows.insert(keyb, elem)
	}
	elem.Event(event, when)
}

func (tab *FlowTable) Remove(entry Flow) {
	if !tab.eof {
		if !entry.Active() {
			panic("Removing already dead flow")
		}
		tab.flows.delete(entry.Key().Bytes())
	}
}

func (tab *FlowTable) EOF(now Time) {
	tab.eof = true
	tab.flows.walk(func(v Flow) {
		if now > v.NextEvent() {
			v.Expire(now)
		}
		if v.Active() {
			v.EOF(now)
		}
	})
	tab.flows = newTree()
	tab.eof = false
}
