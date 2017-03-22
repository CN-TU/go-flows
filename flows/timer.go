package flows

type TimerID int

const (
	TimerIdle TimerID = iota
	TimerActive
)

type TimerCallback func(int64)

type funcEntry struct {
	function TimerCallback
	when     int64
	id       TimerID
}

type funcEntries []*funcEntry

func (s funcEntries) Len() int           { return len(s) }
func (s funcEntries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s funcEntries) Less(i, j int) bool { return s[i].when < s[j].when }
