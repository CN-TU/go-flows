package flows

const (
	// NanoSecondsInNanoSeconds holds the time value of one nanosecond.
	NanoSecondsInNanoSeconds DateTimeNanoSeconds = 1
	// MicroSecondsInNanoSeconds holds the time value of one microsecond.
	MicroSecondsInNanoSeconds DateTimeNanoSeconds = 1000 * NanoSecondsInNanoSeconds
	// MilliSecondsInNanoSeconds holds the time value of one millisecond.
	MilliSecondsInNanoSeconds DateTimeNanoSeconds = 1000 * MicroSecondsInNanoSeconds
	// SecondsInNanoSeconds holds the time value of one second.
	SecondsInNanoSeconds DateTimeNanoSeconds = 1000 * MilliSecondsInNanoSeconds
	// MinutesInNanoSeconds holds the time value of one minute.
	MinutesInNanoSeconds DateTimeNanoSeconds = 60 * SecondsInNanoSeconds
	// HoursInNanoSeconds holds the time value of one hour.
	HoursInNanoSeconds DateTimeNanoSeconds = 60 * MinutesInNanoSeconds
)
