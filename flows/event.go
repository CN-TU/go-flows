package flows

// Event describes a top level event (e.g., a single packet)
type Event interface {
	// Timestamp returns the timestamp of the event.
	Timestamp() DateTimeNanoseconds
	// Key returns a flow key.
	Key() string
}
