package packet

import (
	"testing"

	"github.com/google/gopacket/layers"
)

func BenchmarkDynamicFiveTuple4(b *testing.B) {
	buffer4 := BufferFromLayers(0,
		&layers.IPv4{SrcIP: []byte{1, 2, 3, 4}, DstIP: []byte{1, 2, 3, 4}, Protocol: layers.IPProtocolTCP},
		&layers.TCP{SrcPort: 80, DstPort: 80},
	)
	key := MakeDynamicKeySelector(
		[]string{"sourceIPAddress",
			"destinationIPAddress",
			"protocolIdentifier",
			"sourceTransportPort",
			"destinationTransportPort"}, true, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.Key(buffer4)
	}
}

func BenchmarkDynamicFiveTuple6(b *testing.B) {
	buffer6 := BufferFromLayers(0,
		&layers.IPv6{SrcIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, DstIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, NextHeader: layers.IPProtocolTCP},
		&layers.TCP{SrcPort: 80, DstPort: 80},
	)
	key := MakeDynamicKeySelector(
		[]string{"sourceIPAddress",
			"destinationIPAddress",
			"protocolIdentifier",
			"sourceTransportPort",
			"destinationTransportPort"}, true, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.Key(buffer6)
	}
}
