package packet

import (
	"net"

	"pm.cn.tuwien.ac.at/ipfix/go-flows/flows"
)

////////////////////////////////////////////////////////////////////////////////
type sourceIPAddress struct {
	flows.BaseFeature
}

func (f *sourceIPAddress) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().SrcIP()), when)
	}
}

func (f *sourceIPAddress) Type() string {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return "sourceIPv4Address"
	}
	return "sourceIPv6Address"
}

var SourceIPAdress = flows.RegisterFeature("sourceIPAddress", func() flows.Feature { return &sourceIPAddress{} })

////////////////////////////////////////////////////////////////////////////////
type destinationIPAddress struct {
	flows.BaseFeature
}

func (f *destinationIPAddress) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(net.IP(f.Key().DstIP()), when)
	}
}

func (f *destinationIPAddress) Type() string {
	val := f.Value()
	if val == nil || len(val.(net.IP)) == 4 {
		return "destinationIPv4Address"
	}
	return "destinationIPv6Address"
}

var DestinationIPAdress = flows.RegisterFeature("destinationIPAddress", func() flows.Feature { return &destinationIPAddress{} })

////////////////////////////////////////////////////////////////////////////////
type protocolIdentifier struct {
	flows.BaseFeature
}

func (f *protocolIdentifier) Event(new interface{}, when flows.Time) {
	if f.Value() == nil {
		f.SetValue(f.Key().Proto(), when)
	}
}

var ProtocolIdentifier = flows.RegisterFeature("protocolIdentifier", func() flows.Feature { return &protocolIdentifier{} })
