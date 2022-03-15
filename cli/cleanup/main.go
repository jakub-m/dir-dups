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
)

func main() {
	opts := getOptions()
	log.DebugEnabled = opts.debug
	log.Debugf("options: %+v", opts)

	root := load.LoadNodesFromConctenatedFiles(opts.pathsToFileLists, opts.ignoreFilesOrDirs)
	log.Printf("merged size: %s", strings.FormatBytes(root.Size))

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
			fmt.Printf("keep\t%s\t%s\n", n.Hash, n.FullPath())
			if fileCount != -1 && fileCount != n.FileCount {
				log.Fatalf("RATS! the nodes reported as similar but have different file counts: %v", nodes)
			}
			if size != -1 && size != n.Size {
				log.Fatalf("RATS! the nodes reported as similar but have different sizes: %v", nodes)
			}
			fileCount, size = n.FileCount, n.Size
		}
		fmt.Printf("# %d dirs, each %s in %d files\n", len(nodes), strings.FormatBytes(size), fileCount)
		fmt.Println("#")
		totalSavingBytes += (len(nodes) - 1) * size
	})

	fmt.Printf("# Total %s of duplicates to remove\n", strings.FormatBytes(totalSavingBytes))

	// re-run tool with updated manifest
	// dry-run, genereate bash commands if all good (all entries from manifest used AND all dirs in manifest)
	// run bash commands
}

type options struct {
	debug bool
	// verbose              bool
	ignoreFilesOrDirs []string
	// paths             []string
	// profile              string
	// sort                 bool
	// tree                 bool
	// selectDirs           bool
	// selectDuplicatedDirs bool
	pathsToFileLists []string
}

func getOptions() options {
	opts := options{
		ignoreFilesOrDirs: load.GetDefaultIgnoredFilesAndDirs(),
	}
	flag.Var(libflag.NewStringList(&opts.pathsToFileLists), "l", "Path to result of \"listfiles\" command. Can be set many times.")
	flag.BoolVar(&opts.debug, "d", false, "Debug logging")
	// flag.BoolVar(&opts.verbose, "v", false, "More verbose logging")
	//flag.Var(libflag.CommaSepValue{Value: &opts.ignoreFilesOrDirs}, "i", fmt.Sprintf("Comma separated of files or directores to ignore (default %+v)", opts.ignoreFilesOrDirs))
	//flag.Var(libflag.NewStringList(), "i", "Input directorwies")
	// flag.String(opts.ManifestPath "", "")
	// flag.Parse()
	// if len(flag.Args()) == 0 {
	// 	log.Fatalf("expecting at least one argument with path with the list")
	// }
	// opts.paths = flag.Args()
	flag.Parse()
	if len(opts.pathsToFileLists) < 1 {
		log.Fatalf("expecting at least one path to list of files.")
	}
	return opts
}
