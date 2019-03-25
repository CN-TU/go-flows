package builtin

import "github.com/CN-TU/go-flows/packet"

func sourceMacAddressKey(packet packet.Buffer, scratch, scratchNoSort []byte) (int, int) {
	link := packet.LinkLayer()
	if link == nil {
		return 0, 0
	}
	return copy(scratch, link.LinkFlow().Src().Raw()), 0
}

func destinationMacAddressKey(packet packet.Buffer, scratch, scratchNoSort []byte) (int, int) {
	link := packet.LinkLayer()
	if link == nil {
		return 0, 0
	}
	return copy(scratch, link.LinkFlow().Dst().Raw()), 0
}

func init() {
	packet.RegisterKeyPair(
		packet.RegisterStringKey("sourceMacAddress",
			"source address of link layer",
			packet.KeyTypeSource, packet.KeyLayerLink, func(string) packet.KeyFunc { return sourceMacAddressKey }),
		packet.RegisterStringKey("destinationMacAddress",
			"destination address of link layer",
			packet.KeyTypeDestination, packet.KeyLayerLink, func(string) packet.KeyFunc { return destinationMacAddressKey }),
	)
}

////////////////////////////////////////////////////////////////////////////////

func ethernetTypeKey(packet packet.Buffer, scratch, scratchNoSort []byte) (int, int) {
	t := packet.EtherType()
	scratch[0] = byte(t >> 8)
	scratch[1] = byte(t & 0x00FF)
	return 2, 0
}

func init() {
	packet.RegisterStringKey("ethernetType",
		"protocol identifier of link layer",
		packet.KeyTypeUnidirectional, packet.KeyLayerLink, func(string) packet.KeyFunc { return ethernetTypeKey })
}

////////////////////////////////////////////////////////////////////////////////

func vlanKey(packet packet.Buffer, scratch, scratchNoSort []byte) (int, int) {
	t := packet.Dot1QLayers()
	for i := range t {
		vlan := t[i].VLANIdentifier
		scratch[i*2] = byte(vlan >> 8)
		scratch[i*2+1] = byte(vlan & 0x00FF)
	}
	return len(t) * 2, 0
}

func init() {
	packet.RegisterStringKey("vlanID",
		"vlan ID",
		packet.KeyTypeUnidirectional, packet.KeyLayerLink, func(string) packet.KeyFunc { return vlanKey })
}
