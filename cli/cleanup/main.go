// cleanup cli suggests which duplicates an be safely removed.

package main

import (
	"flag"
	"fmt"
	"greasytoad/analyze"
	coll "greasytoad/collections"
	libflag "greasytoad/flag"
	"greasytoad/load"
	"greasytoad/log"
	"greasytoad/strings"
	"os"
)

func main() {
	opts := getOptions()
	log.DebugEnabled = opts.debug
	log.Debugf("options: %+v", opts)

	if len(opts.pathsToFileLists) == 0 {
		transformManifestToBash(opts)
	} else {
		processListfilesToManifest(opts)
	}
}

func processListfilesToManifest(opts options) {
	root := load.LoadNodesFromConctenatedFiles(opts.pathsToFileLists, opts.ignoreFilesOrDirs)
	log.Printf("merged input size: %s", strings.FormatBytes(root.Size))

	manifestFile := os.Stdout
	if opts.manifestFile != "" {
		f, err := os.Create(opts.manifestFile)
		if err != nil {
			log.Fatalf("Cannot open %s for writing: %v", opts.manifestFile, err)
		}
		manifestFile = f
		defer manifestFile.Close()
	}

	totalSavingBytes := 0
	analyze.FindSimilarities(root, func(st analyze.SimilarityType, nodes []*analyze.Node) {
		if st != analyze.FullDuplicate {
			return
		}
		isFile := func(n *analyze.Node) bool { return n.IsFile() }
		if coll.Any(nodes, isFile) {
			return
		}
		fileCount, size := -1, -1
		for _, n := range nodes {
			fmt.Fprintf(manifestFile, "keep\t%s\t%s\n", n.Hash, n.FullPath())
			if fileCount != -1 && fileCount != n.FileCount {
				log.Fatalf("RATS! the nodes reported as similar but have different file counts: %v", nodes)
			}
			if size != -1 && size != n.Size {
				log.Fatalf("RATS! the nodes reported as similar but have different sizes: %v", nodes)
			}
			fileCount, size = n.FileCount, n.Size
		}
		fmt.Fprintf(manifestFile, "# %d dirs, each %s in %d files\n", len(nodes), strings.FormatBytes(size), fileCount)
		fmt.Fprintf(manifestFile, "#\n")
		totalSavingBytes += (len(nodes) - 1) * size
	})

	fmt.Fprintf(manifestFile, "# Total %s of duplicates to remove\n", strings.FormatBytes(totalSavingBytes))
}

func transformManifestToBash(opts options) {

}

type options struct {
	debug             bool
	ignoreFilesOrDirs []string
	pathsToFileLists  []string
	manifestFile      string
}

func getOptions() options {
	opts := options{
		ignoreFilesOrDirs: load.GetDefaultIgnoredFilesAndDirs(),
	}
	flag.Var(libflag.NewStringList(&opts.pathsToFileLists), "l", "Path to result of \"listfiles\" command. Can be set many times.")
	flag.StringVar(&opts.manifestFile, "m", "", "Path to manifest file. If listfiles are not set, then this command will parse manifest file and return bash script")
	flag.BoolVar(&opts.debug, "d", false, "Debug logging")
	flag.Parse()
	if len(opts.pathsToFileLists) < 1 {
		log.Fatalf("expecting at least one path to list of files.")
	}
	return opts
}
