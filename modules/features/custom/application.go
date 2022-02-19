package custom

import (
	"strings"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	ipfix "github.com/CN-TU/go-ipfix"
	"github.com/google/gopacket"
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

// GetDNS returns the DNS layer of the packet or nil
func GetDNS(new interface{}) *layers.DNS {
	tl := new.(packet.Buffer).TransportLayer()
	if tl != nil {
		payload := tl.LayerPayload()
		if tl.LayerType() == layers.LayerTypeTCP {
			// when TCP, omit DNS length field (otherwise DNS parser fails)
			if len(payload) > 2 {
				payload = payload[2:]
			} else {
				return nil
			}
		}
		dns := &layers.DNS{}
		err := dns.DecodeFromBytes(payload, gopacket.NilDecodeFeedback)
		if err == nil {
			return dns
		}
	}
	return nil
}

type _dnsDomain struct {
	flows.BaseFeature
}

func (f *_dnsDomain) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsQuestion := range dns.Questions {
			f.SetValue(string(dnsQuestion.Name), context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsDomain", "returns domains from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsDomain{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsID struct {
	flows.BaseFeature
}

func (f *_dnsID) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		f.SetValue(dns.ID, context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsID", "returns ID of DNS query.", ipfix.Unsigned16Type, 0, flows.PacketFeature, func() flows.Feature { return &_dnsID{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsQR struct {
	flows.BaseFeature
}

func (f *_dnsQR) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		f.SetValue(dns.QR, context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsQR", "returns QR of DNS query.", ipfix.BooleanType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsQR{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsResponseCode struct {
	flows.BaseFeature
}

func (f *_dnsResponseCode) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		f.SetValue(dns.ResponseCode.String(), context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsResponseCode", "returns response code from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsResponseCode{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsQDCount struct {
	flows.BaseFeature
}

func (f *_dnsQDCount) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		f.SetValue(dns.QDCount, context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsQDCount", "returns number of questions in DNS packets.", ipfix.Unsigned16Type, 0, flows.PacketFeature, func() flows.Feature { return &_dnsQDCount{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsANCount struct {
	flows.BaseFeature
}

func (f *_dnsANCount) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		f.SetValue(dns.ANCount, context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsANCount", "returns number of answers in DNS packets.", ipfix.Unsigned16Type, 0, flows.PacketFeature, func() flows.Feature { return &_dnsANCount{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsNSCount struct {
	flows.BaseFeature
}

func (f *_dnsNSCount) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		f.SetValue(dns.NSCount, context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsNSCount", "returns number of authorities in DNS packets.", ipfix.Unsigned16Type, 0, flows.PacketFeature, func() flows.Feature { return &_dnsNSCount{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsARCount struct {
	flows.BaseFeature
}

func (f *_dnsARCount) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		f.SetValue(dns.ARCount, context, src)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsARCount", "returns number of additional records in DNS packets.", ipfix.Unsigned16Type, 0, flows.PacketFeature, func() flows.Feature { return &_dnsARCount{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsType struct {
	flows.BaseFeature
}

func (f *_dnsType) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsQuestion := range dns.Questions {
			f.SetValue(dnsQuestion.Type.String(), context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsType", "returns request types from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsType{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsClass struct {
	flows.BaseFeature
}

func (f *_dnsClass) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsQuestion := range dns.Questions {
			f.SetValue(dnsQuestion.Class.String(), context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsClass", "returns request classes from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsClass{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsAnsName struct {
	flows.BaseFeature
}

func (f *_dnsAnsName) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsAnswer := range dns.Answers {
			f.SetValue(string(dnsAnswer.Name), context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsAnsName", "returns answer domain names from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsAnsName{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsAnsType struct {
	flows.BaseFeature
}

func (f *_dnsAnsType) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsAnswer := range dns.Answers {
			f.SetValue(dnsAnswer.Type.String(), context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsAnsType", "returns answer DNS type from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsAnsType{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsAnsClass struct {
	flows.BaseFeature
}

func (f *_dnsAnsClass) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsAnswer := range dns.Answers {
			f.SetValue(dnsAnswer.Class.String(), context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsAnsClass", "returns answer DNS class from DNS packets.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_dnsAnsClass{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsAnsTTL struct {
	flows.BaseFeature
}

func (f *_dnsAnsTTL) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsAnswer := range dns.Answers {
			f.SetValue(dnsAnswer.TTL, context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsAnsTTL", "returns answer TTL from DNS packets.", ipfix.Unsigned32Type, 0, flows.PacketFeature, func() flows.Feature { return &_dnsAnsTTL{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////

type _dnsAnsDataLength struct {
	flows.BaseFeature
}

func (f *_dnsAnsDataLength) Event(new interface{}, context *flows.EventContext, src interface{}) {
	dns := GetDNS(new)
	if dns != nil {
		for _, dnsAnswer := range dns.Answers {
			f.SetValue(dnsAnswer.DataLength, context, src)
		}
	}
}

func init() {
	flows.RegisterTemporaryFeature("_dnsAnsDataLength", "returns length of data in answers from DNS packets.", ipfix.Unsigned16Type, 0, flows.PacketFeature, func() flows.Feature { return &_dnsAnsDataLength{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////
