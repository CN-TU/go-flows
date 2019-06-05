package custom

import (
	"sort"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/modules/features"
	"github.com/CN-TU/go-flows/packet"
	ipfix "github.com/CN-TU/go-ipfix"
	"github.com/google/gopacket/layers"
)

type _tcpEceTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpEceTotalCountFlow) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *_tcpEceTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpEceTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.ECE)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpEceTotalCount", "count of TCP packets with ECE flag", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &_tcpEceTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpEceTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpEceTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.ECE), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpEceTotalCount", "count of TCP packets with ECE flag", ipfix.Unsigned64Type, 1, flows.PacketFeature, func() flows.Feature { return &_tcpEceTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpCwrTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpCwrTotalCountFlow) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *_tcpCwrTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpCwrTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.CWR)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpCwrTotalCount", "count of TCP packets with CWR flag", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &_tcpCwrTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpCwrTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpCwrTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.CWR), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpCwrTotalCount", "count of TCP packets with CWR flag", ipfix.Unsigned64Type, 1, flows.PacketFeature, func() flows.Feature { return &_tcpCwrTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpNsTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpNsTotalCountFlow) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *_tcpNsTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpNsTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.NS)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpNsTotalCount", "count of TCP packets with NS flag", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &_tcpNsTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpNsTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpNsTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.NS), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpNsTotalCount", "count of TCP packets with NS flag", ipfix.Unsigned64Type, 1, flows.PacketFeature, func() flows.Feature { return &_tcpNsTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpFragment struct {
	seq    features.Sequence
	plen   int
	packet packet.Buffer
}

type tcpFragments []tcpFragment

func (a tcpFragments) Len() int           { return len(a) }
func (a tcpFragments) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a tcpFragments) Less(i, j int) bool { return a[i].seq.Difference(a[j].seq) < 0 }

type uniTCPStreamFragments struct {
	fragments tcpFragments
	nextSeq   features.Sequence
}

func (f *uniTCPStreamFragments) push(seq features.Sequence, plen int, packet packet.Buffer) {
	f.fragments = append(f.fragments, tcpFragment{seq, plen, packet.Copy()})
	sort.Stable(f.fragments)
}

func (f *uniTCPStreamFragments) forwardOld(context *flows.EventContext, src interface{}) {
	if len(f.fragments) == 0 {
		return
	}
	deleted := 0

	for i := len(f.fragments) - 1; i >= 0; i-- {
		fragment := f.fragments[i]
		if diff := fragment.seq.Difference(f.nextSeq); diff == 0 {
			// packet in order now
			f.forwardPacket(fragment.seq, fragment.plen, fragment.packet, context, src)
			fragment.packet.Recycle()
			deleted++
		} else if diff == -1 {
			if fragment.plen == 0 {
				// valid in order keep alive (seq diff -1 && len == 0)
				f.forwardPacket(fragment.seq, fragment.plen, fragment.packet, context, src)
			}
			fragment.packet.Recycle()
			deleted++
		} else if diff > 0 {
			//packet in future
			break
		}
	}
	if deleted == 0 {
		return
	}
	f.fragments = f.fragments[:len(f.fragments)-deleted]
}

func (f *uniTCPStreamFragments) forwardPacket(seq features.Sequence, plen int, packet packet.Buffer, context *flows.EventContext, src interface{}) {
	add := 0
	tcp := packet.TransportLayer().(*layers.TCP)
	if tcp.FIN || tcp.SYN { // hmm what happens if we have SYN and FIN at the same time? (should not happen - but well internet...)
		add = 1
	}
	f.nextSeq = f.nextSeq.Add(plen + add)
	context.Event(packet, context, src)
}

func (f *uniTCPStreamFragments) maybeForwardOld(ack features.Sequence, context *flows.EventContext, src interface{}) {
	if len(f.fragments) == 0 {
		return
	}
	zero := f.fragments[len(f.fragments)-1]
	if zero.seq.Difference(ack) < 0 {
		return
	}
	f.nextSeq = zero.seq
	f.forwardPacket(zero.seq, zero.plen, zero.packet, context, src)
	zero.packet.Recycle()
	f.fragments = f.fragments[:len(f.fragments)-1]
	f.forwardOld(context, src)
}

