package main

import (
	"fmt"
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

	log.Printf("size left: %dB", nodeLeft.Size)
	log.Printf("size right: %dB", nodeRight.Size)
	similars := FindSimilar(nodeLeft, nodeRight)
	log.Printf("found %d similarities", len(similars))
	for _, s := range similars {
		for _, t := range s.similar {
			fmt.Printf("%s\t%s", s.Name, t.Name)
		}
	}
}

func loadNode(path string) (*Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadNodesFromFileList(f)
}
