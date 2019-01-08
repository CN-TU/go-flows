package nta

import (
	"encoding/binary"
	"fmt"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	ipfix "github.com/CN-TU/go-ipfix"
	"github.com/google/gopacket/layers"
)

type flowID struct {
	flows.BaseFeature
}

func (f *flowID) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() == nil {
		nl := new.(packet.Buffer).NetworkLayer()
		if nl == nil {
			return
		}
		flow := nl.NetworkFlow()
		f.SetValue(fmt.Sprintf("%s > %s", flow.Src(), flow.Dst()), context, f)
	}
}

func init() {
	flows.RegisterTemporaryFeature("__NTAFlowID", "src address > dst address", ipfix.StringType, 0, flows.FlowFeature, func() flows.Feature { return &flowID{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type protocol struct {
	flows.BaseFeature
}

func (f *protocol) Event(new interface{}, context *flows.EventContext, src interface{}) {
	v := f.Value()
	if v == -1 {
		return
	}
	p := new.(packet.Buffer).Proto()
	if v == nil {
		f.SetValue(p, context, f)
		return
	}
	if v != p {
		f.SetValue(-1, context, f)
	}
}

func init() {
	flows.RegisterTemporaryFeature("__NTAProtocol", "protocol or -1 if multiple protocols", ipfix.Signed16Type, 0, flows.FlowFeature, func() flows.Feature { return &protocol{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type ports struct {
	flows.BaseFeature
	src  uint16
	dst  uint16
	init bool
}

func (f *ports) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.src = 0
	f.dst = 0
	f.init = false
}

func (f *ports) Event(new interface{}, context *flows.EventContext, src interface{}) {
	if f.Value() != nil {
		return
	}
	transport := new.(packet.Buffer).TransportLayer()
	if transport == nil {
		return
	}
	flow := transport.TransportFlow()
	srcRaw := flow.Src().Raw()
	dstRaw := flow.Dst().Raw()
	// assume that both are non nil if one is
	if srcRaw == nil {
		return
	}
	s := binary.BigEndian.Uint16(srcRaw)
	dst := binary.BigEndian.Uint16(dstRaw)
	if !f.init {
		f.src = s
		f.dst = dst
		return
	}
	if f.src != s {
		f.SetValue("-1", context, f)
		return
	}
	if f.dst != dst {
		f.SetValue("-1", context, f)
		return
	}
}

func (f *ports) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if f.Value() == nil {
		f.SetValue(fmt.Sprintf("%d:%d", f.src, f.dst), context, f)
	}
}

func init() {
	flows.RegisterTemporaryFeature("__NTAPorts", "srcport:dstport or -1 if multiple ports", ipfix.StringType, 0, flows.FlowFeature, func() flows.Feature { return &ports{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type tdata struct {
	flows.BaseFeature
	bytes uint64
}

func (f *tdata) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.bytes = 0
}

func (f *tdata) Event(new interface{}, context *flows.EventContext, src interface{}) {
	buffer := new.(packet.Buffer)
	f.bytes += uint64(buffer.PayloadLength())
	tl := buffer.TransportLayer()
	if tl != nil {
		tlt := tl.LayerType()
		if tlt != layers.LayerTypeTCP && tlt != layers.LayerTypeUDP {
			// add size of transport layer header if non-TCP and non-UDP
			f.bytes += uint64(len(tl.LayerContents()))
		}
	}
}

func (f *tdata) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if f.Value() == nil {
		f.SetValue(f.bytes, context, f)
	}
}

func init() {
	flows.RegisterTemporaryFeature("__NTATData", "sum of packet lengths (=payload length + transport header if nontcp/nonudp)", ipfix.Unsigned64Type, 0, flows.FlowFeature, func() flows.Feature { return &tdata{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type secwindow struct {
	flows.BaseFeature
}

func (f *secwindow) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tsec := uint32(new.(packet.Buffer).Timestamp() / flows.SecondsInNanoseconds)
	if f.Value() != tsec {
		f.SetValue(tsec, context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("__NTASecWindow", "1s time window for packets", ipfix.Unsigned64Type, 0, flows.Selection, func() flows.Feature { return &secwindow{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type ton struct {
	flows.BaseFeature
	last uint32
	ton  uint32
}

func (f *ton) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.last = 0
	f.ton = 0
}

func (f *ton) Event(new interface{}, context *flows.EventContext, src interface{}) {
	t := new.(uint32)
	if f.ton == 0 {
		f.last = t
		f.ton = 1
		return
	}
	if t == f.last+1 {
		f.last = t
		f.ton++
		return
	}
	f.SetValue(f.ton, context, f)
	f.last = t
	f.ton = 1
}

func (f *ton) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
	if f.ton != 0 {
		f.SetValue(f.ton, context, f)
	}
}

func init() {
	flows.RegisterTemporaryFeature("__NTATOn", "returns on time (consec. seconds with packet)", ipfix.Unsigned64Type, 0, flows.PacketFeature, func() flows.Feature { return &ton{} }, flows.Selection)
	//	flows.RegisterTemporaryFeature("__NTATOn", "returns on time (consec. seconds with packet)", ipfix.Unsigned64Type, 0, flows.Selection, func() flows.Feature { return &ton{} }, flows.Selection)
}

////////////////////////////////////////////////////////////////////////////////

type toff struct {
	flows.BaseFeature
	last uint32
	ton  uint32
}

func (f *toff) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
	f.last = 0
}

func (f *toff) Event(new interface{}, context *flows.EventContext, src interface{}) {
	t := new.(uint32)
	if f.last == 0 {
		f.last = t
		return
	}
	if t == f.last+1 {
		f.last = t
		return
	}
	f.SetValue(t-f.last-1, context, f)
	f.last = t
}

func init() {
	flows.RegisterTemporaryFeature("__NTATOff", "returns off time (consec. seconds without packet)", ipfix.Unsigned64Type, 0, flows.PacketFeature, func() flows.Feature { return &toff{} }, flows.Selection)
}

////////////////////////////////////////////////////////////////////////////////
