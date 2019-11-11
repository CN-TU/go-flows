package custom

import (
	"strings"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	ipfix "github.com/CN-TU/go-ipfix"
	"github.com/google/gopacket/layers"
)

type _payload struct {
	flows.BaseFeature
}

func (f *_payload) Event(new interface{}, context *flows.EventContext, src interface{}) {
	tl := new.(packet.Buffer).TransportLayer()
	if tl == nil {
		return
	}
	payload := tl.LayerPayload()
	if len(payload) == 0 {
		return
	}
	f.SetValue(payload, context, f)
}

func init() {
	flows.RegisterTemporaryFeature("_payload", "application layer of a packet", ipfix.OctetArrayType, 0, flows.PacketFeature, func() flows.Feature { return &_payload{} }, flows.RawPacket)
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
	f.BaseFeature.Start(context)
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
	flows.RegisterTemporaryFeature("httpLines", "returns headers from a http session", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_HTTPLines{} }, flows.PacketFeature)
	flows.RegisterTemporaryCompositeFeature("_HTTPLines", "returns headers from a http session", ipfix.StringType, 0, "httpLines", "_tcpReorderPayload")
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
	flows.RegisterTemporaryFeature("__httpRequestHost", "extract the host header from lines of text", ipfix.StringType, 0, flows.FlowFeature, func() flows.Feature { return &httpRequestHost{} }, flows.PacketFeature)
	flows.RegisterStandardCompositeFeature("httpRequestHost", "__httpRequestHost", "_HTTPLines")
}

////////////////////////////////////////////////////////////////////////////////

type _DNSDomain struct {
	flows.BaseFeature
}

func (f *_DNSDomain) Start(context *flows.EventContext) {
	f.BaseFeature.Start(context)
}

func (f *_DNSDomain) Event(new interface{}, context *flows.EventContext, src interface{}) {
	layer := new.(packet.Buffer).Layer(layers.LayerTypeDNS)
	if layer != nil {
		dns := layer.(*layers.DNS)
		for _, dnsQuestion := range dns.Questions {
			f.SetValue(string(dnsQuestion.Name), context, src)
		}
	}
}

func (f *_DNSDomain) Stop(reason flows.FlowEndReason, context *flows.EventContext) {
}

func init() {
	flows.RegisterTemporaryFeature("_DNSDomain", "returns domains from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_DNSDomain{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
