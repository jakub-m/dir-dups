package main

import (
	"flag"
	"fmt"
	"greasytoad/analyze"
	"greasytoad/log"
	libstrings "greasytoad/strings"
	"os"
	"runtime/pprof"
	"sort"
	"strings"
)

const (
	KB = 1 << 10
)

func main() {
	opts := getOptions()
	if opts.debug {
		log.DebugEnabled = true
	}
	log.Printf("Ignoring: %v", opts.ignoreFilesOrDirs)
	log.Debugf("options: %+v", opts)

	if opts.profile != "" {
		f, err := os.Create(opts.profile)
		if err != nil {
			log.Fatalf("%s", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

	}

	inputNodes := []*analyze.Node{}
	for _, path := range opts.paths {
		log.Printf("loading: %s", path)
		node, err := loadNode(path, opts.ignoreFilesOrDirs)
		if err != nil {
			log.Fatalf("cannot load file %s: %v", path, err)
		}
		log.Printf("size: %s", libstrings.FormatBytes(node.Size))
		inputNodes = append(inputNodes, node)
	}

	nameRoots(inputNodes...)
	tree := mergeNodesIntoSingleTree(inputNodes...)

	if opts.tree {
		printSimilarityTree(tree, opts)
	} else {
		printSimilarityFlat(tree, opts)
	}
}

func printSimilarityFlat(root *analyze.Node, opts options) {
	var nodePrinter func(analyze.SimilarityType, []*analyze.Node)
	if opts.verbose {
		nodePrinter = func(st analyze.SimilarityType, nodes []*analyze.Node) {
			names := formatNodesPaths(nodes)
			//fmt.Printf("%s\t%s\t%s\n", st, nodes[0].Hash, len(nodes), names)
			fmt.Printf("%s\t%d\t%s\n", st, len(nodes), names)
		}
	} else {
		nodePrinter = func(st analyze.SimilarityType, nodes []*analyze.Node) {
			names := formatNodesPaths(nodes)
			fmt.Printf("%s\t%s\n", st, names)
		}
	}

	analyze.FindSimilarities(root, func(similarity analyze.SimilarityType, nodes []*analyze.Node) {
		if opts.sort {
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].FullPath() < nodes[j].FullPath()
			})
		}
		nodePrinter(similarity, nodes)
	})
}

type nodeMeta struct {
	similarityType analyze.SimilarityType
	similar        []*analyze.Node
}

func printSimilarityTree(root *analyze.Node, opts options) {
	meta := make(map[*analyze.Node]nodeMeta)
	analyze.FindSimilarities(root, func(st analyze.SimilarityType, nodes []*analyze.Node) {
		for _, n := range nodes {
			meta[n] = nodeMeta{st, nodes}
		}
	})

	decorator := func(n *analyze.Node) string {
		if m, ok := meta[n]; ok {
			if m.similarityType == analyze.FullDuplicate {
				return fmt.Sprintf("\t[%s %dx%s %s]",
					m.similarityType,
					len(m.similar),
					libstrings.FormatBytes(n.Size),
					m.similar[0].Hash)
			} else {
				return fmt.Sprintf("\t[%s]", m.similarityType)
			}
		} else {
			return ""
		}
	}

	var nodeFilter func([]*analyze.Node) []*analyze.Node
	if opts.selectDuplicatedDirs {
		shouldPrint := getNodeSetForPrintTree(root, meta)
		nodeFilter = func(nodes []*analyze.Node) []*analyze.Node {
			selected := []*analyze.Node{}
			for _, node := range nodes {
				if ok := shouldPrint[node]; ok {
					selected = append(selected, node)
				}
			}
			return selected
		}
	} else {
		nodeFilter = nodeSelectorAll
	}

	printTree(getFirstNamedNode(root), decorator, nodeFilter)
}

func nodeSelectorAll(nodes []*analyze.Node) []*analyze.Node {
	return nodes
}

