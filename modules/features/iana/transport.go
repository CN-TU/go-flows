package iana

import (
	"encoding/binary"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/modules/features"
	"github.com/CN-TU/go-flows/packet"
)

type sourceTransportPort struct {
	flows.BaseFeature
}

func (f *sourceTransportPort) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		transport := new.(packet.Buffer).TransportLayer()
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
		transport := new.(packet.Buffer).TransportLayer()
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

type tcpControlBits struct {
	flows.BaseFeature
}

func (f *tcpControlBits) Event(new interface{}, context *flows.EventContext, src interface{}) {
	var value uint16
	tcp := features.GetTCP(new)
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

type tcpSynTotalCountFlow struct {
	flows.BaseFeature
	count uint64
}

func (f *tcpSynTotalCountFlow) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *tcpSynTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpSynTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.SYN)
}

func init() {
	flows.RegisterStandardFeature("tcpSynTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpSynTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpSynTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpSynTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.SYN), context, f)
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
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *tcpFinTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpFinTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.FIN)
}

func init() {
	flows.RegisterStandardFeature("tcpFinTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpFinTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpFinTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpFinTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.FIN), context, f)
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
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *tcpRstTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpRstTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.RST)
}

func init() {
	flows.RegisterStandardFeature("tcpRstTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpRstTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpRstTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpRstTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.RST), context, f)
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
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *tcpPshTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpPshTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.PSH)
}

func init() {
	flows.RegisterStandardFeature("tcpPshTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpPshTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpPshTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpPshTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.PSH), context, f)
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
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *tcpAckTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpAckTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.ACK)
}

func init() {
	flows.RegisterStandardFeature("tcpAckTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpAckTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
type tcpAckTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpAckTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.ACK), context, f)
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
	f.BaseFeature.Start(context)
	f.count = 0
}

func (f *tcpUrgTotalCountFlow) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	f.SetValue(f.count, context, f)
}

func (f *tcpUrgTotalCountFlow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.count += features.BoolInt(tcp.URG)
}

func init() {
	flows.RegisterStandardFeature("tcpUrgTotalCount", flows.FlowFeature, func() flows.Feature { return &tcpUrgTotalCountFlow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpUrgTotalCountPacket struct {
	flows.BaseFeature
}

func (f *tcpUrgTotalCountPacket) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tcp := features.GetTCP(new)
	if tcp == nil {
		return
	}
	f.SetValue(features.BoolInt(tcp.URG), context, f)
}

func init() {
	flows.RegisterStandardFeature("tcpUrgTotalCount", flows.PacketFeature, func() flows.Feature { return &tcpUrgTotalCountPacket{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tcpSequenceNumber struct {
	flows.BaseFeature
	isn    features.Sequence
	cutoff bool
}

func (f *tcpSequenceNumber) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.isn = features.InvalidSequence
	f.cutoff = false
}

func (f *tcpSequenceNumber) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if f.isn != features.InvalidSequence {
		f.SetValue(uint32(f.isn), context, f)
	}
}

func (f *tcpSequenceNumber) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if !new.(packet.Buffer).Forward() {
		return
	}
	tcp := features.GetTCP(new)
	if tcp == nil || f.cutoff {
		return
	}
	seq := features.Sequence(tcp.Seq)
	if f.isn == features.InvalidSequence {
		// no sequence number recorded
		f.isn = seq
		return
	}
	if f.isn.Add(0x40000000).Difference(seq) > 0 {
		// stop searching for out of order packets if we went through lots of data...
		// otherwise search for out of order would be impossible due to wraparound of sequence number
		f.cutoff = true
		return
	}
	if f.isn.Difference(seq) < 0 {
		// sequence number is before the recorded one (out of order packet)
		f.isn = seq
	}
}

type reverseTCPSequenceNumber struct {
	flows.BaseFeature
	isn    features.Sequence
	cutoff bool
}

func (f *reverseTCPSequenceNumber) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.isn = features.InvalidSequence
	f.cutoff = false
}

func (f *reverseTCPSequenceNumber) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if f.isn != features.InvalidSequence {
		f.SetValue(uint32(f.isn), context, f)
	}
}

func (f *reverseTCPSequenceNumber) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if new.(packet.Buffer).Forward() {
		return
	}
	tcp := features.GetTCP(new)
	if tcp == nil || f.cutoff {
		return
	}
	seq := features.Sequence(tcp.Seq)
	if f.isn == features.InvalidSequence {
		// no sequence number recorded
		f.isn = seq
		return
	}
	if f.isn.Add(0x40000000).Difference(seq) > 0 {
		// stop searching for out of order packets if we went through lots of data...
		// otherwise search for out of order would be impossible due to wraparound of sequence number
		f.cutoff = true
		return
	}
	if f.isn.Difference(seq) < 0 {
		// sequence number is before the recorded one (out of order packet)
		f.isn = seq
	}
}

func init() {
	flows.RegisterStandardFeature("tcpSequenceNumber", flows.FlowFeature, func() flows.Feature { return &tcpSequenceNumber{} }, flows.RawPacket)
	flows.RegisterStandardReverseFeature("tcpSequenceNumber", flows.FlowFeature, func() flows.Feature { return &reverseTCPSequenceNumber{} }, flows.RawPacket)
}
