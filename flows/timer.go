package flows

// Time represents a point in time stored as unix timestamp in nanoseconds.
type Time int64

/*
This adds a minute of runtime
func (t Time) String() string {
	return time.Unix(0, int64(t)).UTC().String()
} */

const (
	// NanoSeconds holds the time value of one nanosecond.
	NanoSeconds Time = 1
	// MicroSeconds holds the time value of one microsecond.
	MicroSeconds Time = 1000 * NanoSeconds
	// MilliSeconds holds the time value of one millisecond.
	MilliSeconds Time = 1000 * MicroSeconds
	// Seconds holds the time value of one second.
	Seconds Time = 1000 * MilliSeconds
	// Minutes holds the time value of one minute.
	Minutes Time = 60 * Seconds
	// Hours holds the time value of one hour.
	Hours Time = 60 * Minutes
)

// TimerID represents a single timer
type TimerID int

// TimerCallback is a function the gets called upon a timer event. This event receives the expiry time and the current time.
type TimerCallback func(Time, Time)

var timerMaxID TimerID

// RegisterTimer registers a new timer and returns the new TimerID.
func RegisterTimer() TimerID {
	ret := timerMaxID
	timerMaxID++
	return ret
}

var (
	timerIdle   = RegisterTimer()
	timerActive = RegisterTimer()
)
