package iana

import (
	"net"
	"testing"

	"github.com/CN-TU/go-flows/packet"
	"github.com/CN-TU/go-flows/packet_test"
	"github.com/google/gopacket/layers"
)

func TestSourceIPAddress(t *testing.T) {
	table := packet_test.MakeFlowFeatureTest(t, "sourceIPAddress")
	events := []packet.SerializableLayerType{
		&layers.IPv4{SrcIP: []byte{1, 2, 3, 4}, DstIP: []byte{1, 2, 3, 4}},
		&layers.IPv6{SrcIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, DstIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		&layers.IPv4{SrcIP: []byte{1, 2, 3, 4}, DstIP: []byte{1, 2, 3, 4}},
		&layers.IPv6{SrcIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, DstIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		&layers.IPv4{SrcIP: []byte{1, 2, 3, 5}, DstIP: []byte{1, 2, 3, 4}},
		&layers.IPv6{SrcIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}, DstIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
	}
	for _, event := range events {
		table.EventLayers(0, event)
	}
	table.Finish(0)
	table.AssertFeatureList([]packet_test.FeatureLine{
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceIPv4Address", Value: net.IP{1, 2, 3, 4}}}},
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceIPv6Address", Value: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}}},
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceIPv4Address", Value: net.IP{1, 2, 3, 5}}}},
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceIPv6Address", Value: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}}}},
	})
}

func TestSourceTransportPort(t *testing.T) {
	table := packet_test.MakeFlowFeatureTest(t, "sourceTransportPort")
	events := []packet.SerializableLayerType{
		&layers.UDP{SrcPort: 80, DstPort: 80},
		&layers.UDP{SrcPort: 80, DstPort: 80},
		&layers.TCP{SrcPort: 80, DstPort: 80},
		&layers.TCP{SrcPort: 80, DstPort: 80},
		&layers.UDP{SrcPort: 80, DstPort: 81},
		&layers.UDP{SrcPort: 81, DstPort: 80},
	}
	for _, event := range events {
		table.EventLayers(0, event)
	}
	table.Finish(0)
	table.AssertFeatureList([]packet_test.FeatureLine{
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceTransportPort", Value: uint16(80)}}}, //UDP 80:80
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceTransportPort", Value: uint16(80)}}}, //TCP 80:80
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceTransportPort", Value: uint16(80)}}}, //UDP 80:81
		{When: 0, Features: []packet_test.FeatureResult{{Name: "sourceTransportPort", Value: uint16(81)}}}, //UDP 81:80
	})
}

func TestDestinationTransportPort(t *testing.T) {
	table := packet_test.MakeFlowFeatureTest(t, "destinationTransportPort")
	events := []packet.SerializableLayerType{
		&layers.UDP{SrcPort: 80, DstPort: 80},
		&layers.UDP{SrcPort: 80, DstPort: 80},
		&layers.TCP{SrcPort: 80, DstPort: 80},
		&layers.TCP{SrcPort: 80, DstPort: 80},
		&layers.UDP{SrcPort: 80, DstPort: 81},
		&layers.UDP{SrcPort: 81, DstPort: 80},
	}
	for _, event := range events {
		table.EventLayers(0, event)
	}
	table.Finish(0)
	table.AssertFeatureList([]packet_test.FeatureLine{
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationTransportPort", Value: uint16(80)}}}, //UDP 80:80
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationTransportPort", Value: uint16(80)}}}, //TCP 80:80
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationTransportPort", Value: uint16(81)}}}, //UDP 80:81
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationTransportPort", Value: uint16(80)}}}, //UDP 81:80
	})
}
func TestDestinationIPAddress(t *testing.T) {
	table := packet_test.MakeFlowFeatureTest(t, "destinationIPAddress")
	events := []packet.SerializableLayerType{
		&layers.IPv4{DstIP: []byte{1, 2, 3, 4}, SrcIP: []byte{1, 2, 3, 4}},
		&layers.IPv6{DstIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, SrcIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		&layers.IPv4{DstIP: []byte{1, 2, 3, 4}, SrcIP: []byte{1, 2, 3, 4}},
		&layers.IPv6{DstIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, SrcIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		&layers.IPv4{DstIP: []byte{1, 2, 3, 5}, SrcIP: []byte{1, 2, 3, 4}},
		&layers.IPv6{DstIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}, SrcIP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
	}
	for _, event := range events {
		table.EventLayers(0, event)
	}
	table.Finish(0)
	table.AssertFeatureList([]packet_test.FeatureLine{
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationIPv4Address", Value: net.IP{1, 2, 3, 4}}}},
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationIPv6Address", Value: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}}},
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationIPv4Address", Value: net.IP{1, 2, 3, 5}}}},
		{When: 0, Features: []packet_test.FeatureResult{{Name: "destinationIPv6Address", Value: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}}}},
	})
}
