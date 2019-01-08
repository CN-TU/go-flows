package packet

import (
	"bytes"
	"fmt"
	"strings"
)

type keyBuilder struct {
	spec keySpecification
	name string
}

func (k keyBuilder) make() KeyFunc {
	return k.spec.make(k.name)
}

// MakeDynamicKeySelector creates a selector function from a dynamic key definition
func MakeDynamicKeySelector(key []string, bidirectional, allowZero bool) (ret DynamicKeySelector) {
	ret.noZero = !allowZero
	if len(key) == 0 {
		ret.empty = true
		return
	}

	stringMatcher := make(map[string]*stringKey)
	regexpMatcher := make([]*regexpKey, 0, len(keyRegistry))
	for _, k := range keyRegistry {
		switch matcher := k.(type) {
		case *stringKey:
			for _, m := range matcher.match {
				stringMatcher[m] = matcher
			}
		case *regexpKey:
			regexpMatcher = append(regexpMatcher, matcher)
		}
	}

	isFivetuple := make(map[int]bool, len(fivetupleMust))
	for _, id := range fivetupleMust {
		isFivetuple[id] = true
	}

	keys := make([]keyBuilder, len(key))
	used := make(map[int]bool, len(key))
	pairs := make(map[int][]int, keyPairID)
MAIN:
	for i := range key {
		spec, ok := stringMatcher[key[i]]
		if ok {
			if used[spec.id] {
				panic(fmt.Sprintf("Key '%s' used twice", key[i]))
			}
			used[spec.id] = true
			keys[i].spec = spec
			keys[i].name = key[i]
			if spec.t == KeyTypeSource || spec.t == KeyTypeDestination {
				pairs[spec.pair] = append(pairs[spec.pair], i)
			}
			delete(isFivetuple, spec.id)
			continue
		}
		for _, spec := range regexpMatcher {
			if spec.match.MatchString(key[i]) {
				if used[spec.id] {
					panic(fmt.Sprintf("Key '%s' used twice", key[i]))
				}
				used[spec.id] = true
				keys[i].spec = spec
				keys[i].name = key[i]
				if spec.t == KeyTypeSource || spec.t == KeyTypeDestination {
					pairs[spec.pair] = append(pairs[spec.pair], i)
				}
				delete(isFivetuple, spec.id)
				continue MAIN
			}
		}
		panic(fmt.Sprintf("Unknown key_feature '%s'", key[i]))
	}

	done := make(map[int]bool, len(key))

	if bidirectional {
		if len(isFivetuple) == 0 {
			ret.fivetuple = true
		}
		for _, pair := range pairs {
			if len(pair) != 2 {
				continue
			}
			if keys[pair[0]].spec.getType() == KeyTypeSource {
				ret.source = append(ret.source, keys[pair[0]].make())
				ret.destination = append(ret.destination, keys[pair[1]].make())
			} else {
				ret.source = append(ret.source, keys[pair[1]].make())
				ret.destination = append(ret.destination, keys[pair[0]].make())
			}
			done[pair[0]] = true
			done[pair[1]] = true
		}
	}

	for i := range keys {
		if done[i] {
			continue
		}
		ret.uni = append(ret.uni, keys[i].make())
	}

	ret.bidirectional = bidirectional

	return
}

// DynamicKeySelector holds the definition for a flow key function
type DynamicKeySelector struct {
	source        []KeyFunc
	destination   []KeyFunc
	uni           []KeyFunc
	noZero        bool
	bidirectional bool
	fivetuple     bool
	empty         bool
}

func sourceIPAddressKey(packet Buffer, scratch, scratchNoSort []byte) (int, int) {
	network := packet.NetworkLayer()
	if network == nil {
		return 0, 0
	}
	return copy(scratch, network.NetworkFlow().Src().Raw()), 0
}

func destinationIPAddressKey(packet Buffer, scratch, scratchNoSort []byte) (int, int) {
	network := packet.NetworkLayer()
	if network == nil {
		return 0, 0
	}
	return copy(scratch, network.NetworkFlow().Dst().Raw()), 0
}

func protocolIdentifierKey(packet Buffer, scratch, scratchNoSort []byte) (int, int) {
	scratch[0] = byte(packet.Proto())
	return 1, 0
}

