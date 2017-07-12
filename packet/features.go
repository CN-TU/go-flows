package packet

import (
	"net"

	"encoding/binary"

	"github.com/google/gopacket/layers"
	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

////////////////////////////////////////////////////////////////////////////////

type sourceIPAddress struct {
	flows.BaseFeature
}

func (f *sourceIPAddress) Event(new interface{}, when flows.Time, src interface{}) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().SrcIP()), when, f)
	}
}

func (f *sourceIPAddress) Type() string {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return "sourceIPv4Address"
	}
	return "sourceIPv6Address"
}

func init() {
	flows.RegisterFeature("sourceIPAddress", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &sourceIPAddress{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type sourceTransportPort struct {
	flows.BaseFeature
}

func (f *sourceTransportPort) Event(new interface{}, when flows.Time, src interface{}) {
	if f.Value() == nil {
		f.SetValue(binary.BigEndian.Uint16(f.Key().SrcPort()), when, f)
	}
}

func init() {
	flows.RegisterFeature("sourceTransportPort", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &sourceTransportPort{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type destinationTransportPort struct {
	flows.BaseFeature
}

func (f *destinationTransportPort) Event(new interface{}, when flows.Time, src interface{}) {
	if f.Value() == nil {
		f.SetValue(binary.BigEndian.Uint16(f.Key().DstPort()), when, f)
	}
}

func init() {
	flows.RegisterFeature("destinationTransportPort", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &destinationTransportPort{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type destinationIPAddress struct {
	flows.BaseFeature
}

func (f *destinationIPAddress) Event(new interface{}, when flows.Time, src interface{}) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().DstIP()), when, f)
	}
}

func (f *destinationIPAddress) Type() string {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return "destinationIPv4Address"
	}
	return "destinationIPv6Address"
}

func init() {
	flows.RegisterFeature("destinationIPAddress", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &destinationIPAddress{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type protocolIdentifier struct {
	flows.BaseFeature
}

func (f *protocolIdentifier) Event(new interface{}, when flows.Time, src interface{}) {
	if f.Value() == nil {
		f.SetValue(f.Key().Proto(), when, f)
	}
}

func init() {
	flows.RegisterFeature("protocolIdentifier", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &protocolIdentifier{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type flowEndReason struct {
	flows.BaseFeature
}

func (f *flowEndReason) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(reason, when, f)
}

func init() {
	flows.RegisterFeature("flowEndReason", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &flowEndReason{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type flowEndNanoSeconds struct {
	flows.BaseFeature
}

func (f *flowEndNanoSeconds) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(when, when, f)
}

func init() {
	flows.RegisterFeature("flowEndNanoSeconds", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &flowEndNanoSeconds{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type octetTotalCountPacket struct {
	flows.BaseFeature
}

func octetCount(packet *packetBuffer) flows.Unsigned64 {
	length := packet.Metadata().Length
	if net := packet.LinkLayer(); net != nil {
		length -= len(net.LayerContents())
	}
	return flows.Unsigned64(length)
}

func (f *octetTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	f.SetValue(octetCount(new.(*packetBuffer)), when, f)
}

type octetTotalCountFlow struct {
	flows.BaseFeature
	total flows.Unsigned64
}

func (f *octetTotalCountFlow) Start(when flows.Time) {
	f.total = 0
}

func (f *octetTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	f.total = f.total.Add(octetCount(new.(*packetBuffer))).(flows.Unsigned64)
}

func (f *octetTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.total, when, f)
}

func init() {
	flows.RegisterFeature("octetTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &octetTotalCountFlow{} }, nil},
		{flows.FeatureTypePacket, func() flows.Feature { return &octetTotalCountPacket{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

func ipTotalLength(packet *packetBuffer) flows.Unsigned64 {
	network := packet.NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		return flows.Unsigned64(ip.Length)
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
					return flows.Unsigned64(l)
				}
			}
		}
		return flows.Unsigned64(ip.Length)
	}
	return 0
}

type ipTotalLengthPacket struct {
	flows.BaseFeature
}

func (f *ipTotalLengthPacket) Event(new interface{}, when flows.Time, src interface{}) {
	f.SetValue(ipTotalLength(new.(*packetBuffer)), when, f)
}

type ipTotalLengthFlow struct {
	flows.BaseFeature
	total flows.Unsigned64
}

func (f *ipTotalLengthFlow) Start(when flows.Time) {
	f.total = 0
}

func (f *ipTotalLengthFlow) Event(new interface{}, when flows.Time, src interface{}) {
	f.total = f.total.Add(ipTotalLength(new.(*packetBuffer))).(flows.Unsigned64)
}

func (f *ipTotalLengthFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.total, when, f)
}

func init() {
	flows.RegisterFeature("ipTotalLength", []flows.FeatureCreator{
		{flows.FeatureTypeFlow, func() flows.Feature { return &ipTotalLengthFlow{} }, nil},
		{flows.FeatureTypePacket, func() flows.Feature { return &ipTotalLengthPacket{} }, nil},
	})
	flows.RegisterCompositeFeature("minimumIpTotalLength", []interface{}{"min", "ipTotalLength"})
	flows.RegisterCompositeFeature("maximumIpTotalLength", []interface{}{"max", "ipTotalLength"})
}

////////////////////////////////////////////////////////////////////////////////

type tcpControlBits struct {
	flows.BaseFeature
}

func getTcp(packet *packetBuffer) *layers.TCP {
	tcp := packet.Layer(layers.LayerTypeTCP)
	if tcp == nil {
		return nil
	}
	packet_tcp := tcp.(*layers.TCP)
	return packet_tcp
}

func (f *tcpControlBits) Event(new interface{}, when flows.Time, src interface{}) {
	var value uint16
	tcp := getTcp(new.(*packetBuffer))
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
	f.SetValue(value, when, f)
}

func init() {
	flows.RegisterFeature("tcpControlBits", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &tcpControlBits{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type _intraPacketTimeNanoseconds struct {
	flows.BaseFeature
	time int64
}

func (f *_intraPacketTimeNanoseconds) Event(new interface{}, when flows.Time, src interface{}) {
	var time int64
	if f.time != 0 {
		time = int64(when) - f.time
	}
	f.time = int64(when)
	f.SetValue(time, when, f)
}

func init() {
	flows.RegisterFeature("_intraPacketTimeNanoseconds", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &_intraPacketTimeNanoseconds{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type _intraPacketTimeMilliseconds struct {
	flows.BaseFeature
	time int64
}

func (f *_intraPacketTimeMilliseconds) Event(new interface{}, when flows.Time, src interface{}) {
	var time int64
	if f.time != 0 {
		time = int64(when) - f.time
	}
	f.time = int64(when)
	new_time := float64(time) / 1000000.
	f.SetValue(new_time, when, f)
}

func init() {
	flows.RegisterFeature("_intraPacketTimeMilliseconds", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &_intraPacketTimeMilliseconds{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type join struct {
	flows.MultiBaseFeature
}

func (f *join) Event(new interface{}, when flows.Time, src interface{}) {
	values := f.EventResult(new, src)
	if values == nil {
		return
	}
	f.SetValue(values, when, f)
}

func init() {
	flows.RegisterFeature("join", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &join{} }, []flows.FeatureType{flows.FeatureTypePacket}},
		{flows.FeatureTypePacket, func() flows.Feature { return &join{} }, []flows.FeatureType{flows.FeatureTypePacket, flows.FeatureTypeEllipsis}},
		{flows.FeatureTypeFlow, func() flows.Feature { return &join{} }, []flows.FeatureType{flows.FeatureTypeFlow}},
		{flows.FeatureTypeFlow, func() flows.Feature { return &join{} }, []flows.FeatureType{flows.FeatureTypeFlow, flows.FeatureTypeEllipsis}},
	})
}

////////////////////////////////////////////////////////////////////////////////

type tcpSynTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *tcpSynTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *tcpSynTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func boolInt(b bool) flows.Unsigned64 {
	if b {
		return 1
	} else {
		return 0
	}
}

func (f *tcpSynTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.SYN)).(flows.Unsigned64)
}

type tcpSynTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpSynTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.SYN), when, f)
}

func init() {
	flows.RegisterFeature("tcpSynTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &tcpSynTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &tcpSynTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type tcpFinTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *tcpFinTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *tcpFinTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *tcpFinTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.FIN)).(flows.Unsigned64)
}

type tcpFinTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpFinTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.FIN), when, f)
}

func init() {
	flows.RegisterFeature("tcpFinTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &tcpFinTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &tcpFinTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type tcpRstTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *tcpRstTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *tcpRstTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *tcpRstTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.RST)).(flows.Unsigned64)
}

type tcpRstTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpRstTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.RST), when, f)
}

func init() {
	flows.RegisterFeature("tcpRstTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &tcpRstTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &tcpRstTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type tcpPshTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *tcpPshTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *tcpPshTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *tcpPshTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.PSH)).(flows.Unsigned64)
}

type tcpPshTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpPshTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.PSH), when, f)
}

func init() {
	flows.RegisterFeature("tcpPshTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &tcpPshTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &tcpPshTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type tcpAckTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *tcpAckTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *tcpAckTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *tcpAckTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.ACK)).(flows.Unsigned64)
}

type tcpAckTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpAckTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.ACK), when, f)
}

func init() {
	flows.RegisterFeature("tcpAckTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &tcpAckTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &tcpAckTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type tcpUrgTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *tcpUrgTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *tcpUrgTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *tcpUrgTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.URG)).(flows.Unsigned64)
}

type tcpUrgTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpUrgTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.URG), when, f)
}

func init() {
	flows.RegisterFeature("tcpUrgTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &tcpUrgTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &tcpUrgTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type _tcpEceTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *_tcpEceTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *_tcpEceTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *_tcpEceTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.ECE)).(flows.Unsigned64)
}

type _tcpEceTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpEceTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.ECE), when, f)
}

func init() {
	flows.RegisterFeature("_tcpEceTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &_tcpEceTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &_tcpEceTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type _tcpCwrTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *_tcpCwrTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *_tcpCwrTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *_tcpCwrTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.CWR)).(flows.Unsigned64)
}

type _tcpCwrTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpCwrTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.CWR), when, f)
}

