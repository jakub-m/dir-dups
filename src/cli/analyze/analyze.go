package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type Node struct {
	Name     string
	Size     int
	Hash     hash
	Children map[string]*Node // node name to node
	similar  []*Node
}

type hash []byte

func (h hash) Equal(other hash) bool {
	return bytes.Equal(h, other)
}

func (n *Node) addSimilar(other *Node) {
	n.similar = append(n.similar, other)
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
				n.Children[p] = newChild
				newChild.Size = parsed.size
			} else {
				if p == "" {
					continue
				}
				if ch, ok := n.Children[p]; ok {
					n = ch
				} else {
					newChild := newNode(p)
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
		size := 0
		for _, ch := range node.Children {
			updateSizeRec(ch)
			size += ch.Size
		}
		node.Size = size
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
	h := md5.New()
	io.WriteString(h, fmt.Sprintf("%s %d", node.Name, node.Size))

	children := []*Node{}
	for _, ch := range node.Children {
		children = append(children, ch)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})
	for _, ch := range children {
		h.Write(ch.Hash)
	}
	return h.Sum(nil)
}

func newNode(name string) *Node {
	return &Node{
		Name:     name,
		Children: make(map[string]*Node),
		similar:  []*Node{},
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
	indexLeft := indexNodes(left)
	indexRight := indexNodes(right)

	// O(n^2) ahead
	for _, left := range indexLeft {
		for _, right := range indexRight {
			if isSimilar(left, right) {
				left.addSimilar(right)
				right.addSimilar(left)
			}
		}
	}

	similar := []*Node{}
	walk(left, func(n *Node) bool {
		if len(n.similar) == 0 {
			return true
		}
		similar = append(similar, n)
		return false // don't continue if found similar
	})

	return similar
}

func indexNodes(root *Node) []*Node {
	nodes := []*Node{}
	walk(root, func(n *Node) bool {
		nodes = append(nodes, n)
		return true
	})
	return nodes
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

func isSimilar(left, right *Node) bool {
	// Here one can implement not-exact similarity. For now, depend on hash.
	return left.Hash.Equal(right.Hash)
}
