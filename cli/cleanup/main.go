// cleanup cli suggests which duplicates an be safely removed.

package main

import (
	_ "embed"
	"flag"
	"fmt"
	"greasytoad/analyze"
	"greasytoad/cleanup"
	"greasytoad/cleanup/parser"
	coll "greasytoad/collections"
	libflag "greasytoad/flag"
	"greasytoad/load"
	"greasytoad/log"
	"greasytoad/strings"
	"io"
	"os"
	"path"
	gostrings "strings"
	"text/template"
)

func main() {
	opts := getOptions()
	log.DebugEnabled = opts.debug
	log.Debugf("options: %+v", opts)
	if len(opts.pathsToFileLists) > 0 {
		processListfilesToManifest(opts)
	} else if opts.scriptFile != "" {
		transformManifestWithScript(opts)
	} else {
		transformManifestToBash(opts)
	}
}

type options struct {
	debug                bool
	ignoreFilesOrDirs    []string
	pathsToFileLists     []string
	manifestFile         string
	targetPrefix         string
	targetPrefixToRemove string
	useCopyRemove        bool
	scriptFile           string
}

func getOptions() options {
	opts := options{
		ignoreFilesOrDirs: load.GetDefaultIgnoredFilesAndDirs(),
	}
	flag.BoolVar(&opts.debug, "d", false, "Debug logging")
	flag.Var(libflag.NewStringList(&opts.pathsToFileLists), "l", "Path to result of \"listfiles\" command. Can be set many times.")
	flag.StringVar(&opts.manifestFile, "m", "", "Path to manifest file. If listfiles are not set, then this command will parse manifest file and return bash script")
	flag.StringVar(&opts.scriptFile, "s", "", "Path to script to parse and modify the manifest file")
	flag.StringVar(&opts.targetPrefix, "t", "", "Target directory for moving the files")
	flag.StringVar(&opts.targetPrefixToRemove, "p", "", "Common prefix to remove for target directories")
	flag.BoolVar(&opts.useCopyRemove, "cp", false, "Use cp and rm instead of mv in case of \"Operation not supported\" error")
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
			fmt.Fprintf(manifestFile, "%s\t%s\t%s\n", parser.Keep, n.Hash, n.FullPath())
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

func transformManifestWithScript(opts options) {
	if opts.scriptFile == "" {
		log.Fatalf("set script file")
	}
	script, err := loadScriptFromFile(opts.scriptFile)
	if err != nil {
		log.Fatalf("error loading script file %s: %s", opts.scriptFile, err)
	}

	var manifestReader io.ReadCloser
	if opts.manifestFile == "-" || opts.manifestFile == "" {
		manifestReader = os.Stdin
	} else {
		r, err := os.Open(opts.manifestFile)
		manifestReader = r
		if err != nil {
			log.Fatalf("failed to load manifest %s: %v", opts.manifestFile, err)
		}
		defer manifestReader.Close()
	}

	err = cleanup.ProcessManifestWithScript(manifestReader, script, os.Stdout)
	if err != nil {
		log.Fatalf("error pocessing manifest: %s", err)
	}
}

func transformManifestToBash(opts options) {
	if opts.targetPrefix == "" {
		log.Fatalf(`Set target path with "-t"`)
	}

	log.Debugf("transformManifestToBash: open manifest file \"%s\"", opts.manifestFile)
	manifest, err := loadManifestFromFile(opts.manifestFile)

	if err != nil {
		log.Fatalf("failed to parse manifest %s: %v", opts.manifestFile, err)
	}

	verifyOneKeepPerHash(manifest)
	verifyPathCommonPrefix(manifest, opts.targetPrefixToRemove)

	tmpl, err := template.New("bashTemplate").Parse(templateBody)
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
	dataEntries := coll.TransformSlice(manifest, func(m cleanup.ManifestEntry) DataEntry {
		return DataEntry{
			ManifestEntry: m,
			TargetPath:    path.Join(opts.targetPrefix, path.Dir(removeCommonPrefix(opts.targetPrefixToRemove, path.Clean(m.Path)))),
		}
	})

	getTargetPath := func(s DataEntry) string { return s.TargetPath }
	isMove := func(s DataEntry) bool { return s.Operation == parser.Move }
	err = tmpl.Execute(os.Stdout, Data{
		UseCpRm:     opts.useCopyRemove,
		Entries:     dataEntries,
		TargetPaths: coll.Uniq(coll.TransformSlice(coll.FilterSlice(dataEntries, isMove), getTargetPath)),
	})
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
}

type Data struct {
	UseCpRm     bool
	Entries     []DataEntry
	TargetPaths []string
}

type DataEntry struct {
	cleanup.ManifestEntry
	TargetPath string
}

func verifyOneKeepPerHash(manifest cleanup.Manifest) {
	hashHasKeep := make(map[string]bool)
	for _, m := range manifest {
		hashHasKeep[m.Hash] = hashHasKeep[m.Hash] || (m.Operation == parser.Keep)
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

func verifyPathCommonPrefix(manifest cleanup.Manifest, prefix string) {
	if prefix == "" {
		return
	}
	for _, m := range manifest {
		pathWitoutPrefix := removeCommonPrefix(prefix, m.Path)
		if pathWitoutPrefix == m.Path {
			log.Fatalf(`Path does not have common specified prefix: path "%s", prefix "%s"`, m.Path, prefix)
		}
	}
}

func removeCommonPrefix(prefix string, pathToModify string) string {
	if prefix == "" {
		return pathToModify
	}
	cleanPath := path.Clean(pathToModify)
	cleanPrefix := path.Clean(prefix)
	if gostrings.HasPrefix(cleanPath, cleanPrefix) {
		return cleanPath[len(cleanPrefix):]
	}
	return pathToModify
}

//go:embed bash.gotemplate
var templateBody string

func loadScriptFromFile(path string) (cleanup.Script, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return cleanup.ReadScript(f)
}

func loadManifestFromFile(path string) (cleanup.Manifest, error) {
	var manifestFile *os.File
	if path == "-" || path == "" {
		manifestFile = os.Stdin
	} else {
		r, err := os.Open(path)
		manifestFile = r
		if err != nil {
			log.Fatalf("failed to load manifest %s: %v", path, err)
		}
		defer manifestFile.Close()
	}

	return cleanup.ReadManifest(manifestFile)
}