type tcpReorder struct {
	flows.NoopFeature
	forward  uniTCPStreamFragments
	backward uniTCPStreamFragments
}

func (f *tcpReorder) Start(*flows.EventContext) {
	f.forward = uniTCPStreamFragments{
		nextSeq: features.InvalidSequence,
	}
	f.backward = uniTCPStreamFragments{
		nextSeq: features.InvalidSequence,
	}
}

func (f *tcpReorder) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	//FIXME: do timeout for tcp wait close, or replay left over packets?
	//	if context.IsHard() {
	for _, sequence := range f.forward.fragments {
		sequence.packet.Recycle()
	}
	for _, sequence := range f.backward.fragments {
		sequence.packet.Recycle()
	}
	/*	} else {
		context.Keep()
	}*/
}

func (f *tcpReorder) Event(new interface{}, context *flows.EventContext, src interface{}) {
	packet := new.(packet.Buffer)
	tcp, ok := packet.TransportLayer().(*layers.TCP)
	if !ok {
		// not a tcp packet -> forward unchanged
		context.Event(new, context, src)
		return
	}

	var fragments, back *uniTCPStreamFragments
	if context.Forward() {
		fragments = &f.forward
		back = &f.backward
	} else {
		fragments = &f.backward
		back = &f.forward
	}

	back.maybeForwardOld(features.Sequence(tcp.Ack), context, src)

	seq, plen := features.Sequence(tcp.Seq), packet.PayloadLength()

	if fragments.nextSeq == features.InvalidSequence {
		// first packet; set sequence start and emit
		fragments.nextSeq = seq
		fragments.forwardPacket(seq, plen, packet, context, src)
	} else if diff := fragments.nextSeq.Difference(seq); diff == 0 {
		// packet at current position -> forward for further processing + look if we have old ones segments
		fragments.forwardPacket(seq, plen, packet, context, src)
		fragments.forwardOld(context, src)
	} else if diff > 0 {
		// packet from the future -> store fore later
		fragments.push(seq, plen, packet)
	} else if diff == -1 && plen == 0 {
		// keep alive packet -> let it through
		context.Event(packet, context, src)
	}
	// ignore all the other packets (past, invalid keep alive)
}

func init() {
	flows.RegisterFilterFeature("tcpReorder", "returns tcp packets ordered by sequence number; non-tcp packets are passed unmodified.", func() flows.Feature { return &tcpReorder{} })
}

////////////////////////////////////////////////////////////////////////////////

type tcpflags struct {
	flows.BaseFeature
}

func (f *tcpflags) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	flag := ""
	if tcp != nil {
		if tcp.SYN {
			flag += "S"
		}
		if tcp.FIN {
			flag += "F"
		}
		if tcp.RST {
			flag += "R"
		}
		if tcp.PSH {
			flag += "P"
		}
		if tcp.ACK {
			flag += "A"
		}
		if tcp.URG {
			flag += "U"
		}
		if tcp.ECE {
			flag += "E"
		}
		if tcp.CWR {
			flag += "C"
		}
		if tcp.NS {
			flag += "N"
		}
	}
	f.SetValue(flag, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpFlags", "returns a textual representation of the tcpflags", ipfix.StringType, 1, flows.FlowFeature, func() flows.Feature { return &tcpflags{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpflagsPacket struct {
	flows.BaseFeature
}

func (f *tcpflagsPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	flag := ""
	if tcp != nil {
		if tcp.SYN {
			flag += "S"
		}
		if tcp.FIN {
			flag += "F"
		}
		if tcp.RST {
			flag += "R"
		}
		if tcp.PSH {
			flag += "P"
		}
		if tcp.ACK {
			flag += "A"
		}
		if tcp.URG {
			flag += "U"
		}
		if tcp.ECE {
			flag += "E"
		}
		if tcp.CWR {
			flag += "C"
		}
		if tcp.NS {
			flag += "N"
		}
	}
	f.SetValue(flag, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpFlags", "returns a textual representation of the tcpflags", ipfix.StringType, 1, flows.PacketFeature, func() flows.Feature { return &tcpflags{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