func sourceTransportPortKey(packet Buffer, scratch, scratchNoSort []byte) (int, int) {
	transport := packet.TransportLayer()
	if transport == nil {
		return 0, 0
	}
	flow := transport.TransportFlow()
	if flow.EndpointType() != icmpEndpointType {
		return copy(scratch, flow.Src().Raw()), 0
	}
	return 0, copy(scratchNoSort, flow.Src().Raw())
}

func destinationTransportPortKey(packet Buffer, scratch, scratchNoSort []byte) (int, int) {
	transport := packet.TransportLayer()
	if transport == nil {
		return 0, 0
	}
	flow := transport.TransportFlow()
	if flow.EndpointType() != icmpEndpointType {
		return copy(scratch, flow.Dst().Raw()), 0
	}
	return 0, copy(scratchNoSort, flow.Dst().Raw())
}

var fivetupleMust []int

func init() {
	srcIP := RegisterStringsKey([]string{"sourceIPv4Address", "sourceIPv6Address", "sourceIPAddress"},
		"source address of network layer",
		KeyTypeSource, KeyLayerNetwork, func(string) KeyFunc { return sourceIPAddressKey })
	dstIP := RegisterStringsKey([]string{"destinationIPv4Address", "destinationIPv6Address", "destinationIPAddress"},
		"destination address of network layer",
		KeyTypeDestination, KeyLayerNetwork, func(string) KeyFunc { return destinationIPAddressKey })
	RegisterKeyPair(srcIP, dstIP)
	proto := RegisterStringKey("protocolIdentifier",
		"protocol identifier field of network layer",
		KeyTypeUnidirectional, KeyLayerNetwork, func(string) KeyFunc { return protocolIdentifierKey })
	srcPort := RegisterStringKey("sourceTransportPort",
		"source port of transport layer",
		KeyTypeSource, KeyLayerTransport, func(string) KeyFunc { return sourceTransportPortKey })
	dstPort := RegisterStringKey("destinationTransportPort",
		"destination port of transport layer",
		KeyTypeDestination, KeyLayerTransport, func(string) KeyFunc { return destinationTransportPortKey })
	RegisterKeyPair(srcPort, dstPort)

	fivetupleMust = []int{srcIP, dstIP, proto, srcPort, dstPort}
}

var scratchSourceKey [1024]byte
var scratchDestinationKey [1024]byte
var scratchUniKey [2048]byte
var emptyKey string

// Key computes a key according to the given selector. Returns key, isForward, ok
// This function _must not_ be called concurrently.
func (selector *DynamicKeySelector) Key(packet Buffer) (string, bool, bool) {
	if selector.empty {
		return emptyKey, true, true
	}

	if !selector.bidirectional {
		i := 0
		for _, f := range selector.uni {
			a, b := f(packet, scratchUniKey[i:], scratchUniKey[i:])
			if a+b == 0 && selector.noZero {
				return emptyKey, true, false
			}
			i += a + b
		}
		return string(scratchUniKey[:i]), true, true
	}

	forward := true
	uni := 0

	source := 0
	for _, f := range selector.source {
		a, b := f(packet, scratchSourceKey[source:], scratchUniKey[uni:])
		if a+b == 0 && selector.noZero {
			return emptyKey, true, false
		}
		source += a
		uni += b
	}

	destination := 0
	for _, f := range selector.destination {
		a, b := f(packet, scratchDestinationKey[destination:], scratchUniKey[uni:])
		if a+b == 0 && selector.noZero {
			return emptyKey, true, false
		}
		destination += a
		uni += b
	}

	for _, f := range selector.uni {
		a, b := f(packet, scratchUniKey[uni:], scratchUniKey[uni:])
		if a+b == 0 && selector.noZero {
			return emptyKey, true, false
		}
		uni += a + b
	}

	builder := strings.Builder{}
	builder.Grow(source + destination + uni)

	s := scratchSourceKey[:source]
	d := scratchDestinationKey[:destination]
	if bytes.Compare(s, d) > 0 {
		forward = false
		builder.Write(d)
		builder.Write(s)
	} else {
		builder.Write(s)
		builder.Write(d)
	}
	builder.Write(scratchUniKey[:uni])

	return builder.String(), forward, true
}
