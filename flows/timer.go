package flows

type Time int64

const (
	NanoSeconds  Time = 1
	MicroSeconds Time = 1000 * NanoSeconds
	MilliSeconds Time = 1000 * MicroSeconds
	Seconds      Time = 1000 * MilliSeconds
	Minutes      Time = 60 * Seconds
	Hours        Time = 60 * Minutes
)

type TimerID int

const (
	TimerIdle TimerID = iota
	TimerActive
)

type TimerCallback func(Time)

type funcEntry struct {
	function TimerCallback
	when     Time
	id       TimerID
}

type funcEntries []*funcEntry

func (s funcEntries) Len() int           { return len(s) }
func (s funcEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s funcEntries) Less(i, j int) bool { return s[i].when < s[j].when }
