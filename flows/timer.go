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

type funcEntry struct {
	function TimerCallback
	when     Time
}

type funcEntries []funcEntry

func makeFuncEntries() funcEntries {
	return make(funcEntries, 2)
}

func (fe *funcEntries) expire(when Time) Time {
	var next Time
	fep := *fe
	for i, v := range fep {
		if v.when != 0 {
			if v.when <= when {
				fep[i].function(v.when, when)
				fep[i].when = 0
			} else if next == 0 || v.when <= next {
				next = v.when
			}
		}
	}
	return next
}

func (fe *funcEntries) addTimer(id TimerID, f TimerCallback, when Time) {
	fep := *fe
	if int(id) >= len(fep) {
		fep = append(fep, make(funcEntries, len(fep)-int(id)+1)...)
		*fe = fep
	}
	fep[id].function = f
	fep[id].when = when
}

func (fe *funcEntries) hasTimer(id TimerID) bool {
	fep := *fe
	if int(id) >= len(fep) {
		return false
	}
	if fep[id].when == 0 {
		return false
	}
	return true
}
