package flows

// TimerID represents a single timer
type TimerID int

// TimerCallback is a function the gets called upon a timer event. This event receives the expiry time and the current time.
type TimerCallback func(expires, now DateTimeNanoseconds)

var timerMaxID TimerID

// RegisterTimer registers a new timer and returns the new TimerID.
func RegisterTimer() TimerID {
	ret := timerMaxID
	timerMaxID++
	return ret
}

var (
	// TimerIdle is the idle timer of every flow
	TimerIdle = RegisterTimer()
	// TimerActive is the active timer of every flow
	TimerActive = RegisterTimer()
)

type funcEntry struct {
	function TimerCallback
	expires  DateTimeNanoseconds
}

type funcEntries []funcEntry

func makeFuncEntries() funcEntries {
	return make(funcEntries, 2)
}

func (fe *funcEntries) expire(when DateTimeNanoseconds) DateTimeNanoseconds {
	var next DateTimeNanoseconds
	fep := *fe
	for i, v := range fep {
		if v.expires != 0 {
			if v.expires <= when {
				fep[i].function(fep[i].expires, when)
				fep[i].expires = 0
			} else if next == 0 || v.expires <= next {
				next = v.expires
			}
		}
	}
	return next
}

func (fe *funcEntries) addTimer(id TimerID, f TimerCallback, when DateTimeNanoseconds) {
	fep := *fe
	if int(id) >= len(fep) {
		fep = append(fep, make(funcEntries, int(id)-len(fep)+1)...)
		*fe = fep
	}
	fep[id].function = f
	fep[id].expires = when
}

func (fe *funcEntries) hasTimer(id TimerID) bool {
	fep := *fe
	if int(id) >= len(fep) || id < 0 {
		return false
	}
	if fep[id].expires == 0 {
		return false
	}
	return true
}

func (fe *funcEntries) removeTimer(id TimerID) {
	fep := *fe
	if int(id) >= len(fep) || id < 0 {
		return
	}
	fep[id].expires = 0
}
