package flows

import (
	"bytes"
)

type walkFn func(f Flow)

type edge struct {
	node  *node
	label byte
}

type node struct {
	leaf   Flow
	prefix []byte
	edges  map[byte]edge
}

func (n *node) isLeaf() bool {
	return n.leaf != nil
}

func (n *node) addEdge(e edge) {
	n.edges[e.label] = e
}

func (n *node) replaceEdge(e edge) {
	n.edges[e.label] = e
}

func (n *node) getEdge(label byte) *node {
	ret, found := n.edges[label]
	if !found {
		return nil
	}
	return ret.node
}

func (n *node) delEdge(label byte) {
	delete(n.edges, label)
}

type tree struct {
	root *node
}

func newTree() *tree {
	return &tree{root: &node{edges: make(map[byte]edge)}}
}

func longestPrefix(k1, k2 []byte) int {
	max := len(k1)
	if l := len(k2); l < max {
		max = l
	}
	var i int
	for i = 0; i < max; i++ {
		if k1[i] != k2[i] {
			break
		}
	}
	return i
}

func (t *tree) insert(search []byte, f Flow) {
	var parent *node
	n := t.root
	for {
		if len(search) == 0 {
			if n.isLeaf() {
				panic("Already existent")
			}

			n.leaf = f
			return
		}

		parent = n
		n = n.getEdge(search[0])

		if n == nil {
			e := edge{
				label: search[0],
				node: &node{
					leaf:   f,
					prefix: search,
					edges:  make(map[byte]edge),
				},
			}
			parent.addEdge(e)
			return
		}

		commonPrefix := longestPrefix(search, n.prefix)
		if commonPrefix == len(n.prefix) { //exact edge match
			search = search[commonPrefix:]
			continue
		}

		child := &node{
			prefix: search[:commonPrefix],
			edges:  make(map[byte]edge),
		}
		parent.replaceEdge(edge{
			label: search[0],
			node:  child,
		})

		child.addEdge(edge{
			label: n.prefix[commonPrefix],
			node:  n,
		})
		n.prefix = n.prefix[commonPrefix:]

		search = search[commonPrefix:]
		if len(search) == 0 {
			child.leaf = f
			return
		}

		child.addEdge(edge{
			label: search[0],
			node: &node{
				leaf:   f,
				prefix: search,
				edges:  make(map[byte]edge),
			},
		})
		//maybe optimize the double sort?
		return
	}
}

func (t *tree) delete(search []byte) {
	var parent *node
	var label byte
	n := t.root
	for {
		if len(search) == 0 {
			if !n.isLeaf() {
				panic("delete non existent flow A")
			}
			goto DELETE
		}

		parent = n
		label = search[0]
		n = n.getEdge(label)
		if n == nil {
			panic("delete non existent flow B")
		}

		// Consume the search prefix
		if bytes.HasPrefix(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			panic("delete non existent flow C")
		}
	}

DELETE:
	n.leaf = nil

	// Check if we should delete this node from the parent
	if parent != nil && len(n.edges) == 0 {
		parent.delEdge(label)
	}

	// Check if we should merge this node
	if n != t.root && len(n.edges) == 1 {
		n.mergeChild()
	}

	// Check if we should merge the parent's other child
	if parent != nil && parent != t.root && len(parent.edges) == 1 && !parent.isLeaf() {
		parent.mergeChild()
	}

	return
}

func (n *node) mergeChild() {
	var e edge
	for _, e = range n.edges {
		break
	}
	child := e.node
	n.prefix = append(n.prefix, child.prefix...)
	n.leaf = child.leaf
	n.edges = child.edges
}

// Get is used to lookup a specific key, returning
// the value and if it was found
func (t *tree) get(s []byte) (Flow, bool) {
	n := t.root
	search := s
	for {
		// Check for key exhaution
		if len(search) == 0 {
			if n.isLeaf() {
				return n.leaf, true
			}
			break
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if bytes.HasPrefix(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
	return nil, false
}

func (t *tree) walk(fn walkFn) {
	recursiveWalk(t.root, fn)
}

func recursiveWalk(n *node, fn walkFn) {
	// Visit the leaf values if any
	if n == nil {
		return
	}
	if n.leaf != nil {
		fn(n.leaf)
		return
	}

	// Recurse on the children
	for _, e := range n.edges {
		recursiveWalk(e.node, fn)
	}
}
