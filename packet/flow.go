package packet

import (
	"github.com/google/gopacket/layers"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

type TCPFlow struct {
	flows.BaseFlow
	srcFIN, dstFIN, dstACK, srcACK bool
}

type UniFlow struct {
	flows.BaseFlow
}

func NewFlow(event flows.Event, table *flows.FlowTable, key flows.FlowKey, time flows.DateTimeNanoSeconds) flows.Flow {
	if table.FiveTuple() {
		tp := event.(PacketBuffer).TransportLayer()
		if tp != nil && tp.LayerType() == layers.LayerTypeTCP {
			ret := new(TCPFlow)
			ret.Init(table, key, time)
			return ret
		}
	}
	ret := new(UniFlow)
	ret.Init(table, key, time)
	return ret
}

func (flow *TCPFlow) Event(event flows.Event, when flows.DateTimeNanoSeconds) {
	flow.BaseFlow.Event(event, when)
	if !flow.Active() {
		return
	}
	buffer := event.(PacketBuffer)
	tcp := buffer.TransportLayer().(*layers.TCP)
	if tcp.RST {
		flow.ExportWithoutContext(flows.FlowEndReasonEnd, when)
		return
	}
	if buffer.Forward() {
		if tcp.FIN {
			flow.srcFIN = true
		}
		if flow.dstFIN && tcp.ACK {
			flow.dstACK = true
		}
	} else {
		if tcp.FIN {
			flow.dstFIN = true
		}
		if flow.srcFIN && tcp.ACK {
			flow.srcACK = true
		}
	}

	if flow.srcFIN && flow.srcACK && flow.dstFIN && flow.dstACK {
		flow.ExportWithoutContext(flows.FlowEndReasonEnd, when)
	}
}
