// cleanup cli suggests which duplicates an be safely removed.

package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"greasytoad/analyze"
	coll "greasytoad/collections"
	libflag "greasytoad/flag"
	"greasytoad/load"
	"greasytoad/log"
	"greasytoad/strings"
	"io"
	"os"
	"regexp"
	gostrings "strings"
	"text/template"
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
	return opts
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
	// Parse existing manifest
	// Verify that for each hash there is at least one "keep"
	// Produce bash file that can be safely executed.

	f, err := os.Open(opts.manifestFile)
	if err != nil {
		log.Fatalf("failed to load manifest %s: %v", opts.manifestFile, err)
	}
	defer f.Close()
	manifest, err := parseManifest(f)
	if err != nil {
		log.Fatalf("failed to parse manifest %s: %v", opts.manifestFile, err)
	}

	verifyOneKeepPerHash(manifest)

	tmpl, err := template.New("bashTemplate").Parse(templateBody)
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
	data := struct{ Manifest Manifest }{manifest}
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
}

func verifyOneKeepPerHash(manifest Manifest) {
	hashHasKeep := make(map[string]bool)
	for _, m := range manifest {
		hashHasKeep[m.Hash] = hashHasKeep[m.Hash] || (m.Operation == Keep)
	}
	shouldFail := false
	for h, b := range hashHasKeep {
		if !b {
			log.Fatalf("No \"keep\" for: %s", h)
			shouldFail = true
		}
	}
	if shouldFail {
		log.Fatalf("There must be at least one \"keep\" for each hash. Aborting.")
	}
}

var manifestLineRegex = regexp.MustCompile(`^(keep|move)\t(\S+)\t(.+)$`)

func parseManifest(r io.Reader) (Manifest, error) {
	manifest := Manifest{}
	s := bufio.NewScanner(r)
	nLine := 0
	for s.Scan() {
		nLine++
		line := s.Text()
		if gostrings.HasPrefix(line, "#") {
			continue
		}
		submatches := manifestLineRegex.FindStringSubmatch(line)
		if submatches == nil {
			return manifest, fmt.Errorf("illegal line %d: '%s'", nLine, line)
		}
		me := ManifestEntry{
			Operation: ManifestOperation(submatches[1]),
			Hash:      submatches[2],
			Path:      submatches[3],
		}
		manifest = append(manifest, me)
	}
	if err := s.Err(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

type Manifest []ManifestEntry

type ManifestEntry struct {
	Operation ManifestOperation
	Hash      string
	Path      string
}

type ManifestOperation string

const (
	Keep ManifestOperation = "keep"
	Move                   = "move"
)

//go:embed bash.gotemplate
var templateBody string
