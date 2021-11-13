package main

import (
	"fmt"
	"greasytoad/analyze"
	libstrings "greasytoad/strings"
	"log"
	"os"
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
			fmt.Printf(
				"%s\t%d\t%d\t%s\n",
				node.SimilarityType,
				node.Size/1024,
				node.FileCount,
				node.FullPath(),
			)
		} else {
			for _, sim := range node.Similar {
				fmt.Printf(
					"%s\t%d\t%d\t%s\t%s\n",
					node.SimilarityType,
					node.Size/1024,
					node.FileCount,
					node.FullPath(),
					sim.FullPath(),
				)

			}
		}
		return node.SimilarityType != analyze.FullDuplicate
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
