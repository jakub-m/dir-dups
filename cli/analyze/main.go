package main

import (
	"flag"
	"fmt"
	"greasytoad/analyze"
	"greasytoad/log"
	libstrings "greasytoad/strings"
	"os"
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
	pathLeft := opts.paths[0]
	log.Printf("loading: %s", pathLeft)
	nodeLeft, err := loadNode(pathLeft, opts.ignoreUnimportant)
	if err != nil {
		log.Fatalf("cannot load file %s: %v", pathLeft, err)
	}

	log.Printf("size left: %s", libstrings.FormatBytes(nodeLeft.Size))

	tree := nodeLeft

	analyze.FindSimilarities(tree, func(similarity analyze.SimilarityType, nodes []*analyze.Node) {
		names := []string{}
		for _, n := range nodes {
			names = append(names, n.FullPath())
		}
		//fmt.Printf("%s\t%s\t%s\n", similarity, nodes[0].Hash, strings.Join(names, "\t"))
		fmt.Printf("%s\t%s\n", similarity, strings.Join(names, "\t"))
	})
}

type options struct {
	printAll          bool
	ignoreUnimportant bool
	paths             []string
	debug             bool
}

func getOptions() options {
	opts := options{}
	flag.BoolVar(&opts.debug, "d", false, "Debug")
	flag.BoolVar(&opts.printAll, "p", false, "Print all paths. The alternative is to not descend to directories that are all full duplicates or unique.")
	flag.BoolVar(&opts.ignoreUnimportant, "ignore-unimportant", true, "Ignore unimportant files like DS_Store")
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatalf("expecting one argument with path with the list")
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

func nameRoots(nodes ...*analyze.Node) {
	if len(nodes) > 1 {
		return
	}
	letters := "abcdefghijklmn"
	for i, node := range nodes {
		node.Name = fmt.Sprintf("%c", letters[i])
	}
}

func mergeNodesIntoSingleTree(nodes ...*analyze.Node) *analyze.Node {
	root := analyze.NewNode("")
	for _, node := range nodes {
		root.Children[node.Name] = node
	}
	return root
}
