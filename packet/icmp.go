package packet

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var icmpEndpointType = gopacket.RegisterEndpointType(1000, gopacket.EndpointTypeMetadata{Name: "ICMP", Formatter: func(b []byte) string {
	return fmt.Sprintf("%d:%d", b[0], b[1])
}})

type ICMPv4Flow struct {
	layers.ICMPv4
}

func (i *ICMPv4Flow) TransportFlow() gopacket.Flow {
	return gopacket.NewFlow(icmpEndpointType, emptyPort, i.Contents[0:2])
}

type ICMPv6Flow struct {
	layers.ICMPv6
}

func (i *ICMPv6Flow) TransportFlow() gopacket.Flow {
	return gopacket.NewFlow(icmpEndpointType, emptyPort, i.Contents[0:2])
}
