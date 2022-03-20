package analyze

import (
	"bufio"
	"fmt"
	"greasytoad/log"
	"hash/fnv"
	"io"
	"sort"
	"strconv"
	"strings"
)

type hash uint64

func (h hash) String() string {
	return fmt.Sprintf("%x", uint64(h))
}

type Node struct {
	Name           string
	Size           int
	FileCount      int
	Hash           hash
	Children       map[string]*Node // map children node name to node
	Parent         *Node            `json:"-"`
	cachedFullPath *string
	// IsRoot tells if the node is a root node. Mind that the root node is not a root of directory tree. Root node does not have any parent.
	IsRoot bool
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
	if n.cachedFullPath == nil {
		p := n.getFullPath()
		n.cachedFullPath = &p
	}
	return *n.cachedFullPath
}

func (n *Node) getFullPath() string {
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

func (n *Node) ChildrenSlice() []*Node {
	children := []*Node{}
	for _, ch := range n.Children {
		children = append(children, ch)
	}
	return children
}

func LoadNodesFromFileList(data io.Reader) (*Node, error) {
	return LoadNodesFromFileListOpts(data, LoadOpts{})
}

type LoadOpts struct {
	FilesOrDirsToIgnore []string
}

// LoadNodesFromFileListOpts loads Nodes from a file with list of files with sizes and signatures, a result of "listfiles" tool.
func LoadNodesFromFileListOpts(data io.Reader, opts LoadOpts) (*Node, error) {
	filesOrDirsToIgnore := make(map[string]bool)
	if opts.FilesOrDirsToIgnore != nil {
		for _, name := range opts.FilesOrDirsToIgnore {
			filesOrDirsToIgnore[name] = true
		}
	}

	shouldIgnorePath := func(chunkedPath []string) (string, bool) {
		for _, chunk := range chunkedPath {
			if filesOrDirsToIgnore[chunk] {
				return chunk, true
			}
		}
		return "", false
	}

	scanner := bufio.NewScanner(data)

	root := NewRootNode()

	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \r\n")
		if len(line) == 0 {
			// Ignore empty lines. This is useful especially when concatenating input files.
			continue
		}
		parsed, err := parseLine(line)
		if err != nil {
			return nil, err
		}

		if match, ok := shouldIgnorePath(parsed.path); ok {
			log.Debugf("ignore %s because of %s", parsed.fullPath, match)
			continue
		}

		n := root
		for i, p := range parsed.path {
			if i == len(parsed.path)-1 {
				// last, that is the file
				newChild := NewNode(p)
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
					newChild := NewNode(p)
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

	updateTreeValues(root)
	return root, nil
}

func MergeNodes(nodes ...*Node) *Node {
	root := NewRootNode()
	for _, node := range nodes {
		if _, exists := root.Children[node.Name]; exists {
			log.Fatalf("Cannot merge nodes with same name, consider renaming. Offending name: '%s'", node.Name)
		}
		root.Children[node.Name] = node
	}
	updateTreeValues(root)
	return root
}

func updateTreeValues(root *Node) {
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

func NewNode(name string) *Node {
	return &Node{
		Name:     name,
		Children: make(map[string]*Node),
	}
}

// NewRootNode returns a node that is a root to all the node tree. This is NOT a root directort. The root directory
// would be a single node under the root node.
func NewRootNode() *Node {
	r := NewNode("")
	r.IsRoot = true
	return r
}

type parsed struct {
	path     []string
	fullPath string
	size     int
	hash     string
}

func parseLine(line string) (parsed, error) {
	line = strings.Trim(line, "\n")
	parts := strings.Split(line, "\t")
	parsed := parsed{}
	if len(parts) != 3 {
		return parsed, fmt.Errorf("bad line: %d parts, `%v`", len(parts), line)
	}
	parsed.fullPath = parts[0]
	parsed.path = strings.Split(parsed.fullPath, "/")
	size, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return parsed, err
	}
	parsed.size = int(size)
	parsed.hash = parts[2]
	return parsed, nil
}

type AnalizeOpts int32

const (
	None AnalizeOpts = 0
	// OptimizeSimilarities removes redundant similarities. DOES NOT WORK.
	OptimizeSimilarities = (1 << iota)
)

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

func WalkAll(root *Node, onNode func(*Node)) {
	Walk(root, func(n *Node) bool {
		onNode(n)
		return true
	})
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

func FormatNodes(nodes []*Node, formatFunc func(*Node) string) []string {
	strings := []string{}
	for _, n := range nodes {
		strings = append(strings, formatFunc(n))
	}
	return strings
}
