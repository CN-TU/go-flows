package features

import (
	"github.com/CN-TU/go-flows/packet"
	"github.com/google/gopacket/layers"
)

// GetTCP returns the TCP layer of the packet or nil
func GetTCP(new interface{}) *layers.TCP {
	tcp := new.(packet.Buffer).TransportLayer()
	if tcp == nil {
		return nil
	}
	packetTCP := tcp.(*layers.TCP)
	return packetTCP
}
