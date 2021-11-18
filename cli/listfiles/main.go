package main

import (
	"flag"
	"fmt"
	libhash "greasytoad/hash"
	strings "greasytoad/strings"
	"io/fs"
	"io/ioutil"
	"log"
	gopath "path"
	"sort"
)

var debugEnabled = false

func main() {
	opts := getOptions()
	debugEnabled = opts.debug

	ignoredCount := 0
	fileCount := 0
	var totalSize int64 = 0
	printFileInfo := func(path string, info fs.FileInfo) error {
		switch {
		case info.Mode().IsRegular():
			size := info.Size()
			h, err := opts.hashFunction(path, info)
			if err != nil {
				return err
			}
			fmt.Printf("%s\t%d\t%s\n", path, size, h)
			totalSize += size
			fileCount++
		default:
			logDebug("not a file, ignoring: %s", path)
			ignoredCount++
		}
		return nil
	}

	logInfo("start at: %s", opts.startPath)
	if err := listFilesRec(opts.startPath, printFileInfo); err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	logInfo("ignored: %d", ignoredCount)
	logInfo("file count: %d", fileCount)
	logInfo("total file size: %s (%d)", formatSize(totalSize), totalSize)
}

type options struct {
	startPath    string
	debug        bool
	hashFunction libhash.FileHashFunc
}

const (
	hashFuncOptionFull     = "h"
	hashFuncOptionSample   = "s"
	hashFuncOptionNameSize = "n"
)

func getOptions() options {
	opts := options{}
	flag.BoolVar(&opts.debug, "v", false, "verbose logging")
	var hashFuncSelect string
	flag.StringVar(&hashFuncSelect, "x", hashFuncOptionFull,
		fmt.Sprintf("hash options. (%s) full file, (%s) sample from the middle of the file, name and size, and (%s) name and size only",
			hashFuncOptionFull, hashFuncOptionSample, hashFuncOptionNameSize))
	flag.Parse()

	switch hashFuncSelect {
	case hashFuncOptionFull:
		opts.hashFunction = libhash.GetFullContentHash
	case hashFuncOptionSample:
		opts.hashFunction = libhash.GetSampleHash
	case hashFuncOptionNameSize:
		opts.hashFunction = libhash.GetNameSizeHash
	default:
		log.Fatalf("bad hash option: %s", hashFuncSelect)
	}
	if len(flag.Args()) != 1 {
		log.Fatal("expected dir path as a first argument")
	}
	opts.startPath = flag.Arg(0)
	return opts
}

func listFilesRec(path string, onFile func(string, fs.FileInfo) error) error {
	infos, err := ioutil.ReadDir(path)
	logDebug("got %d items in dir %s", len(infos), path)
	if err != nil {
		return fmt.Errorf("listFilesRec: error on %s: %v", path, err)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name() < infos[j].Name()
	})
	for _, info := range infos {
		switch {
		case info.IsDir():
			dirPath := gopath.Join(path, info.Name())
			if err := listFilesRec(dirPath, onFile); err != nil {
				logInfo("error: %v", err) // e.g. permission denied
				continue
			}
		default:
			filePath := gopath.Join(path, info.Name())
			if err := onFile(filePath, info); err != nil {
				return fmt.Errorf("error on file: %s: %v", filePath, err)
			}
		}
	}
	return nil
}

func logInfo(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func logDebug(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf(format, args...)
	}
}

func formatSize(size int64) string {
	return strings.FormatBytes(int(size))
}
