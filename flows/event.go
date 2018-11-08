package flows

// Event describes a top level event (e.g., a single packet)
type Event interface {
	// Timestamp returns the timestamp of the event.
	Timestamp() DateTimeNanoseconds
	// Key returns a flow key.
	Key() string
	// LowToHigh returns if the direction is from the lower key to the higher key for bidirectional
	LowToHigh() bool
}
