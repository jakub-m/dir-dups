package main

import (
	"flag"
	"fmt"
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
	printSize := func(path string, info fs.FileInfo) error {
		switch {
		case info.Mode().IsRegular():
			size := info.Size()
			fmt.Printf("%s\t%d\n", path, size)
			totalSize += size
			fileCount++
		default:
			logDebug("not a file, ignoring: %s", path)
			ignoredCount++
		}
		return nil
	}

	logInfo("start at: %s", opts.startPath)
	if err := listFilesRec(opts.startPath, printSize); err != nil {
		logInfo("ERROR: %v", err)
	}
	logInfo("ignored: %d", ignoredCount)
	logInfo("file count: %d", fileCount)
	logInfo("total file size: %s (%d)", formatSize(totalSize), totalSize)
}

type options struct {
	startPath string
	debug     bool
}

func getOptions() options {
	opts := options{}
	flag.BoolVar(&opts.debug, "v", false, "debug logging")
	flag.StringVar(&opts.startPath, "p", ".", "path where to start listing the files")
	flag.Parse()
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

const (
	_ = 1 << (10 * iota)
	KB
	MB
	GB
)

func formatSize(size int64) string {
	var f float32 = float32(size)
	u := "B"
	switch {
	case size >= GB:
		f, u = f/GB, "GB"
	case size >= MB:
		f, u = f/MB, "MB"
	case size >= KB:
		f, u = f/KB, "KB"
	}
	return fmt.Sprintf("%.1f%s", f, u)
}
