package flows

// Event describes a top level event (e.g., a single packet)
type Event interface {
	// Timestamp needs to return the timestamp of the event.
	Timestamp() Time
	// Key needs to return a flow key.
	Key() FlowKey
}
