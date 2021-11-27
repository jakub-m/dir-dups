package analyze

import (
	"greasytoad/log"
	"strings"
)

func FindSimilarities(root *Node, onNodes func(SimilarityType, []*Node)) {
	indexByHash := indexNodesByHashOptimized(root)
	// alreadyReported holds nodes that appeared on in the output. This is to skip analysing nodes that already appeared as duplicate
	// of another node. This results in less noise on the output.
	alreadyReported := make(map[*Node]bool)

	//alreadyReportedAsDuplicate prevents descending into reported duplicates
	alreadyReportedAsDuplicate := make(map[*Node]bool)

	Walk(root, func(n *Node) bool {
		if _, ok := alreadyReportedAsDuplicate[n]; ok {
			log.Debugf("FindSimilarities: skip '%s', already reported as duplicate. Do not descend.", n.FullPath())
			return false
		}

		if _, ok := alreadyReported[n]; ok {
			log.Debugf("FindSimilarities: skip '%s', already reported", n.FullPath())
			return true
		}

		similar, ok := indexByHash[n.Hash]
		if !ok {
			// This can happen for optimized index with removed parents with similar hashes as children.
			log.Debugf("FindSimilarities: skip '%s', not indexes (has duplicate child)", n.FullPath())
			return true
		}

		//similar = filterDuplicate(n, similar)
		if len(similar) == 1 {
			updateNodeSet(alreadyReported, similar)
			onNodes(Unique, similar)
			return true
		} else {
			updateNodeSet(alreadyReported, similar)
			updateNodeSet(alreadyReportedAsDuplicate, similar)
			onNodes(FullDuplicate, similar)
			// do not descend on full duplicate, analysing children will not add any new information.
			return false
		}
	})
}

func indexNodesByHashOptimized(root *Node) map[hash][]*Node {
	m := make(map[hash][]*Node)
	walkAll(root, func(n *Node) {
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

func updateNodeSet(m map[*Node]bool, nodes []*Node) {
	for _, n := range nodes {
		m[n] = true
	}
}

func formatNodes(nodes []*Node) string {
	ss := []string{}
	for _, n := range nodes {
		ss = append(ss, n.FullPath())
	}
	return strings.Join(ss, ", ")
}

func walkAll(root *Node, onNode func(*Node)) {
	Walk(root, func(n *Node) bool {
		onNode(n)
		return true
	})
}
