package analyze

import (
	"fmt"
	"greasytoad/log"
	"sort"
	"strings"
)

type similarityMap map[*Node]similarityMapValue

type similarityMapValue struct {
	similarityType SimilarityType
	sameHash       []*Node
}

func (m similarityMap) set(node *Node, st SimilarityType, sameHash []*Node) {
	m[node] = similarityMapValue{st, sameHash}
}

func (m similarityMap) getType(node *Node) SimilarityType {
	return m.get(node).similarityType
}

func (m similarityMap) get(node *Node) similarityMapValue {
	if v, ok := m[node]; ok {
		return v
	} else {
		return similarityMapValue{Unknown, nil}
	}
}

func FindSimilarities(root *Node, onNodes func(SimilarityType, []*Node)) {
	similarityMap := getSimilarityMap(root)

	// alreadyReported holds nodes that appeared on in the output. This is to skip analysing nodes that already appeared as duplicate
	// of another node. This results in less noise on the output.
	alreadyReported := make(map[*Node]bool)

	Walk(root, func(currentNode *Node) bool {
		log.Debugf("FindSimilarities: now walk %s", currentNode.FullPath())

		similarity := similarityMap.get(currentNode)

		if _, wasAlreadyReported := alreadyReported[currentNode]; !wasAlreadyReported {
			// do not call callback if was already reported
			log.Debugf("FindSimilarities: callback '%s', similarity %s", currentNode.FullPath(), similarity.similarityType)
			onNodes(similarity.similarityType, similarity.sameHash)
		}

		updateNodeSet(alreadyReported, currentNode)
		updateNodeSet(alreadyReported, similarity.sameHash...)

		if similarity.similarityType == FullDuplicate {
			if someChildren(currentNode, condSameHash(currentNode.Hash)) {
				return true
			} else {
				// if current node is a full duplicate of other node, and there is no child node with similar hash, then do not
				// descend. it won't bring any useful information.
				log.Debugf("FindSimilarities: FullDuplicate, do not descend %s", currentNode.FullPath())
				return false
			}
		}

		return true
	})
}

func indexNodesByHashOptimized(root *Node) map[hash][]*Node {
	// if there is a directory structure a/b/f then a, b and f will have the same hash. Here we report only one of those three,
	// otherwise they would show up as duplicates of each other.
	m := make(map[hash][]*Node)
	walkAll(root, func(n *Node) {
		hasSameHash := func(d *Node) bool {
			return n.Hash == d.Hash
		}
		if ch := n.FindChild(hasSameHash); ch != nil {
			// Do not index node if the there is a node with the same hash. This results in omitting parent nodes that have
			// duplicated children, which results in less noise on output.
			log.Debugf("indexNodesByHashOptimized: ignore '%s' because of a duplicated child '%s'", n.FullPath(), ch.Name)
			return
		}
		nodes, ok := m[n.Hash]
		if !ok {
			nodes = []*Node{}
			m[n.Hash] = nodes
		}
		m[n.Hash] = append(nodes, n)
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

func updateNodeSet(m map[*Node]bool, nodes ...*Node) {
	panicAssertf(nodes != nil, "nodes are null")
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

func getSimilarityMap(root *Node) similarityMap {
	similarityMap := make(similarityMap)

	nodesByHash := indexNodesByHashOptimized(root)

	var updateSimilarityRec func(*Node)
	updateSimilarityRec = func(node *Node) {
		for _, ch := range node.Children {
			// guarantee that the children have the status already set.
			updateSimilarityRec(ch)
		}
		similarNodes := nodesByHash[node.Hash]
		sort.Slice(similarNodes, func(i, j int) bool {
			return similarNodes[i].FullPath() < similarNodes[j].FullPath()
		})
		if len(nodesByHash[node.Hash]) > 1 {
			// there are nodes with similar hashes, so it is a duplicate.
			similarityMap.set(node, FullDuplicate, similarNodes)
			return
		}
		// the code below assumes that there are no other nodes with similar hashes

		fullOrWeakDuplicate := func(n *Node) bool {
			return similarityMap.getType(n) == FullDuplicate || similarityMap.getType(n) == WeakDuplicate
		}
		unique := func(n *Node) bool {
			return similarityMap.getType(n) == Unique
		}
		uniqueOrPartiallyUnique := func(n *Node) bool {
			return similarityMap.getType(n) == Unique || similarityMap.getType(n) == PartiallyUnique
		}
		unknown := func(n *Node) bool {
			return similarityMap.getType(n) == Unknown
		}

		if node.IsFile() {
			// a file without similar nodes is a unique.
			similarityMap.set(node, Unique, similarNodes)
			return
		}
		// all child nodes are full duplicates, but not necessarily in a similar file tree.
		// this node is marked as weak duplicate.
		if allChildren(node, fullOrWeakDuplicate) {
			similarityMap.set(node, WeakDuplicate, similarNodes)
			return
		}
		if allChildren(node, unique) {
			similarityMap.set(node, Unique, similarNodes)
			return
		}
		if allChildren(node, uniqueOrPartiallyUnique) {
			similarityMap.set(node, PartiallyUnique, similarNodes)
			return
		}
		if someChildren(node, fullOrWeakDuplicate) &&
			someChildren(node, uniqueOrPartiallyUnique) &&
			noChildren(node, unknown) {
			similarityMap.set(node, PartiallyUnique, similarNodes)
			return
		}
		log.Debugf("ERROR: unknown similarity type for: %s", node.FullPath())
	}

	updateSimilarityRec(root)
	return similarityMap
}

func condSameHash(referenceHash hash) func(*Node) bool {
	return func(n *Node) bool {
		return n.Hash == referenceHash
	}
}

func panicAssertf(cond bool, format string, args ...interface{}) {
	if !cond {
		m := fmt.Sprintf(format, args...)
		panic(m)
	}
}
