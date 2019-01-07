package packet

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// KeyFunc is a function that writes the flow key for a single property
// The key must be written to scratch and the written length returned as scratchLen.
//
// If the property is part of a keypair, it must write to scratch (both parts of the pair must use the same length!)
// if the result must be ordered for bidirectional flows,
// and to scratchNoSort (with coresponding length in scratchNoSortLen) if this part must not be ordered.
// All other properties must use scratch.
type KeyFunc func(packet Buffer, scratch, scratchNoSort []byte) (scratchLen int, scratchNoSortLen int)

// MakeKeyFunc must return a KeyFunc. Additional
type MakeKeyFunc func(name string) KeyFunc

// KeyType specifies the type of this key (unidirectional, source, or destination )
type KeyType int

const (
	// KeyTypeUnidirectional is a key that mustn't be sorted for bidirectional flows
	KeyTypeUnidirectional KeyType = iota
	// KeyTypeSource is a source key that must be sorted for bidirectional flows
	KeyTypeSource
	// KeyTypeDestination is a destination key that must be sorted for bidirectional flows
	KeyTypeDestination
)

// KeyLayer specifies on which layer this key resides
type KeyLayer int

// do not change the order here! This is the order for bidirectional flow comparison
const (
	// KeyLayerNetwork specifies a key on the network layer
	KeyLayerNetwork KeyLayer = iota
	// KeyLayerTransport specifies a key on the transport layer
	KeyLayerTransport
	// KeyLayerApplication specifies a key on the application layer
	KeyLayerApplication
	// KeyLayerLink specifies a key on the link layer
	KeyLayerLink
	// KeyLayerNone specifies a key with no layer
	KeyLayerNone
)

type regexpKey struct {
	match   *regexp.Regexp
	keyfunc MakeKeyFunc
	t       KeyType
	layer   KeyLayer
	id      int
	pair    int
}

func (k *regexpKey) make(name string) KeyFunc {
	return k.keyfunc(name)
}

func (k *regexpKey) getID() int {
	return k.id
}

func (k *regexpKey) getPair() int {
	return k.pair
}

func (k *regexpKey) setPair(i int) {
	k.pair = i
}

func (k *regexpKey) getLayer() KeyLayer {
	return k.layer
}

func (k *regexpKey) getType() KeyType {
	return k.t
}

type stringKey struct {
	match   []string
	keyfunc MakeKeyFunc
	t       KeyType
	layer   KeyLayer
	id      int
	pair    int
}

func (k *stringKey) make(name string) KeyFunc {
	return k.keyfunc(name)
}

func (k *stringKey) getID() int {
	return k.id
}

func (k *stringKey) getPair() int {
	return k.pair
}

func (k *stringKey) setPair(i int) {
	k.pair = i
}

func (k *stringKey) getLayer() KeyLayer {
	return k.layer
}

func (k *stringKey) getType() KeyType {
	return k.t
}

type keySpecification interface {
	make(string) KeyFunc
	getID() int
	getPair() int
	setPair(int)
	getLayer() KeyLayer
	getType() KeyType
}

var keyRegistry []keySpecification
var keyPairID = 1

// RegisterKeyPair registers the given key ids as a source/destination pair
func RegisterKeyPair(a, b int) {
	if a < 0 || a > len(keyRegistry) {
		panic(fmt.Sprintf("Key with id %d not registered", a))
	}
	if b < 0 || b > len(keyRegistry) {
		panic(fmt.Sprintf("Key with id %d not registered", b))
	}
	first := keyRegistry[a]
	second := keyRegistry[b]
	if first.getLayer() != second.getLayer() {
		panic(fmt.Sprintf("Key layers of %d and %d don't match!", a, b))
	}
	if !((first.getType() == KeyTypeSource && second.getType() == KeyTypeDestination) ||
		(second.getType() == KeyTypeSource && first.getType() == KeyTypeDestination)) {
		panic(fmt.Sprintf("One of %d and %d must be source and one destination", a, b))
	}
	first.setPair(keyPairID)
	second.setPair(keyPairID)
	keyPairID++
}

// RegisterRegexpKey registers a regex key function
func RegisterRegexpKey(name string, t KeyType, layer KeyLayer, make MakeKeyFunc) int {
	id := len(keyRegistry)
	keyRegistry = append(keyRegistry, &regexpKey{
		match:   regexp.MustCompile(name),
		keyfunc: make,
		t:       t,
		layer:   layer,
		id:      id,
	})
	return id
}

// RegisterStringKey registers a regex key function
func RegisterStringKey(name string, t KeyType, layer KeyLayer, make MakeKeyFunc) int {
	return RegisterStringsKey([]string{name}, t, layer, make)
}

// RegisterStringsKey registers a regex key function
func RegisterStringsKey(name []string, t KeyType, layer KeyLayer, make MakeKeyFunc) int {
	id := len(keyRegistry)
	keyRegistry = append(keyRegistry, &stringKey{
		match:   name,
		keyfunc: make,
		t:       t,
		layer:   layer,
		id:      id,
	})
	return id
}

// ListKeys writes a list of keys to w
func ListKeys(w io.Writer) {
	for _, key := range keyRegistry {
		switch k := key.(type) {
		case *regexpKey:
			fmt.Fprintln(w, k.match.String())
		case *stringKey:
			fmt.Fprintln(w, strings.Join(k.match, "|"))
		}
	}
}