func init() {
	flows.RegisterFeature("_tcpCwrTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &_tcpCwrTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &_tcpCwrTotalCountFlow{} }, nil},
	})
}

////////////////////////////////////////////////////////////////////////////////

type _tcpNsTotalCountFlow struct {
	flows.BaseFeature
	count flows.Unsigned64
}

func (f *_tcpNsTotalCountFlow) Start(when flows.Time) {
	f.count = 0
}

func (f *_tcpNsTotalCountFlow) Stop(reason flows.FlowEndReason, when flows.Time) {
	f.SetValue(f.count, when, f)
}

func (f *_tcpNsTotalCountFlow) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.count = f.count.Add(boolInt(tcp.NS)).(flows.Unsigned64)
}

type _tcpNsTotalCountPacket struct {
	flows.BaseFeature
}

func (f *_tcpNsTotalCountPacket) Event(new interface{}, when flows.Time, src interface{}) {
	tcp := getTcp(new.(*packetBuffer))
	if tcp == nil {
		return
	}
	f.SetValue(boolInt(tcp.NS), when, f)
}

func init() {
	flows.RegisterFeature("_tcpNsTotalCount", []flows.FeatureCreator{
		{flows.FeatureTypePacket, func() flows.Feature { return &_tcpNsTotalCountPacket{} }, nil},
		{flows.FeatureTypeFlow, func() flows.Feature { return &_tcpNsTotalCountFlow{} }, nil},
	})
}
