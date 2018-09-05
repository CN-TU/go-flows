package features

import (
	"github.com/CN-TU/go-flows/packet"
	"github.com/google/gopacket/layers"
)

// GetTCP returns the TCP layer of the packet or nil
func GetTCP(new interface{}) *layers.TCP {
	tcp := new.(packet.Buffer).TransportLayer()
	if tcp == nil {
		return nil
	}
	packetTCP := tcp.(*layers.TCP)
	return packetTCP
}

// Since it is not possible to reference this without pulling in side effects, the following is
// from github.com/google/gopacket/tcpassembly/assembly.go licensed under github.com/google/gopacket/LICENSE
// vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv

// InvalidSequence represents an invalid sequence number
const InvalidSequence = -1
const uint32Size = 1 << 32

// Sequence is a TCP sequence number.  It provides a few convenience functions
// for handling TCP wrap-around.  The sequence should always be in the range
// [0,0xFFFFFFFF]... its other bits are simply used in wrap-around calculations
// and should never be set.
type Sequence int64

// Difference defines an ordering for comparing TCP sequences that's safe for
// roll-overs.  It returns:
//    > 0 : if t comes after s
//    < 0 : if t comes before s
//      0 : if t == s
// The number returned is the sequence difference, so 4.Difference(8) will
// return 4.
//
// It handles rollovers by considering any sequence in the first quarter of the
// uint32 space to be after any sequence in the last quarter of that space, thus
// wrapping the uint32 space.
func (s Sequence) Difference(t Sequence) int {
	if s > uint32Size-uint32Size/4 && t < uint32Size/4 {
		t += uint32Size
	} else if t > uint32Size-uint32Size/4 && s < uint32Size/4 {
		s += uint32Size
	}
	return int(t - s)
}

// Add adds an integer to a sequence and returns the resulting sequence.
func (s Sequence) Add(t int) Sequence {
	return (s + Sequence(t)) & (uint32Size - 1)
}

// ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
