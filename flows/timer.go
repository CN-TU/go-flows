package flows

// TimerID represents a single timer
type TimerID int

// TimerCallback is a function the gets called upon a timer event. This event receives the expiry time and the current time.
type TimerCallback func(*EventContext, DateTimeNanoseconds)

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
	context  *EventContext
}

type funcEntries []funcEntry

func makeFuncEntries() funcEntries {
	return make(funcEntries, 2)
}

func (fe *funcEntries) expire(when DateTimeNanoseconds) DateTimeNanoseconds {
	var next DateTimeNanoseconds
	fep := *fe
	for i, v := range fep {
		if v.context != nil {
			if v.context.when <= when {
				fep[i].function(v.context, when)
				fep[i].context = nil
			} else if next == 0 || v.context.when <= next {
				next = v.context.when
			}
		}
	}
	return next
}

func (fe *funcEntries) addTimer(id TimerID, f TimerCallback, context *EventContext) {
	fep := *fe
	if int(id) >= len(fep) {
		fep = append(fep, make(funcEntries, int(id)-len(fep)+1)...)
		*fe = fep
	}
	fep[id].function = f
	fep[id].context = context
}

func (fe *funcEntries) hasTimer(id TimerID) bool {
	fep := *fe
	if int(id) >= len(fep) || id < 0 {
		return false
	}
	if fep[id].context == nil {
		return false
	}
	return true
}
