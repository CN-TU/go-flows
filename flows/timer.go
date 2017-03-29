package flows

type Time int64

/*
This adds a minute of runtime
func (t Time) String() string {
	return time.Unix(0, int64(t)).UTC().String()
} */

const (
	NanoSeconds  Time = 1
	MicroSeconds Time = 1000 * NanoSeconds
	MilliSeconds Time = 1000 * MicroSeconds
	Seconds      Time = 1000 * MilliSeconds
	Minutes      Time = 60 * Seconds
	Hours        Time = 60 * Minutes
)

type TimerID int

type TimerCallback func(Time, Time)

var timerMaxID TimerID = 0

func RegisterTimer() TimerID {
	ret := timerMaxID
	timerMaxID++
	return ret
}

var (
	timerIdle   = RegisterTimer()
	timerActive = RegisterTimer()
)
