package flows

// Event describes a top level event (e.g., a single packet)
type Event interface {
	// Timestamp returns the timestamp of the event.
	Timestamp() DateTimeNanoseconds
	// Key returns a flow key.
	Key() string
	// LowToHigh returns if the direction is from the lower key to the higher key for bidirectional
	LowToHigh() bool
	// SetWindow sets the window id. Must be unique for every window, and packets belong to the same window must be consecutive.
	SetWindow(uint64)
	// Window returns the window id
	Window() uint64
	// EventNr returns the number of the event
	EventNr() uint64
}
