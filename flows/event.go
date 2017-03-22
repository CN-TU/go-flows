package flows

type Event interface {
	Timestamp() Time
	Key() FlowKey
}
