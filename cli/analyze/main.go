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
	nodeLeft, err := loadNode(pathLeft)
	if err != nil {
		log.Fatalf("cannot load file %s: %v", pathLeft, err)
	}
	nodeRight, err := loadNode(pathRight)
	if err != nil {
		log.Fatalf("cannot load file %s: %v", pathRight, err)
	}

	log.Printf("size left: %s", libstrings.FormatBytes(nodeLeft.Size))
	log.Printf("size right: %s", libstrings.FormatBytes(nodeRight.Size))
	similars := analyze.FindSimilar(nodeLeft, nodeRight)
	log.Printf("found %d similarities", len(similars))
	for _, s := range similars {
		for _, t := range s.Similar {
			fmt.Printf("%d\t%d\t%s\t%s\n", s.Size/1024, s.FileCount, s.FullPath(), t.FullPath())
		}
	}
}

func loadNode(path string) (*analyze.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return analyze.LoadNodesFromFileList(f)
}
