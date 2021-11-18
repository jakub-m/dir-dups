package analyze

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
)

type hash uint64

func (h hash) Equal(other hash) bool {
	return h == other
}

type Node struct {
	Name           string
	Size           int
	FileCount      int
	Hash           hash
	Children       map[string]*Node // map children node name to node
	SimilarityType SimilarityType
	// Similar nodes have the same hash.
	Similar []*Node `json:"-"`
	Parent  *Node   `json:"-"`
}

type SimilarityType int

const (
	// Unknown is a zero value.
	Unknown SimilarityType = iota
	// FullDuplicate hashes are equal. it means that files are the same..
	FullDuplicate
	// WeakDuplicate applicable only for directory, it means that all the content of a directory
	// is a full duplicate, but the structure of the files is not the same.
	WeakDuplicate
	// PartiallyUnique for directory, some of the content is not duplicated.
	PartiallyUnique
	// Unique not duplicated.
	Unique
)

func (s SimilarityType) String() string {
	switch s {
	case Unknown:
		return "x"
	case FullDuplicate:
		return "D"
	case WeakDuplicate:
		return "d"
	case PartiallyUnique:
		return "u"
	case Unique:
		return "U"
	default:
		return "?"
	}
}

func (n *Node) FullPath() string {
	parts := []string{}

	d := n
	for d != nil {
		parts = append(parts, d.Name)
		d = d.Parent
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	if !n.IsFile() {
		parts = append(parts, "") // append / to dir
	}
	s := strings.Join(parts, "/")
	return s
}

func (n *Node) IsFile() bool {
	return len(n.Children) == 0
}

func LoadNodesFromFileList(data io.Reader) (*Node, error) {
	scanner := bufio.NewScanner(data)

	root := newNode("")

	for scanner.Scan() {
		parsed, err := parseLine(scanner.Text())
		if err != nil {
			return nil, err
		}

		n := root
		for i, p := range parsed.path {
			if i == len(parsed.path)-1 {
				// last, that is the file
				newChild := newNode(p)
				newChild.Size = parsed.size
				newChild.FileCount = 1
				newChild.Parent = n
				newChild.Hash = calculateHashFromString(parsed.hash)
				n.Children[p] = newChild
			} else {
				if p == "" {
					continue
				}
				if ch, ok := n.Children[p]; ok {
					n = ch
				} else {
					newChild := newNode(p)
					newChild.Parent = n
					n.Children[p] = newChild
					n = newChild
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// recalculate sizes
	var updateSizeRec func(node *Node)
	updateSizeRec = func(node *Node) {
		if node.IsFile() {
			return
		}
		size, fileCount := 0, 0
		for _, ch := range node.Children {
			updateSizeRec(ch)
			size += ch.Size
			fileCount += ch.FileCount
		}
		node.Size = size
		node.FileCount = fileCount
	}
	updateSizeRec(root)

	// recalculate hashes
	var updateHashRec func(node *Node)
	updateHashRec = func(node *Node) {
		for _, ch := range node.Children {
			updateHashRec(ch)
		}
		node.Hash = calculateHash(node)
	}
	updateHashRec(root)

	return root, nil
}

func calculateHashFromString(s string) hash {
	h := fnv.New64a()
	io.WriteString(h, s)
	return hash(h.Sum64())
}

func calculateHash(node *Node) hash {
	h := fnv.New64a()
	if node.IsFile() {
		// hash for files is already calculated during ingest of the input data.
		return node.Hash
	} else {
		// a directory derives the hash from its children. Does not take into account
		// directory name, so we can find changed dirs with the same content.
		children := []*Node{}
		for _, ch := range node.Children {
			children = append(children, ch)
		}
		sort.Slice(children, func(i, j int) bool {
			return children[i].Name < children[j].Name
		})
		if len(children) == 1 {
			// bubble up hash of a single child.
			return children[0].Hash
		} else {
			for _, ch := range children {
				b := []byte(strconv.Itoa(int(ch.Hash)))
				h.Write(b)
			}
		}
	}

	return hash(h.Sum64())
}

func newNode(name string) *Node {
	return &Node{
		Name:     name,
		Children: make(map[string]*Node),
		Similar:  []*Node{},
	}
}

type parsed struct {
	path []string
	size int
	hash string
}

func parseLine(line string) (parsed, error) {
	line = strings.Trim(line, "\n")
	parts := strings.Split(line, "\t")
	parsed := parsed{}
	if len(parts) != 3 {
		return parsed, fmt.Errorf("bad line: %d parts, `%v`", len(parts), line)
	}
	parsed.path = strings.Split(parts[0], "/")
	size, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return parsed, err
	}
	parsed.size = int(size)
	parsed.hash = parts[2]
	return parsed, nil
}

func AnalyzeDuplicates(left, right *Node) {
	indexLeft := indexNodesByHash(left)
	indexRight := indexNodesByHash(right)

	log.Printf("hashes: left %d, right %d", len(indexLeft), len(indexRight))
	leftOnly, overlap, rightOnly := findHashOverlap(indexLeft, indexRight)
	log.Printf("hashes: left only %d, overlap %d, right only %d", len(leftOnly), len(overlap), len(rightOnly))

	// cross-reference similar nodes.
	for h := range overlap {
		for _, leftNode := range indexLeft[h] {
			leftNode.Similar = indexRight[h]
		}
		for _, rightNode := range indexRight[h] {
			rightNode.Similar = indexLeft[h]
		}
	}

	updateSimilarity(left)
	updateSimilarity(right)
}

func allChildren(node *Node, cond func(*Node) bool) bool {
	for _, ch := range node.Children {
		if !cond(ch) {
			return false
		}
	}
	return true
}

func someChildren(node *Node, cond func(*Node) bool) bool {
	for _, ch := range node.Children {
		if cond(ch) {
			return true
		}
	}
	return false
}

func noChildren(node *Node, cond func(*Node) bool) bool {
	return !someChildren(node, cond)
}

func indexNodesByHash(root *Node) map[hash][]*Node {
	m := make(map[hash][]*Node)
	Walk(root, func(n *Node) bool {
		nodes, ok := m[n.Hash]
		if !ok {
			m[n.Hash] = []*Node{n}
			return true
		}
		m[n.Hash] = append(nodes, n)
		// The list will now contain parents and children with the same hash, which is not great.
		// This could be optimized out later.
		return true
	})
	return m
}

func findHashOverlap(left, right map[hash][]*Node) (leftOnly, overlap, rightOnly map[hash]bool) {
	leftOnly = make(map[hash]bool)
	overlap = make(map[hash]bool)
	rightOnly = make(map[hash]bool)

	for h := range left {
		if _, ok := right[h]; ok {
			overlap[h] = true
		} else {
			leftOnly[h] = true
		}
	}
	for h := range right {
		if _, ok := left[h]; ok {
			overlap[h] = true
		} else {
			rightOnly[h] = true
		}
	}

	return
}

func updateSimilarity(node *Node) {
	var updateSimilarityRec func(*Node)

	updateSimilarityRec = func(node *Node) {
		for _, ch := range node.Children {
			// guarantee that the children have the status already set.
			updateSimilarityRec(ch)
		}
		if len(node.Similar) > 0 {
			// there are nodes with similar hashes, so it is a duplicate.
			node.SimilarityType = FullDuplicate
			return
		}
		if node.IsFile() {
			// a file without similar nodes is a unique.
			node.SimilarityType = Unique
			return
		}

		fullOrWeakDuplicate := func(n *Node) bool {
			return n.SimilarityType == FullDuplicate || n.SimilarityType == WeakDuplicate
		}
		unique := func(n *Node) bool {
			return n.SimilarityType == Unique
		}
		uniqueOrPartiallyUnique := func(n *Node) bool {
			return n.SimilarityType == Unique || n.SimilarityType == PartiallyUnique
		}
		unknown := func(n *Node) bool {
			return n.SimilarityType == Unknown
		}

		// all child nodes are full duplicates, but not necessarily in a similar file tree.
		// this node is marked as weak duplicate.
		if allChildren(node, fullOrWeakDuplicate) {
			node.SimilarityType = WeakDuplicate
			return
		}
		if allChildren(node, unique) {
			node.SimilarityType = Unique
			return
		}
		if allChildren(node, uniqueOrPartiallyUnique) {
			node.SimilarityType = PartiallyUnique
			return
		}
		if someChildren(node, fullOrWeakDuplicate) &&
			someChildren(node, uniqueOrPartiallyUnique) &&
			noChildren(node, unknown) {
			node.SimilarityType = PartiallyUnique
			return
		}

		log.Printf("xxx %s (len %d)", node.FullPath(), len(node.Children))
		for _, ch := range node.Children {
			log.Printf("xxx %s %s", ch.SimilarityType, ch.FullPath())
		}

		node.SimilarityType = Unknown
	}

	updateSimilarityRec(node)
}

// Walk traverses the node tree. onNode return a boolean flagging if the function chould descend or not.
func Walk(root *Node, onNode func(*Node) bool) {
	var walkRec func(root *Node, onNode func(*Node) bool)

	walkRec = func(root *Node, onNode func(*Node) bool) {
		if shouldDescend := onNode(root); shouldDescend {
			for _, child := range root.Children {
				walkRec(child, onNode)
			}
		}
	}

	walkRec(root, onNode)
}
