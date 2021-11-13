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
	Name      string
	Size      int
	FileCount int
	Hash      hash
	Children  map[string]*Node // map children node name to node
	Similar   []*Node          `json:"-"`
	Parent    *Node            `json:"-"`
}

func (n *Node) addSimilar(other *Node) {
	n.Similar = append(n.Similar, other)
}

func (n *Node) FullPath() string {
	parts := []string{}

	for n != nil {
		parts = append(parts, n.Name)
		n = n.Parent
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "/")
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
		if len(node.Children) == 0 {
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

func calculateHash(node *Node) hash {
	h := fnv.New64a()
	if len(node.Children) == 0 {
		// a file derives the hash from name and size
		io.WriteString(h, fmt.Sprintf("%s %d", node.Name, node.Size))
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
		for _, ch := range children {
			b := []byte(strconv.Itoa(int(ch.Hash)))
			h.Write(b)
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
}

func parseLine(line string) (parsed, error) {
	line = strings.Trim(line, "\n")
	parts := strings.Split(line, "\t")
	parsed := parsed{}
	if len(parts) != 2 {
		return parsed, fmt.Errorf("bad line: `%v`", line)
	}
	parsed.path = strings.Split(parts[0], "/")
	size, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return parsed, err
	}
	parsed.size = int(size)
	return parsed, nil
}

func FindSimilar(left *Node, right *Node) []*Node {

	indexLeft := indexNodesByHash(left)
	indexRight := indexNodesByHash(right)

	log.Printf("hashes: left %d, right %d", len(indexLeft), len(indexRight))
	leftOnly, overlap, rightOnly := findHashOverlap(indexLeft, indexRight)
	log.Printf("hashes: left only %d, overlap %d, right only %d", len(leftOnly), len(overlap), len(rightOnly))

	for h := range overlap {
		for _, leftNode := range indexLeft[h] {
			for _, rightNode := range indexRight[h] {
				leftNode.addSimilar(rightNode)
				rightNode.addSimilar(leftNode)
			}
		}
	}

	similar := []*Node{}
	walk(left, func(n *Node) bool {
		if len(n.Similar) == 0 {
			return true
		}
		similar = append(similar, n)
		return false // don't continue if found similar
	})

	return similar
}

func indexNodesByHash(root *Node) map[hash][]*Node {
	m := make(map[hash][]*Node)
	walk(root, func(n *Node) bool {
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

func walk(root *Node, onNode func(*Node) bool) {
	var walkRec func(root *Node, onNode func(*Node) bool)

	walkRec = func(root *Node, onNode func(*Node) bool) {
		if shouldDescent := onNode(root); shouldDescent {
			for _, child := range root.Children {
				walkRec(child, onNode)
			}
		}
	}

	walkRec(root, onNode)
}
