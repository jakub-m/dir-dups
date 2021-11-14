package main

import (
	"fmt"
	"greasytoad/analyze"
	libstrings "greasytoad/strings"
	"io"
	"log"
	"os"
)

const (
	KB = 1 << 10
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("expecting exactly two arguments, paths with the lists")
	}

	pathLeft, pathRight := os.Args[1], os.Args[2]
	log.Printf("loading: %s", pathLeft)
	nodeLeft, err := loadNode(pathLeft)
	if err != nil {
		log.Fatalf("cannot load file %s: %v", pathLeft, err)
	}
	log.Printf("loading: %s", pathRight)
	nodeRight, err := loadNode(pathRight)
	if err != nil {
		log.Fatalf("cannot load file %s: %v", pathRight, err)
	}

	log.Printf("size left: %s", libstrings.FormatBytes(nodeLeft.Size))
	log.Printf("size right: %s", libstrings.FormatBytes(nodeRight.Size))
	analyze.AnalyzeDuplicates(nodeLeft, nodeRight)

	analyze.Walk(nodeLeft, func(node *analyze.Node) bool {
		if len(node.Similar) == 0 {
			printNode(os.Stdout, node)
		} else {
			printNodeWithSimilar(os.Stdout, node)
		}
		// If a node is a full duplicate then do not descend into children, because the
		// childeren must be full duplicates as well. Same with uniques.
		return !(node.SimilarityType == analyze.FullDuplicate || node.SimilarityType == analyze.Unique)
	})
}

func loadNode(path string) (*analyze.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return analyze.LoadNodesFromFileList(f)
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
