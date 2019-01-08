package time

import (
	"encoding/binary"
	"time"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
)

const keyPrefix = "__timeWindow"

func makeTimeWindowKey(name string) packet.KeyFunc {
	d, err := time.ParseDuration(name[len(keyPrefix):])
	if err != nil {
		panic(err)
	}
	var start flows.DateTimeNanoseconds
	duration := flows.DateTimeNanoseconds(d.Nanoseconds())
	var id uint64
	return func(packet packet.Buffer, scratch, scratchNoSort []byte) (int, int) {
		if start == 0 {
			start = packet.Timestamp()
		} else if start+duration < packet.Timestamp() {
			num := (packet.Timestamp() - start) / duration
			start += num * duration
			id += uint64(num)
		}
		binary.BigEndian.PutUint64(scratch, id)
		return 8, 0
	}
}

func init() {
	packet.RegisterRegexpKey("^"+keyPrefix,
		"time window id; Must be suffixed by a duration specification parsable by time.ParseDuration (e.g. 60s)",
		packet.KeyTypeUnidirectional, packet.KeyLayerNone, makeTimeWindowKey)
}
