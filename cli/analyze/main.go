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
		node, err := loadNode(path, opts.ignoreUnimportant)
		if err != nil {
			log.Fatalf("cannot load file %s: %v", path, err)
		}
		log.Printf("size: %s", libstrings.FormatBytes(node.Size))
		inputNodes = append(inputNodes, node)
	}

	nameRoots(inputNodes...)
	tree := mergeNodesIntoSingleTree(inputNodes...)

	analyze.FindSimilarities(tree, func(similarity analyze.SimilarityType, nodes []*analyze.Node) {
		if opts.sort {
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].FullPath() < nodes[j].FullPath()
			})
		}

		names := []string{}
		for _, n := range nodes {
			names = append(names, n.FullPath())
		}
		if opts.printHash {
			fmt.Printf("%s\t%s\t%s\n", similarity, nodes[0].Hash, strings.Join(names, "\t"))
		} else {
			fmt.Printf("%s\t%s\n", similarity, strings.Join(names, "\t"))
		}
	})
}

type options struct {
	debug             bool
	ignoreUnimportant bool
	paths             []string
	// printAll          bool
	printHash bool
	profile   string
	sort      bool
}

func getOptions() options {
	opts := options{}
	flag.BoolVar(&opts.debug, "d", false, "Debug")
	flag.BoolVar(&opts.sort, "s", false, "Sort output. Might slow down significantly.")
	flag.BoolVar(&opts.printHash, "ph", false, "Print hashes")
	// flag.BoolVar(&opts.printAll, "p", false, "Print all paths. The alternative is to not descend to directories that are all full duplicates or unique.")
	flag.BoolVar(&opts.ignoreUnimportant, "ignore-unimportant", true, "Ignore unimportant files like DS_Store")
	flag.StringVar(&opts.profile, "pprof", "", "run profiling")
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatalf("expecting at least one argument with path with the list")
	}
	opts.paths = flag.Args()
	return opts
}

func loadNode(path string, ignoreUnimportant bool) (*analyze.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	filesToIgnore := []string{"Thumbs.db", "._.DS_Store", ".DS_Store"}
	if ignoreUnimportant {
		log.Printf("ignoring files: %v", filesToIgnore)
	}
	opts := analyze.LoadOpts{
		FilesToIgnore: filesToIgnore,
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