func printTree(root *analyze.Node,
	decorator func(*analyze.Node) string,
	nodeFilter func([]*analyze.Node) []*analyze.Node,
) {
	var printTreeRec func(*analyze.Node, string, string)
	printTreeRec = func(node *analyze.Node, immediatePrefix, spacePrefix string) {
		children := node.ChildrenSlice()
		children = nodeFilter(children)
		sort.Slice(children, func(i, j int) bool {
			return children[i].Name < children[j].Name
		})

		additional := decorator(node)
		fmt.Printf("%s%s%s\n", immediatePrefix, node.Name, additional)

		for i, ch := range children {
			isLast := len(children)-1 == i
			newImmediatePrefix, newSpacePrefix := "", ""
			if isLast {
				newImmediatePrefix = spacePrefix + "└──"
			} else {
				newImmediatePrefix = spacePrefix + "├──"
			}
			if isLast {
				newSpacePrefix = spacePrefix + "   "
			} else {
				newSpacePrefix = spacePrefix + "│  "
			}
			printTreeRec(ch, newImmediatePrefix, newSpacePrefix)
		}
	}

	printTreeRec(root, "", "")
}

// getNodeSetForPrintTree returns a node set that can be later used to determine which nodes should be printed in the tree. If a
func getNodeSetForPrintTree(root *analyze.Node, meta map[*analyze.Node]nodeMeta) map[*analyze.Node]bool {
	set := make(map[*analyze.Node]bool)

	var rec func(*analyze.Node)
	rec = func(current *analyze.Node) {
		for _, child := range current.Children {
			rec(child)
			if set[child] {
				// if a node is selected for printing, then recursively select all the nodes that lead to that node.
				set[current] = true
			}
		}

		if m, ok := meta[current]; ok && m.similarityType == analyze.FullDuplicate && !current.IsFile() {
			set[current] = true
		}

	}
	rec(root)

	return set
}

func loadNode(path string, filesOrPathsToIgnore []string) (*analyze.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opts := analyze.LoadOpts{
		FilesOrDirsToIgnore: filesOrPathsToIgnore,
	}
	return analyze.LoadNodesFromFileListOpts(f, opts)
}

func nameRoots(nodes ...*analyze.Node) error {
	if len(nodes) == 1 {
		return nil
	}
	letters := "abcdefghijklmnopqrstuvxyz"
	if len(nodes) > len(letters) {
		return fmt.Errorf("input too large, max %d entries", len(letters))
	}
	for i, node := range nodes {
		node.Name = fmt.Sprintf("%c", letters[i])
	}
	return nil
}

func mergeNodesIntoSingleTree(nodes ...*analyze.Node) *analyze.Node {
	root := analyze.NewNode("")
	for _, node := range nodes {
		root.Children[node.Name] = node
	}
	return root
}

func formatNodesPaths(nodes []*analyze.Node) string {
	fullPath := func(n *analyze.Node) string { return n.FullPath() }
	return strings.Join(analyze.FormatNodes(nodes, fullPath), "\t")
}

func getFirstNamedNode(node *analyze.Node) *analyze.Node {
	if node.Name == "" && len(node.Children) == 1 {
		for _, ch := range node.Children {
			return getFirstNamedNode(ch)
		}
	}
	return node
}

type options struct {
	debug                bool
	verbose              bool
	ignoreFilesOrDirs    []string
	paths                []string
	profile              string
	sort                 bool
	tree                 bool
	selectDuplicatedDirs bool
}

func getOptions() options {
	opts := options{
		ignoreFilesOrDirs: []string{"Thumbs.db", "._.DS_Store", ".DS_Store"},
	}
	flag.BoolVar(&opts.debug, "d", false, "Debug logging")
	flag.BoolVar(&opts.verbose, "v", false, "More verbose logging")
	flag.BoolVar(&opts.sort, "s", false, "Sort output")
	flag.BoolVar(&opts.tree, "t", false, "Print as tree")
	flag.BoolVar(&opts.selectDuplicatedDirs, "dd", false, "Select only duplicated directories")
	flag.Var(commaSplitter{&opts.ignoreFilesOrDirs}, "i", fmt.Sprintf("Names of files or directores to ignore (default %v)", opts.ignoreFilesOrDirs))
	flag.StringVar(&opts.profile, "pprof", "", "run profiling")
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatalf("expecting at least one argument with path with the list")
	}
	opts.paths = flag.Args()
	return opts
}

type commaSplitter struct {
	dest *[]string
}

func (s commaSplitter) Set(input string) error {
	*s.dest = strings.Split(input, ",")
	return nil
}

func (s commaSplitter) String() string {
	return "commaSplitter"
}
