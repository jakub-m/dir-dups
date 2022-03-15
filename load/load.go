// utilities to load nodes from files.

package load

import (
	"fmt"
	"greasytoad/analyze"
	"greasytoad/log"
	libstrings "greasytoad/strings"
	"io"
	"os"
)

func GetDefaultIgnoredFilesAndDirs() []string {
	return []string{"Thumbs.db", "._.DS_Store", ".DS_Store"}
}

func LoadNodesFromConctenatedFiles(paths []string, filesOrPathsToIgnore []string) *analyze.Node {
	readers := []io.Reader{}
	for _, path := range paths {
		r, err := os.Open(path)
		if err != nil {
			log.Fatalf("failed to open %s: %v", path, err)
		}
		defer r.Close()
		readers = append(readers, r)
	}
	multiReader := io.MultiReader(readers...)
	opts := analyze.LoadOpts{
		FilesOrDirsToIgnore: filesOrPathsToIgnore,
	}
	n, err := analyze.LoadNodesFromFileListOpts(multiReader, opts)
	if err != nil {
		log.Fatalf("Failed to load nodes: %v", err)
	}
	return n
}

func LoadNodesFromPaths(paths []string, filesDirsToIgnore []string) []*analyze.Node {
	loaded := []*analyze.Node{}
	for _, path := range paths {
		log.Printf("loading: %s", path)
		node, err := loadNode(path, filesDirsToIgnore)
		if err != nil {
			log.Fatalf("cannot load file %s: %v", path, err)
		}
		log.Printf("size: %s", libstrings.FormatBytes(node.Size))
		loaded = append(loaded, node)
	}
	return loaded
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

func RenameRoots(nodes ...*analyze.Node) {
	if len(nodes) == 1 {
		return
	}
	letters := "abcdefghijklmnopqrstuvxyz"
	if len(nodes) > len(letters) {
		log.Fatalf("input too large, max %d entries", len(letters))
	}
	for i, node := range nodes {
		node.Name = fmt.Sprintf("%c", letters[i])
	}
}
