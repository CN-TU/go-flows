package flows

const (
	// NanosecondsInNanoseconds holds the time value of one nanosecond.
	NanosecondsInNanoseconds DateTimeNanoseconds = 1
	// MicrosecondsInNanoseconds holds the time value of one microsecond.
	MicrosecondsInNanoseconds DateTimeNanoseconds = 1000 * NanosecondsInNanoseconds
	// MillisecondsInNanoseconds holds the time value of one millisecond.
	MillisecondsInNanoseconds DateTimeNanoseconds = 1000 * MicrosecondsInNanoseconds
	// secondsInNanoseconds holds the time value of one second.
	SecondsInNanoseconds DateTimeNanoseconds = 1000 * MillisecondsInNanoseconds
	// MinutesInNanoseconds holds the time value of one minute.
	MinutesInNanoseconds DateTimeNanoseconds = 60 * SecondsInNanoseconds
	// HoursInNanoseconds holds the time value of one hour.
	HoursInNanoseconds DateTimeNanoseconds = 60 * MinutesInNanoseconds
)
