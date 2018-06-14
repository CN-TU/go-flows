package packet

import (
	"github.com/CN-TU/go-flows/flows"
	"github.com/google/gopacket/layers"
)

type TCPFlow struct {
	flows.BaseFlow
	srcFIN, dstFIN, dstACK, srcACK bool
}

type UniFlow struct {
	flows.BaseFlow
}

func NewFlow(event flows.Event, table *flows.FlowTable, key flows.FlowKey, context *flows.EventContext) flows.Flow {
	if table.FiveTuple() {
		tp := event.(PacketBuffer).TransportLayer()
		if tp != nil && tp.LayerType() == layers.LayerTypeTCP {
			ret := new(TCPFlow)
			ret.Init(table, key, context)
			return ret
		}
	}
	ret := new(UniFlow)
	ret.Init(table, key, context)
	return ret
}

func (flow *TCPFlow) Event(event flows.Event, context *flows.EventContext) {
	flow.BaseFlow.Event(event, context)
	if !flow.Active() {
		return
	}
	buffer := event.(PacketBuffer)
	tcp := buffer.TransportLayer().(*layers.TCP)
	if tcp.RST {
		flow.Export(flows.FlowEndReasonEnd, context, context.When())
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
		flow.Export(flows.FlowEndReasonEnd, context, context.When())
	}
}
