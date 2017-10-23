package packet

import (
	"encoding/binary"
	"net"

	"pm.cn.tuwien.ac.at/ipfix/go-ipfix"

	"github.com/google/gopacket/layers"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

////////////////////////////////////////////////////////////////////////////////

type sourceIPAddress struct {
	flows.BaseFeature
}

func (f *sourceIPAddress) Event(new interface{}, context flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(net.IP(new.(PacketBuffer).NetworkLayer().NetworkFlow().Src().Raw()), context, f)
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

func (f *sourceTransportPort) Event(new interface{}, context flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(binary.BigEndian.Uint16(new.(PacketBuffer).TransportLayer().TransportFlow().Src().Raw()), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("sourceTransportPort", flows.FlowFeature, func() flows.Feature { return &sourceTransportPort{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationTransportPort struct {
	flows.BaseFeature
}

func (f *destinationTransportPort) Event(new interface{}, context flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(binary.BigEndian.Uint16(new.(PacketBuffer).TransportLayer().TransportFlow().Dst().Raw()), context, f)
	}
}

func init() {
	flows.RegisterStandardFeature("destinationTransportPort", flows.FlowFeature, func() flows.Feature { return &destinationTransportPort{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type destinationIPAddress struct {
	flows.BaseFeature
}

func (f *destinationIPAddress) Event(new interface{}, context flows.EventContext, src interface{}) {
	if f.Value() == nil {
		f.SetValue(net.IP(new.(PacketBuffer).NetworkLayer().NetworkFlow().Dst().Raw()), context, f)
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

func (f *protocolIdentifier) Event(new interface{}, context flows.EventContext, src interface{}) {
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

func (f *flowEndReason) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(reason, context, f)
}

func init() {
	flows.RegisterStandardFeature("flowEndReason", flows.FlowFeature, func() flows.Feature { return &flowEndReason{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type flowEndNanoseconds struct {
	flows.BaseFeature
}

func (f *flowEndNanoseconds) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(context.When, context, f)
}

func init() {
	flows.RegisterStandardFeature("flowEndNanoseconds", flows.FlowFeature, func() flows.Feature { return &flowEndNanoseconds{} }, flows.RawPacket)
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

func (f *octetTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
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

func (f *octetTotalCountFlow) Start(context flows.EventContext) {
	f.total = 0
}

func (f *octetTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	f.total += octetCount(new.(PacketBuffer))
}

func (f *octetTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
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

func (f *ipTotalLengthPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	f.SetValue(ipTotalLength(new.(PacketBuffer)), context, f)
}

func init() {
	flows.RegisterStandardFeature("ipTotalLength", flows.PacketFeature, func() flows.Feature { return &ipTotalLengthPacket{} }, flows.RawPacket)
}

type ipTotalLengthFlow struct {
	flows.BaseFeature
	total uint64
}

func (f *ipTotalLengthFlow) Start(context flows.EventContext) {
	f.total = 0
}

func (f *ipTotalLengthFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	f.total += ipTotalLength(new.(PacketBuffer))
}

func (f *ipTotalLengthFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
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

func getTcp(packet PacketBuffer) *layers.TCP {
	tcp := packet.Layer(layers.LayerTypeTCP)
	if tcp == nil {
		return nil
	}
	packet_tcp := tcp.(*layers.TCP)
	return packet_tcp
}

func (f *tcpControlBits) Event(new interface{}, context flows.EventContext, src interface{}) {
	var value uint16
	tcp := getTcp(new.(PacketBuffer))
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

func (f *_interPacketTimeNanoseconds) Start(context flows.EventContext) {
	f.time = 0
}

func (f *_interPacketTimeNanoseconds) Event(new interface{}, context flows.EventContext, src interface{}) {
	var time flows.DateTimeNanoseconds
	if f.time != 0 {
		time = context.When - f.time
	}
	f.time = context.When
	f.SetValue(time, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_interPacketTimeNanoseconds", ipfix.DateTimeNanoseconds, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("_interPacketTimeNanoseconds", ipfix.DateTimeMicroseconds, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("_interPacketTimeNanoseconds", ipfix.DateTimeMilliseconds, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
	flows.RegisterTemporaryFeature("_interPacketTimeNanoseconds", ipfix.DateTimeSeconds, 0, flows.PacketFeature, func() flows.Feature { return &_interPacketTimeNanoseconds{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type join struct {
	flows.MultiBaseFeature
}

func (f *join) Event(new interface{}, context flows.EventContext, src interface{}) {
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

func (f *tcpSynTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *tcpSynTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func boolInt(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func (f *tcpSynTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpSynTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpFinTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *tcpFinTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpFinTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpFinTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpRstTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *tcpRstTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpRstTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpRstTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpPshTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *tcpPshTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpPshTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpPshTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpAckTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *tcpAckTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpAckTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpAckTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpUrgTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *tcpUrgTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpUrgTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *tcpUrgTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
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

func (f *_tcpEceTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *_tcpEceTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpEceTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.ECE)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpEceTotalCount", ipfix.Unsigned64, 0, flows.FlowFeature, func() flows.Feature { return &_tcpEceTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpEceTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpEceTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.ECE), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpEceTotalCount", ipfix.Unsigned64, 1, flows.PacketFeature, func() flows.Feature { return &_tcpEceTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpCwrTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpCwrTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *_tcpCwrTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpCwrTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.CWR)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpCwrTotalCount", ipfix.Unsigned64, 0, flows.FlowFeature, func() flows.Feature { return &_tcpCwrTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpCwrTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpCwrTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.CWR), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpCwrTotalCount", ipfix.Unsigned64, 1, flows.PacketFeature, func() flows.Feature { return &_tcpCwrTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpNsTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *_tcpNsTotalCountFlow) Start(context flows.EventContext) {
	f.count = 0
}

func (f *_tcpNsTotalCountFlow) Stop(reason flows.FlowEndReason, context flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *_tcpNsTotalCountFlow) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.count += boolInt(tcp.NS)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpNsTotalCount", ipfix.Unsigned64, 0, flows.FlowFeature, func() flows.Feature { return &_tcpNsTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _tcpNsTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpNsTotalCountPacket) Event(new interface{}, context flows.EventContext, src interface{}) {
	tcp := getTcp(new.(PacketBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.NS), context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_tcpNsTotalCount", ipfix.Unsigned64, 1, flows.PacketFeature, func() flows.Feature { return &_tcpNsTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _payload struct {
	flows.BaseFeature
}

func (f *_payload) Event(new interface{}, context flows.EventContext, src interface{}) {
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
	flows.RegisterTemporaryFeature("_payload", ipfix.OctetArray, 0, flows.PacketFeature, func() flows.Feature { return &_payload{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpFragment struct {
	packet   PacketBuffer
	position uint32
}

type uniTCPStreamFragments struct {
	fragments                        []tcpFragment
	acks                             []tcpFragment
	firstSequence, lastSequence, pos uint32
	seen                             bool
}

func (f *uniTCPStreamFragments) contains(position uint32, ack bool) bool {
	if ack {
		for _, fragment := range f.acks {
			if fragment.position == position {
				return true
			}
		}
	} else {
		for _, fragment := range f.fragments {
			if fragment.position == position {
				return true
			}
		}
	}
	return false
}

func (f *uniTCPStreamFragments) pushFragment(position uint32, packet PacketBuffer) {
	if packet.Metadata().Length-packet.Hlen() == 0 {
		f.acks = append(f.acks, tcpFragment{packet, position})
	} else {
		f.fragments = append(f.fragments, tcpFragment{packet, position})
	}
}

func (f *uniTCPStreamFragments) popFragment(position uint32) (ret PacketBuffer) {
	for i, fragment := range f.acks {
		if fragment.position == position {
			ret = f.acks[i].packet
			f.acks[i] = f.acks[len(f.acks)-1]
			f.acks = f.acks[:len(f.acks)-1]
			return
		}
	}
	for i, fragment := range f.fragments {
		if fragment.position == position {
			ret = f.fragments[i].packet
			f.fragments[i] = f.fragments[len(f.fragments)-1]
			f.fragments = f.fragments[:len(f.fragments)-1]
			return
		}
	}
	return nil
}

type tcpReorder struct {
	flows.EmptyBaseFeature
	forward  uniTCPStreamFragments
	backward uniTCPStreamFragments
}

func (f *tcpReorder) Type() string     { return "tcpReorder" }
func (f *tcpReorder) BaseType() string { return "tcpReorder" }
func (f *tcpReorder) Start(flows.EventContext) {
	f.forward = uniTCPStreamFragments{}
	f.backward = uniTCPStreamFragments{}
}

func payloadLength(packet PacketBuffer) int {
	length := packet.Metadata().Length
	if net := packet.LinkLayer(); net != nil {
		length -= len(net.LayerContents())
	}
	return length
}

func (f *tcpReorder) Event(new interface{}, context flows.EventContext, src interface{}) {
	packet := new.(PacketBuffer)
	tcp := getTcp(packet)
	if tcp == nil {
		return
	}
	var fragments *uniTCPStreamFragments
	if packet.Forward() {
		fragments = &f.forward
	} else {
		fragments = &f.backward
	}
	if !fragments.seen {
		if !tcp.FIN && !tcp.RST && tcp.SYN { //match SYN and SYN,ACK
			fragments.lastSequence = tcp.Seq
			fragments.firstSequence = tcp.Seq
			fragments.seen = true
			f.Emit(new, context, f)
		}
		return
	}
	if fragments.lastSequence == (tcp.Seq + 1) {
		// TCP keepalive -> ignore
		//fixme: include those?
		return
	}
	fragments.lastSequence = tcp.Seq
	position := tcp.Seq - (fragments.firstSequence + 1) // does wraparound work correctly?
	datalen := packet.Metadata().Length - packet.Hlen()
	if position == fragments.pos { // in order
		f.Emit(new, context, f)
		fragments.pos += uint32(datalen)
		for {
			nextpacket := fragments.popFragment(fragments.pos)
			if nextpacket == nil {
				return
			}
			f.Emit(nextpacket, context, f)
			fragments.pos += uint32(nextpacket.Metadata().Length - nextpacket.Hlen())
		}
	}
	// out of order
	if fragments.contains(position, datalen == 0) {
		return // ignore old or already seen packets
	}
	// ignore spurious fragments outside of window?
	fragments.pushFragment(position, packet)
}

func init() {
	flows.RegisterFunction("tcpReorder", flows.RawPacket, func() flows.Feature { return &tcpReorder{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type ipTTL struct {
	flows.BaseFeature
}

func (f *ipTTL) Event(new interface{}, context flows.EventContext, src interface{}) {
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
