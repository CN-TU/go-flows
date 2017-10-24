package packet

import (
	"bytes"
	"strings"

	"pm.cn.tuwien.ac.at/ipfix/go-ipfix"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

/*

Features in here are subject to change. Use them with caution.

*/

////////////////////////////////////////////////////////////////////////////////

type _characters struct {
	flows.BaseFeature
	time flows.DateTimeNanoseconds
	src  []byte
}

func (f *_characters) Event(new interface{}, context flows.EventContext, src interface{}) {
	var time flows.DateTimeNanoseconds
	if f.time != 0 {
		time = context.When - f.time
	}
	if len(f.src) == 0 {
		tmpSrc, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()
		f.src = tmpSrc.Raw()
	}
	f.time = context.When
	newTime := int(time / (100 * flows.MillisecondsInNanoseconds)) // time is now in deciseconds

	srcIP, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()

	var buffer bytes.Buffer
	if bytes.Equal(f.src, srcIP.Raw()) {
		buffer.WriteString("A")
	} else {
		buffer.WriteString("B")
	}

	buffer.WriteString(strings.Repeat("_", newTime))

	f.SetValue(buffer.String(), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_characters", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_characters{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _characters2 struct {
	flows.BaseFeature
	time flows.DateTimeNanoseconds
	src  []byte
}

func (f *_characters2) Event(new interface{}, context flows.EventContext, src interface{}) {
	var time flows.DateTimeNanoseconds
	if f.time != 0 {
		time = context.When - f.time
	}
	if len(f.src) == 0 {
		tmpSrc, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()
		f.src = tmpSrc.Raw()
	}
	f.time = context.When
	newTime := int(time / (100 * flows.MillisecondsInNanoseconds)) // time is now in deciseconds

	srcIP, _ := new.(PacketBuffer).NetworkLayer().NetworkFlow().Endpoints()
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}

	var buffer bytes.Buffer
	if bytes.Equal(f.src, srcIP.Raw()) {
		// A->B
		if tcp.FIN && tcp.ACK {
			buffer.WriteString("n")
		} else if tcp.FIN {
			buffer.WriteString("f")
		} else if tcp.SYN && tcp.ACK {
			buffer.WriteString("k")
		} else if tcp.SYN {
			buffer.WriteString("s")
		} else if tcp.RST && tcp.ACK {
			buffer.WriteString("t")
		} else if tcp.RST {
			buffer.WriteString("r")
		} else if tcp.PSH && tcp.ACK {
			buffer.WriteString("h")
		} else if tcp.PSH {
			buffer.WriteString("p")
		} else if tcp.ACK {
			buffer.WriteString("a")
		} else if tcp.URG {
			buffer.WriteString("u")
		} else {
			buffer.WriteString("o")
		}
	} else {
		// B->A
		if tcp.FIN && tcp.ACK {
			buffer.WriteString("N")
		} else if tcp.FIN {
			buffer.WriteString("F")
		} else if tcp.SYN && tcp.ACK {
			buffer.WriteString("K")
		} else if tcp.SYN {
			buffer.WriteString("S")
		} else if tcp.RST && tcp.ACK {
			buffer.WriteString("T")
		} else if tcp.RST {
			buffer.WriteString("R")
		} else if tcp.PSH && tcp.ACK {
			buffer.WriteString("H")
		} else if tcp.PSH {
			buffer.WriteString("P")
		} else if tcp.ACK {
			buffer.WriteString("A")
		} else if tcp.URG {
			buffer.WriteString("U")
		} else {
			buffer.WriteString("O")
		}
	}

	buffer.WriteString(strings.Repeat("-", newTime))

	f.SetValue(buffer.String(), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_characters2", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_characters2{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

// outputs number of consecutive seconds in which there was at least one packet
// seconds are counted from the first packet
type __consecutiveSeconds struct {
	flows.BaseFeature
	count    uint64
	lastTime flows.DateTimeNanoseconds
}

func (f *__consecutiveSeconds) Start(context flows.EventContext) {
	f.lastTime = 0
	f.count = 0
}

func (f *__consecutiveSeconds) Event(new interface{}, context flows.EventContext, src interface{}) {
	time := context.When
	if f.lastTime == 0 {
		f.lastTime = time
		f.count++
	} else {
		if time-f.lastTime > 1*flows.SecondsInNanoseconds { // if time difference to f.lastTime is more than one second
			f.lastTime = time
			if time-f.lastTime < 2*flows.SecondsInNanoseconds { // if there is less than 2s between this and last packet, count
				f.count++
			} else { // otherwise, there was a break in seconds between packets
				f.SetValue(f.count, context, f)
				f.count = 1
			}
		}
	}
}

func (f *__consecutiveSeconds) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("__consecutiveSeconds", ipfix.Unsigned64Type, 0, flows.PacketFeature, func() flows.Feature { return &__consecutiveSeconds{} }, flows.RawPacket)
	flows.RegisterTemporaryCompositeFeature("__maximumConsecutiveSeconds", ipfix.Unsigned64Type, 0, "maximum", "__consecutiveSeconds")
	flows.RegisterTemporaryCompositeFeature("__minimumConsecutiveSeconds", ipfix.Unsigned64Type, 0, "minimum", "__consecutiveSeconds")
}

////////////////////////////////////////////////////////////////////////////////

// outputs number of seconds in which there was at least one packet
// seconds are counted from the first packet
type _activeForSeconds struct {
	flows.BaseFeature
	count    uint64
	lastTime flows.DateTimeNanoseconds
}

func (f *_activeForSeconds) Start(context flows.EventContext) {
	f.lastTime = 0
	f.count = 0
}

func (f *_activeForSeconds) Event(new interface{}, context flows.EventContext, src interface{}) {
	time := context.When
	if f.lastTime == 0 {
		f.lastTime = time
		f.count++
	} else {
		if time-f.lastTime > 1*flows.SecondsInNanoseconds { // if time difference to f.lastTime is more than one second
			f.lastTime = time
			f.count++
		}
	}
}

func (f *_activeForSeconds) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_activeForSeconds", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &_activeForSeconds{} }, flows.RawPacket)
}
