package packet

import (
	"net"
	"testing"

	"github.com/google/gopacket/layers"
)

func TestSourceIPAddress(t *testing.T) {
	table := makeFlowFeatureTest(t, "sourceIPAddress")
	events := []SerializableLayerType{
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
	table.assertFeatureList([]featureLine{
		{0, []featureResult{{"sourceIPv4Address", net.IP{1, 2, 3, 4}}}},
		{0, []featureResult{{"sourceIPv6Address", net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}}},
		{0, []featureResult{{"sourceIPv4Address", net.IP{1, 2, 3, 5}}}},
		{0, []featureResult{{"sourceIPv6Address", net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}}}},
	})
}

func TestSourceTransportPort(t *testing.T) {
	table := makeFlowFeatureTest(t, "sourceTransportPort")
	events := []SerializableLayerType{
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
	table.assertFeatureList([]featureLine{
		{0, []featureResult{{"sourceTransportPort", uint16(80)}}}, //UDP 80:80
		{0, []featureResult{{"sourceTransportPort", uint16(80)}}}, //TCP 80:80
		{0, []featureResult{{"sourceTransportPort", uint16(80)}}}, //UDP 80:81
		{0, []featureResult{{"sourceTransportPort", uint16(81)}}}, //UDP 81:80
	})
}

func TestDestinationTransportPort(t *testing.T) {
	table := makeFlowFeatureTest(t, "destinationTransportPort")
	events := []SerializableLayerType{
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
	table.assertFeatureList([]featureLine{
		{0, []featureResult{{"destinationTransportPort", uint16(80)}}}, //UDP 80:80
		{0, []featureResult{{"destinationTransportPort", uint16(80)}}}, //TCP 80:80
		{0, []featureResult{{"destinationTransportPort", uint16(81)}}}, //UDP 80:81
		{0, []featureResult{{"destinationTransportPort", uint16(80)}}}, //UDP 81:80
	})
}
func TestDestinationIPAddress(t *testing.T) {
	table := makeFlowFeatureTest(t, "destinationIPAddress")
	events := []SerializableLayerType{
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
	table.assertFeatureList([]featureLine{
		{0, []featureResult{{"destinationIPv4Address", net.IP{1, 2, 3, 4}}}},
		{0, []featureResult{{"destinationIPv6Address", net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}}},
		{0, []featureResult{{"destinationIPv4Address", net.IP{1, 2, 3, 5}}}},
		{0, []featureResult{{"destinationIPv6Address", net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}}}},
	})
}
