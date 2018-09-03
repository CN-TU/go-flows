package packet

import (
	"bytes"
	"fmt"
)

var scratchKey [1024]byte
var emptyKey string

/* According to curated data files we have:
- destinationIPv4Address
- sourceIPv4Address
- protocolIdentifier
- sourceTransportPort
- destinationTransportPort
- ipClassOfService
- ingressPhysicalInterface
- octetTotalCount

- flowStartSeconds <- this does not make sense
*/

// MakeDynamicKeySelector creates a selector function from a dynamic key definition
func MakeDynamicKeySelector(key []string, bidirectional, allowZero bool) (ret DynamicKeySelector) {
	for _, key := range key {
		switch key {
		case "sourceIPv4Address", "sourceIPv6Address", "sourceIPAddress":
			ret.srcIP = true
			ret.network = true
		case "destinationIPv4Address", "destinationIPv6Address", "destinationIPAddress":
			ret.dstIP = true
			ret.network = true
		case "protocolIdentifier":
			ret.protocolIdentifier = true
			ret.network = true
		case "sourceTransportPort":
			ret.srcPort = true
			ret.transport = true
		case "destinationTransportPort":
			ret.dstPort = true
			ret.transport = true
		case "sourceMacAddress":
			ret.srcMac = true
			ret.link = true
		case "destinationMacAddress":
			ret.dstMac = true
			ret.link = true
		case "ethernetType":
			ret.etherType = true
			ret.link = true
		default:
			panic(fmt.Sprintf("Unknown key_feature '%s'", key))
		}
	}
	ret.bidirectional = bidirectional
	ret.fivetuple = ret.srcIP && ret.dstIP && ret.protocolIdentifier && ret.srcPort && ret.dstPort && bidirectional
	ret.empty = !ret.network && !ret.transport
	ret.allowZero = allowZero
	return
}

// DynamicKeySelector holds the definition for a flow key function
type DynamicKeySelector struct {
	allowZero,
	link,
	network,
	transport,
	srcMac,
	dstMac,
	etherType,
	srcIP,
	dstIP,
	protocolIdentifier,
	srcPort,
	dstPort,
	bidirectional,
	fivetuple,
	empty bool
}

/* key schedule:
first do ip;
	if bidirectional
		if dst < src
			swap, forward = false, checked = true
		else if dst > src
			checked = true
then do tcp/udp/icmp
	if bidirectional
		if not checked and dst < src:
			swap if tcp/udp, forward = false, checked = true
		else if dst > src
			checked = true
then do ethernet
	if bidirectional
		if not checked and dst < src:
			swap, forward = false, checked = true
*/

// Key computes a key according to the given selector. Returns key, isForward, ok
// This function _must not_ be called concurrently.
func (selector *DynamicKeySelector) Key(packet Buffer) (string, bool, bool) {
	if selector.empty {
		return emptyKey, true, true
	}

	i := 0
	forward := true
	swapchecked := false

	if selector.network {
		network := packet.NetworkLayer()
		if network == nil {
			if !selector.allowZero {
				return emptyKey, true, false
			}
		} else {
			flow := network.NetworkFlow()
			if selector.bidirectional && selector.srcIP && selector.dstIP {
				a := flow.Src().Raw()
				b := flow.Dst().Raw()
				res := bytes.Compare(a, b)
				if res < 0 {
					swapchecked = true
					i += copy(scratchKey[i:], a)
					i += copy(scratchKey[i:], b)
				} else if res == 0 {
					i += copy(scratchKey[i:], a)
					i += copy(scratchKey[i:], b)
				} else {
					swapchecked = true
					forward = false
					i += copy(scratchKey[i:], b)
					i += copy(scratchKey[i:], a)
				}
			} else {
				if selector.srcIP {
					i += copy(scratchKey[i:], flow.Src().Raw())
				}
				if selector.dstIP {
					i += copy(scratchKey[i:], flow.Dst().Raw())
				}
			}
			if selector.protocolIdentifier {
				scratchKey[i] = byte(packet.Proto())
				i++
			}
		}
	}

	if selector.transport {
		transport := packet.TransportLayer()
		if transport == nil {
			if !selector.allowZero {
				return emptyKey, true, false
			}
		} else {
			flow := transport.TransportFlow()
			if selector.bidirectional && selector.srcPort && selector.dstPort && flow.EndpointType() != icmpEndpointType {
				a := flow.Src().Raw()
				b := flow.Dst().Raw()
				if swapchecked {
					if forward {
						i += copy(scratchKey[i:], a)
						i += copy(scratchKey[i:], b)
					} else {
						i += copy(scratchKey[i:], b)
						i += copy(scratchKey[i:], a)
					}
				} else {
					res := bytes.Compare(a, b)
					if res < 0 {
						swapchecked = true
						i += copy(scratchKey[i:], a)
						i += copy(scratchKey[i:], b)
					} else if res == 0 {
						i += copy(scratchKey[i:], a)
						i += copy(scratchKey[i:], b)
					} else {
						swapchecked = true
						forward = false
						i += copy(scratchKey[i:], b)
						i += copy(scratchKey[i:], a)
					}
				}
			} else {
				if selector.srcPort && flow.EndpointType() != icmpEndpointType {
					i += copy(scratchKey[i:], flow.Src().Raw())
				}
				if selector.dstPort {
					i += copy(scratchKey[i:], flow.Dst().Raw())
				}
			}
		}
	}

	if selector.link {
		link := packet.LinkLayer()
		if link == nil {
			if !selector.allowZero {
				return emptyKey, true, false
			}
		} else {
			flow := link.LinkFlow()
			if selector.bidirectional && selector.srcMac && selector.dstMac {
				a := flow.Src().Raw()
				b := flow.Dst().Raw()
				if swapchecked {
					if forward {
						i += copy(scratchKey[i:], a)
						i += copy(scratchKey[i:], b)
					} else {
						i += copy(scratchKey[i:], b)
						i += copy(scratchKey[i:], a)
					}
				} else {
					res := bytes.Compare(a, b)
					if res < 0 {
						swapchecked = true
						i += copy(scratchKey[i:], a)
						i += copy(scratchKey[i:], b)
					} else if res == 0 {
						i += copy(scratchKey[i:], a)
						i += copy(scratchKey[i:], b)
					} else {
						swapchecked = true
						forward = false
						i += copy(scratchKey[i:], b)
						i += copy(scratchKey[i:], a)
					}
				}
			} else {
				if selector.srcMac {
					i += copy(scratchKey[i:], flow.Src().Raw())
				}
				if selector.dstMac {
					i += copy(scratchKey[i:], flow.Dst().Raw())
				}
			}
			if selector.etherType {
				t := packet.EtherType()
				scratchKey[i] = byte(t >> 8)
				i++
				scratchKey[i] = byte(t & 0x00FF)
				i++
			}
		}
	}

	return string(scratchKey[:i]), forward, true
}
