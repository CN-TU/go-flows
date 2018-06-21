package packet

import (
	"encoding/binary"
	"net"
	"sort"
	"strings"

	"github.com/google/gopacket/tcpassembly"

	"github.com/CN-TU/go-ipfix"

	"github.com/CN-TU/go-flows/flows"
	"github.com/google/gopacket/layers"
)

////////////////////////////////////////////////////////////////////////////////

type sourceIPAddress struct {
	flows.BaseFeature
}

func (f *sourceIPAddress) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		network := new.(PacketBuffer).NetworkLayer()
		if network != nil {
			ipaddr := network.NetworkFlow().Src().Raw()
			if ipaddr != nil {
				fin := net.IP(ipaddr)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func (f *sourceIPAddress) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "sourceIPv4Address"
	}
	return 1 // "sourceIPv6Address"
}

func init() {
	flows.RegisterVariantFeature("sourceIPAddress", []ipfix.InformationElement{
		ipfix.GetInformationElement("sourceIPv4Address"),
		ipfix.GetInformationElement("sourceIPv6Address"),
	}, flows.FlowFeature, func() flows.Feature { return &sourceIPAddress{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type sourceTransportPort struct {
	flows.BaseFeature
}

func (f *sourceTransportPort) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		transport := new.(PacketBuffer).TransportLayer()
		if transport != nil {
			srcp := transport.TransportFlow().Src().Raw()
			if srcp != nil {
				fin := binary.BigEndian.Uint16(srcp)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func init() {
	flows.RegisterStandardFeature("sourceTransportPort", flows.FlowFeature, func() flows.Feature { return &sourceTransportPort{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationTransportPort struct {
	flows.BaseFeature
}

func (f *destinationTransportPort) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		transport := new.(PacketBuffer).TransportLayer()
		if transport != nil {
			dstp := transport.TransportFlow().Dst().Raw()
			if dstp != nil {
				fin := binary.BigEndian.Uint16(dstp)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func init() {
	flows.RegisterStandardFeature("destinationTransportPort", flows.FlowFeature, func() flows.Feature { return &destinationTransportPort{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationIPAddress struct {
	flows.BaseFeature
}

func (f *destinationIPAddress) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		network := new.(PacketBuffer).NetworkLayer()
		if network != nil {
			ipaddr := network.NetworkFlow().Dst().Raw()
			if ipaddr != nil {
				fin := net.IP(ipaddr)
				f.SetValue(fin, context, f)
			}
		}
	}
}

func (f *destinationIPAddress) Variant() int {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return 0 // "destinationIPv4Address"
	}
	return 1 // "destinationIPv6Address"
}

func init() {
	flows.RegisterVariantFeature("destinationIPAddress", []ipfix.InformationElement{
		ipfix.GetInformationElement("destinationIPv4Address"),
		ipfix.GetInformationElement("destinationIPv6Address"),
	}, flows.FlowFeature, func() flows.Feature { return &destinationIPAddress{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type protocolIdentifier struct {
	flows.BaseFeature
}

func (f *protocolIdentifier) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(new.(PacketBuffer).Proto(), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("protocolIdentifier", flows.FlowFeature, func() flows.Feature { return &protocolIdentifier{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowEndReason struct {
	flows.BaseFeature
}

func (f *flowEndReason) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(uint16(reason), context, f)
}

func init() {
	flows.RegisterStandardFeature("flowEndReason", flows.FlowFeature, func() flows.Feature { return &flowEndReason{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowEndNanoseconds struct {
	flows.BaseFeature
}

func (f *flowEndNanoseconds) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(context.When(), context, f)
}

func init() {
	flows.RegisterStandardFeature("flowEndNanoseconds", flows.FlowFeature, func() flows.Feature { return &flowEndNanoseconds{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowStartNanoseconds struct {
	flows.BaseFeature
}

func (f *flowStartNanoseconds) Start(context *flows.EventContext) {
	f.SetValue(context.When(), context, f)
}

func init() {
	flows.RegisterStandardFeature("flowStartNanoseconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowStartMicroseconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowStartMilliseconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
	flows.RegisterStandardFeature("flowStartSeconds", flows.FlowFeature, func() flows.Feature { return &flowStartNanoseconds{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type octetTotalCountPacket struct {
	flows.BaseFeature
}

func octetCount(packet PacketBuffer) uint64 {
	length := packet.Metadata().Length
	if net := packet.LinkLayer(); net != nil {
		length -= len(net.LayerContents())
	}
	return uint64(length)
}

func (f *octetTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(octetCount(new.(PacketBuffer)), context, f)
}

func init() {
	flows.RegisterStandardFeature("octetTotalCount", flows.PacketFeature, func() flows.Feature { return &octetTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type octetTotalCountFlow struct {
	flows.BaseFeature
	total uint64
}

func (f *octetTotalCountFlow) Start(context *flows.EventContext) {
	f.total = 0
}

func (f *octetTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.total += octetCount(new.(PacketBuffer))
}

func (f *octetTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.total, context, f)
}

func init() {
	flows.RegisterStandardFeature("octetTotalCount", flows.FlowFeature, func() flows.Feature { return &octetTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

func ipTotalLength(packet PacketBuffer) uint64 {
	network := packet.NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		return uint64(ip.Length)
	}
	if ip, ok := network.(*layers.IPv6); ok {
		if ip.HopByHop != nil {
			var tlv *layers.IPv6HopByHopOption
			for _, t := range ip.HopByHop.Options {
				if t.OptionType == layers.IPv6HopByHopOptionJumbogram {
					tlv = t
					break
				}
			}
			if tlv != nil && len(tlv.OptionData) == 4 {
				l := binary.BigEndian.Uint32(tlv.OptionData)
				if l > 65535 {
					return uint64(l)
				}
			}
		}
		return uint64(ip.Length)
	}
	return 0
}

type ipTotalLengthPacket struct {
	flows.BaseFeature
}

func (f *ipTotalLengthPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.SetValue(ipTotalLength(new.(PacketBuffer)), context, f)
}

func init() {
	flows.RegisterStandardFeature("ipTotalLength", flows.PacketFeature, func() flows.Feature { return &ipTotalLengthPacket{} }, flows.RawPacket)
}

type ipTotalLengthFlow struct {
	flows.BaseFeature
	total uint64
}

func (f *ipTotalLengthFlow) Start(context *flows.EventContext) {
	f.total = 0
}

func (f *ipTotalLengthFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.total += ipTotalLength(new.(PacketBuffer))
}

func (f *ipTotalLengthFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.total, context, f)
}

func init() {
	flows.RegisterStandardFeature("ipTotalLength", flows.FlowFeature, func() flows.Feature { return &ipTotalLengthFlow{} }, flows.RawPacket)
	flows.RegisterStandardCompositeFeature("minimumIpTotalLength", "min", "ipTotalLength")
	flows.RegisterStandardCompositeFeature("maximumIpTotalLength", "max", "ipTotalLength")
}

////////////////////////////////////////////////////////////////////////////////

type tcpControlBits struct {
	flows.BaseFeature
}

func getTCP(packet PacketBuffer) *layers.TCP {
	tcp := packet.Layer(layers.LayerTypeTCP)
	if tcp == nil {
		return nil
	}
	packetTCP := tcp.(*layers.TCP)
	return packetTCP
}

func (f *tcpControlBits) Event(new interface{}, context *flows.EventContext, src interface{}) {
	var value uint16
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	if tcp.FIN {
		value += 1 << 0
	}
	if tcp.SYN {
		value += 1 << 1
	}
	if tcp.RST {
		value += 1 << 2
	}
	if tcp.PSH {
		value += 1 << 3
	}
	if tcp.ACK {
		value += 1 << 4
	}
	if tcp.URG {
		value += 1 << 5
	}
	if tcp.ECE {
		value += 1 << 6
	}
	if tcp.CWR {
		value += 1 << 7
	}
	if tcp.NS {
		value += 1 << 8
	}
	f.SetValue(value, context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpControlBits", flows.PacketFeature, func() flows.Feature { return &tcpControlBits{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _interPacketTimeNanoseconds struct {
	flows.BaseFeature
	time flows.DateTimeNanoseconds
}

func (f *_interPacketTimeNanoseconds) Start(context *flows.EventContext) {
	f.time = 0
}

func (f *_interPacketTimeNanoseconds) Event(new interface{}, context *flows.EventContext, src interface{}) {
	var time flows.DateTimeNanoseconds
	if f.time != 0 {
		time = context.When() - f.time
	}
	f.time = context.When()
	f.SetValue(time, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_interPacketTimeNanoseconds", ipfix.DateTimeNanosecondsType, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("_interPacketTimeMicroseconds", ipfix.DateTimeMicrosecondsType, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("_interPacketTimeMilliseconds", ipfix.DateTimeMillisecondsType, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("_interPacketTimeSeconds", ipfix.DateTimeSecondsType, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type join struct {
	flows.MultiBaseFeature
}

func (f *join) Event(new interface{}, context *flows.EventContext, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	f.SetValue(values, context, f)
}

func init() {
	flows.RegisterFunction("join", flows.MatchType, func() flows.Feature { return &join{} }, flows.MatchType, flows.Ellipsis)
}

////////////////////////////////////////////////////////////////////////////////

type tcpSynTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *tcpSynTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *tcpSynTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func boolInt(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func (f *tcpSynTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.SYN)
}

func init() {
	flows.RegisterStandardFeature("tcpSynTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpSynTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpSynTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpSynTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.SYN), context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpSynTotalCount", flows.PacketFeature, func() flows.Feature { return &tcpSynTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpFinTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *tcpFinTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *tcpFinTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpFinTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.FIN)
}

func init() {
	flows.RegisterStandardFeature("tcpFinTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpFinTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpFinTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpFinTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.FIN), context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpFinTotalCount", flows.PacketFeature, func() flows.Feature { return &tcpFinTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpRstTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *tcpRstTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *tcpRstTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpRstTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.RST)
}

func init() {
	flows.RegisterStandardFeature("tcpRstTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpRstTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpRstTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpRstTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.RST), context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpRstTotalCount", flows.PacketFeature, func() flows.Feature { return &tcpRstTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpPshTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *tcpPshTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *tcpPshTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpPshTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.PSH)
}

func init() {
	flows.RegisterStandardFeature("tcpPshTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpPshTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpPshTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpPshTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.PSH), context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpPshTotalCount", flows.PacketFeature, func() flows.Feature { return &tcpPshTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpAckTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *tcpAckTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *tcpAckTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpAckTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.ACK)
}

func init() {
	flows.RegisterStandardFeature("tcpAckTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpAckTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
type tcpAckTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpAckTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.ACK), context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpAckTotalCount", flows.PacketFeature, func() flows.Feature { return &tcpAckTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpUrgTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *tcpUrgTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *tcpUrgTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpUrgTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.URG)
}

func init() {
	flows.RegisterStandardFeature("tcpUrgTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpUrgTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpUrgTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpUrgTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.URG), context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpUrgTotalCount", flows.PacketFeature, func() flows.Feature { return &tcpUrgTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpEceTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpEceTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *_tcpEceTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpEceTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.ECE)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpEceTotalCount", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &_tcpEceTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpEceTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpEceTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.ECE), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpEceTotalCount", ipfix.Unsigned64Type, 1, flows.PacketFeature, func() flows.Feature { return &_tcpEceTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpCwrTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpCwrTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *_tcpCwrTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpCwrTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.CWR)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpCwrTotalCount", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &_tcpCwrTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpCwrTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpCwrTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.CWR), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpCwrTotalCount", ipfix.Unsigned64Type, 1, flows.PacketFeature, func() flows.Feature { return &_tcpCwrTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpNsTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpNsTotalCountFlow) Start(context *flows.EventContext) {
	f.count = 0
}

func (f *_tcpNsTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpNsTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.NS)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpNsTotalCount", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &_tcpNsTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpNsTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpNsTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := getTCP(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.NS), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpNsTotalCount", ipfix.Unsigned64Type, 1, flows.PacketFeature, func() flows.Feature { return &_tcpNsTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _payload struct {
	flows.BaseFeature
}

func (f *_payload) Event(new interface{}, context *flows.EventContext, src interface{}) {
	packet := new.(PacketBuffer)
	if packet == nil {
		return
	}
	tl := packet.TransportLayer()
	if tl == nil {
		return
	}
	f.SetValue(string(tl.LayerPayload()), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_payload", ipfix.OctetArrayType, 0, flows.PacketFeature, func() flows.Feature { return &_payload{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

const invalidSequence = -1

type tcpFragment struct {
	seq    tcpassembly.Sequence
	plen   int
	packet PacketBuffer
}

type tcpFragments []tcpFragment

func (a tcpFragments) Len() int           { return len(a) }
func (a tcpFragments) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a tcpFragments) Less(i, j int) bool { return a[i].seq.Difference(a[j].seq) < 0 }

type uniTCPStreamFragments struct {
	fragments tcpFragments
	nextSeq   tcpassembly.Sequence
}

func (f *uniTCPStreamFragments) push(seq tcpassembly.Sequence, plen int, packet PacketBuffer) {
	f.fragments = append(f.fragments, tcpFragment{seq, plen, packet.Copy()})
	sort.Stable(f.fragments)
}

func (f *uniTCPStreamFragments) forwardOld(context *flows.EventContext, src interface{}) {
	if len(f.fragments) == 0 {
		return
	}
	deleted := -1

	for i, fragment := range f.fragments {
		if diff := fragment.seq.Difference(f.nextSeq); diff == 0 {
			// packet in order now
			f.forwardPacket(fragment.seq, fragment.plen, fragment.packet, context, src)
			fragment.packet.Recycle()
			deleted = i
		} else if diff == -1 {
			if fragment.plen == 0 {
				// valid in order keep alive (seq diff -1 && len == 0)
				f.forwardPacket(fragment.seq, fragment.plen, fragment.packet, context, src)
			}
			fragment.packet.Recycle()
			deleted = i
		} else if diff > 0 {
			//packet in future
			break
		}
	}
	if deleted == -1 {
		return
	}
	f.fragments = f.fragments[deleted+1:]
}

func (f *uniTCPStreamFragments) forwardPacket(seq tcpassembly.Sequence, plen int, packet PacketBuffer, context *flows.EventContext, src interface{}) {
	add := 0
	tcp := packet.TransportLayer().(*layers.TCP)
	if tcp.FIN || tcp.SYN { // hmm what happens if we have SYN and FIN at the same time? (should not happen - but well internet...)
		add = 1
	}
	f.nextSeq = f.nextSeq.Add(plen + add)
	context.Event(packet, context, src)
}

type tcpReorder struct {
	flows.EmptyBaseFeature
	forward  uniTCPStreamFragments
	backward uniTCPStreamFragments
}

func (f *tcpReorder) Start(*flows.EventContext) {
	f.forward = uniTCPStreamFragments{
		nextSeq: invalidSequence,
	}
	f.backward = uniTCPStreamFragments{
		nextSeq: invalidSequence,
	}
}

func (f *tcpReorder) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if context.IsHard() {
		for _, sequence := range f.forward.fragments {
			sequence.packet.Recycle()
		}
		for _, sequence := range f.backward.fragments {
			sequence.packet.Recycle()
		}
	} else {
		context.Keep()
	}
}

func (f *tcpReorder) Event(new interface{}, context *flows.EventContext, src interface{}) {
	packet := new.(PacketBuffer)
	tcp, ok := packet.TransportLayer().(*layers.TCP)
	if !ok {
		// not a tcp packet -> forward unchanged
		context.Event(new, context, src)
		return
	}

	var fragments *uniTCPStreamFragments
	if packet.Forward() {
		fragments = &f.forward
	} else {
		fragments = &f.backward
	}

	seq, plen := tcpassembly.Sequence(tcp.Seq), packet.Metadata().Length-packet.Hlen()

	if fragments.nextSeq == invalidSequence {
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
	flows.RegisterFilterFeature("tcpReorder", func() flows.Feature { return &tcpReorder{} })
}

////////////////////////////////////////////////////////////////////////////////

type ipTTL struct {
	flows.BaseFeature
}

func (f *ipTTL) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(PacketBuffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(uint64(ip.TTL), context, f)
	}
	if ip, ok := network.(*layers.IPv6); ok {
		f.SetValue(uint64(ip.HopLimit), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("ipTTL", flows.PacketFeature, func() flows.Feature { return &ipTTL{} }, flows.RawPacket)
	flows.RegisterStandardCompositeFeature("minimumTTL", "min", "ipTTL")
	flows.RegisterStandardCompositeFeature("maximumTTL", "max", "ipTTL")
}

////////////////////////////////////////////////////////////////////////////////

type ipClassOfService struct {
	flows.BaseFeature
}

func (f *ipClassOfService) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(PacketBuffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.TOS, context, f)
	}
	if ip, ok := network.(*layers.IPv6); ok {
		f.SetValue(ip.TrafficClass, context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("ipClassOfService", flows.PacketFeature, func() flows.Feature { return &ipClassOfService{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _HTTPLines struct {
	flows.BaseFeature
	buffer     []byte
	status     uint8
	statusNext uint8
	current    string
	src        string
}

const _None uint8 = 0
const _Request uint8 = 1
const _Response uint8 = 2
const _Header uint8 = 3
const _Body uint8 = 4
const _Error string = "-1"

func (f *_HTTPLines) Start(context *flows.EventContext) {
	f.status = _Request
	f.statusNext = _None
}

func (f *_HTTPLines) extractLine(ignore bool) (ret string) {
	data := string(f.buffer) // TODO maybe it's better to keep it as bytes and not use strings?
	if ignore {
		data = strings.TrimLeft(data, " \t\n\r")
	}
	dataSplit := strings.SplitN(data, "\n", 2)
	if len(dataSplit) != 2 {
		return _Error
	}
	f.buffer = []byte(dataSplit[1]) // TODO here maybe there's unnecessary allocations
	return strings.TrimSpace(dataSplit[0])
}

func (f *_HTTPLines) extractRequest(line string) (ret string) {
	// TODO implement proper request parsing
	lineSplit := strings.Split(line, " ")
	return lineSplit[0]
}

func (f *_HTTPLines) parsePart(context *flows.EventContext, src interface{}) (ret bool) {
	// TODO nothing is implemented except for extracting headers in simple http sessions
	switch f.status {
	case _Request:
		line := f.extractLine(true)
		if line == _Error {
			return false
		}
		f.SetValue(line, context, src)
		f.statusNext = _Response
		f.status = _Header
		f.current = f.extractRequest(line)
		if f.current == "" {
			f.status = _Request
		}
	case _Header:
		line := f.extractLine(false)
		if line == _Error {
			return false
		}
		if line == "" { // end of header
			return false // FIXME this should continue, and do more stuff
		}
		f.SetValue(line, context, src)
	default:
	}
	return true
}

func (f *_HTTPLines) Event(new interface{}, context *flows.EventContext, src interface{}) {
	f.buffer = append(f.buffer, []byte(new.(string))...)
	for f.parsePart(context, src) == true {
		continue
	}
}

func (f *_HTTPLines) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
}

func init() {
	flows.RegisterFunction("httpLines", flows.PacketFeature, func() flows.Feature { return &_HTTPLines{} }, flows.PacketFeature)
	ieText := []byte("_HTTPLines(667)<octetArray>") // FIXME get number for IE
	ie := ipfix.MakeIEFromSpec(ieText)
	flows.RegisterCompositeFeature(ie, "httpLines", "_tcpReorderPayload")
}

type httpRequestHost struct {
	flows.BaseFeature
}

func (f *httpRequestHost) Event(new interface{}, context *flows.EventContext, src interface{}) {
	header := strings.SplitN(new.(string), ":", 2)
	if header[0] == "Host" {
		f.SetValue(strings.TrimSpace(header[1]), context, src)
	}
}

func init() {
	flows.RegisterFunction("__httpRequestHost", flows.FlowFeature, func() flows.Feature { return &httpRequestHost{} }, flows.PacketFeature)
	flows.RegisterStandardCompositeFeature("httpRequestHost", "__httpRequestHost", "_HTTPLines")
}

////////////////////////////////////////////////////////////////////////////////

type flowDirection struct {
	flows.BaseFeature
}

func (f *flowDirection) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(new.(PacketBuffer).Forward(), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("flowDirection", flows.FlowFeature, func() flows.Feature { return &flowDirection{} }, flows.RawPacket)
}
