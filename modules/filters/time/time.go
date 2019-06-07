package time

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/CN-TU/go-flows/packet"
	"github.com/CN-TU/go-flows/util"
	"github.com/google/gopacket"
)

type timeFilter struct {
	id                      string
	before, after           time.Time
	checkbefore, checkafter bool
}

func (tf *timeFilter) ID() string {
	return tf.id
}

func (tf *timeFilter) Init() {
}

func (tf *timeFilter) Matches(lt gopacket.LayerType, data []byte, ci gopacket.CaptureInfo, n uint64) bool {
	if tf.checkbefore && tf.before.Before(ci.Timestamp) {
		return false
	}
	if tf.checkafter && ci.Timestamp.Before(tf.after) {
		return false
	}
	return true
}

func newTimeFilter(args []string) (arguments []string, ret util.Module, err error) {
	var before, after time.Time
	var checkbefore, checkafter bool

	if len(args) == 0 {
		return nil, nil, errors.New("time filter needs a keyword (before, after, between) and time, was given none")
	}
	switch args[0] {
	case "before":
		if len(args) < 2 {
			return nil, nil, errors.New("time filter keyword 'before' needs a time argument")
		}
		checkbefore = true
		before, err = time.Parse(time.RFC3339Nano, args[1])
		if err != nil {
			return
		}
		arguments = args[2:]
	case "after":
		if len(args) < 2 {
			return nil, nil, errors.New("time filter keyword 'after' needs a time argument")
		}
		checkafter = true
		after, err = time.Parse(time.RFC3339Nano, args[1])
		if err != nil {
			return
		}
		arguments = args[2:]
	case "between":
		if len(args) < 3 {
			return nil, nil, errors.New("time filter keyword 'between' needs two time arguments")
		}
		checkafter = true
		after, err = time.Parse(time.RFC3339Nano, args[1])
		if err != nil {
			return
		}
		checkbefore = true
		before, err = time.Parse(time.RFC3339Nano, args[2])
		if err != nil {
			return
		}
		arguments = args[3:]
	default:
		return nil, nil, fmt.Errorf("time filter needs a keyword (before, after, between), but was given '%s'", args[0])
	}

	name := "time"
	if checkafter {
		name += fmt.Sprint("|>", after)
	}
	if checkbefore {
		name += fmt.Sprint("|>", before)
	}

	ret = &timeFilter{
		id:          name,
		before:      before,
		after:       after,
		checkafter:  checkafter,
		checkbefore: checkbefore,
	}
	return
}

func timeHelp(name string) {
	fmt.Fprintf(os.Stderr, `
The %s filter allows to filter packets based on time. Times must be
specified according to RFC3339 and can be given in  nanosecond accuracy.

Possible filters:
  before <time>
    Only packets with time < <time> are accepted.
  after <time>
    Only packets with time > <time> are accepted.
  between <a> <b>
    Only packets with <a> < arrival < <b> are accepted.

Usage:
  filter %s before|after|bewteen time [time]
`, name, name)
}

func init() {
	packet.RegisterFilter("time", "Filter packets based on time.", newTimeFilter, timeHelp)
}
