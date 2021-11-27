package analyze

import (
	"greasytoad/log"
	"strings"
)

func FindSimilarities(root *Node, onNodes func(SimilarityType, []*Node)) {
	indexByHash := indexNodesByHashOptimized(root)

	Walk(root, func(n *Node) bool {
		similar, ok := indexByHash[n.Hash]
		if !ok {
			// This can happen for optimized index with removed parents with similar hashes as children.
			log.Debugf("FindSimilarities: skip '%s', not indexes (has duplicate child)", n.FullPath())
			return true
		}

		//similar = filterDuplicate(n, similar)
		if len(similar) == 1 {
			onNodes(Unique, similar)
		} else {
			onNodes(FullDuplicate, similar)
		}

		return true
	})
}

func indexNodesByHashOptimized(root *Node) map[hash][]*Node {
	m := make(map[hash][]*Node)
	Walk(root, func(n *Node) bool {
		nodes, ok := m[n.Hash]
		if !ok {
			nodes = []*Node{}
			m[n.Hash] = nodes
		}
		hasSameHash := func(d *Node) bool {
			return n.Hash == d.Hash
		}
		if ch := n.FindChild(hasSameHash); ch != nil {
			// Do not index node if the there is a node with the same hash. This results in omitting parent nodes that have
			// duplicated children, which results in less noise on output.
			log.Debugf("indexNodesByHashOptimized: ignore '%s' because of a duplicated child '%s'", n.FullPath(), ch.Name)
		} else {
			m[n.Hash] = append(nodes, n)
		}
		return true
	})
	return m
}

func (n *Node) FindChild(cond func(*Node) bool) *Node {
	if n.Children == nil {
		return nil
	}
	for _, ch := range n.Children {
		if cond(ch) {
			return ch
		}
	}
	return nil
}

func formatNodes(nodes []*Node) string {
	ss := []string{}
	for _, n := range nodes {
		ss = append(ss, n.FullPath())
	}
	return strings.Join(ss, ", ")
}
