package custom

import (
	"github.com/google/gopacket/layers"

	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	ipfix "github.com/CN-TU/go-ipfix"
)

////////////////////////////////////////////////////////////////////////////////

type _ipChecksum struct {
	flows.BaseFeature
}

func (f *_ipChecksum) Event(new interface{}, context *flows.EventContext, src interface{}) {
	network := new.(packet.Buffer).NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		f.SetValue(ip.Checksum, context, f)
	}
}

func init() {
	flows.RegisterTemporaryFeature("_ipChecksum", "returns a textual representation of the ipchecksum", ipfix.StringType, 1, flows.PacketFeature, func() flows.Feature { return &_ipChecksum{} }, flows.RawPacket)
}

////////////////////////////////////////////////////////////////////////////////