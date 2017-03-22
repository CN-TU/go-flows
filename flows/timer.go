package flows

type Time int64

const (
	NanoSeconds  Time = 1
	MicroSeconds      = 1e3 * NanoSeconds
	MilliSeconds      = 1e3 * MicroSeconds
	Seconds           = 1e3 * MilliSeconds
	Minutes           = 60 * Seconds
	Hours             = 60 * Minutes
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
