package main

import (
	"flag"
	"fmt"
	"greasytoad/analyze"
	"greasytoad/log"
	libstrings "greasytoad/strings"
	"io"
	"os"
)

const (
	KB = 1 << 10
)

func main() {
	opts := getOptions()
	pathLeft, pathRight := opts.paths[0], opts.paths[1]
	log.Printf("loading: %s", pathLeft)
	nodeLeft, err := loadNode(pathLeft, opts.ignoreUnimportant)
	if err != nil {
		log.Fatalf("cannot load file %s: %v", pathLeft, err)
	}
	log.Printf("loading: %s", pathRight)
	nodeRight, err := loadNode(pathRight, opts.ignoreUnimportant)
	if err != nil {
		log.Fatalf("cannot load file %s: %v", pathRight, err)
	}

	log.Printf("size left: %s", libstrings.FormatBytes(nodeLeft.Size))
	log.Printf("size right: %s", libstrings.FormatBytes(nodeRight.Size))
	analyze.AnalyzeDuplicates(nodeLeft, nodeRight)

	var shouldDescend func(*analyze.Node) bool
	if opts.printAll {
		shouldDescend = func(n *analyze.Node) bool { return true }
	} else {
		shouldDescend = func(node *analyze.Node) bool {
			return !(node.SimilarityType == analyze.FullDuplicate || node.SimilarityType == analyze.Unique)
		}
	}

	analyze.Walk(nodeLeft, func(node *analyze.Node) bool {
		if len(node.Similar) == 0 {
			printNode(os.Stdout, node)
		} else {
			printNodeWithSimilar(os.Stdout, node)
		}
		// If a node is a full duplicate then do not descend into children, because the
		// childeren must be full duplicates as well. Same with uniques.
		return shouldDescend(node)
	})
	analyze.Walk(nodeRight, func(node *analyze.Node) bool {
		// Print only nodes that do not have similarities. those with similarities were already
		// duplicates of nodeLeft tree.
		if len(node.Similar) == 0 {
			printNode(os.Stdout, node)
			// } else {
			// 	printNodeWithSimilar(os.Stdout, node)
		}
		return shouldDescend(node)
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
	flag.BoolVar(&opts.debug, "debug", false, "Debug")
	flag.BoolVar(&opts.printAll, "print-all", false, "Print all paths. The alternative is to not descend to directories that are all full duplicates or unique.")
	flag.BoolVar(&opts.ignoreUnimportant, "ignore-unimportant", true, "Ignore unimportant files like DS_Store")
	flag.Parse()
	if len(flag.Args()) != 2 {
		log.Fatalf("expecting exactly two arguments, paths with the lists")
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
func printNode(w io.Writer, node *analyze.Node) {
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\n",
		node.SimilarityType,
		node.Size/KB,
		node.FileCount,
		node.FullPath(),
	)
}

func printNodeWithSimilar(w io.Writer, node *analyze.Node) {
	for _, sim := range node.Similar {
		fmt.Fprintf(w,
			"%s\t%d\t%d\t%s\t%s\n",
			node.SimilarityType,
			node.Size/KB,
			node.FileCount,
			node.FullPath(),
			sim.FullPath(),
		)
	}
}
