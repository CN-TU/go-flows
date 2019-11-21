package custom

import (
	"github.com/CN-TU/go-flows/flows"
	"github.com/CN-TU/go-flows/packet"
	ipfix "github.com/CN-TU/go-ipfix"
	"github.com/google/gopacket/layers"
)

// Gets LinuxSLL layer
func GetSLL(new interface{}) *layers.LinuxSLL {
	link := new.(packet.Buffer).LinkLayer()
	if link == nil {
		return nil
	}
	packetSLL, _ := link.(*layers.LinuxSLL)
	return packetSLL
}

type _sllAddr struct {
	flows.BaseFeature
}

func (f *_sllAddr) Event(new interface{}, context *flows.EventContext, src interface{}) {
	sll := GetSLL(new)
	if sll == nil {
		return
	}
	f.SetValue(string(sll.Addr), context, src)
}

func init() {
	flows.RegisterTemporaryFeature("_sllAddr", "returns address stored in linux coooked capture layer.", ipfix.StringType, 0, flows.PacketFeature, func() flows.Feature { return &_sllAddr{} }, flows.RawPacket)
}
