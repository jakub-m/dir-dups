package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	strings "greasytoad/strings"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	gopath "path"
	"sort"
)

const (
	sampleHashSize = 1024
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
			h, err := getFullHash(path, info, sampleHashSize)
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
	startPath string
	debug     bool
}

func getOptions() options {
	opts := options{}
	flag.BoolVar(&opts.debug, "v", false, "verbose logging")
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Println("expected dir path as a first argument")
		os.Exit(1)
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

// getSampleHash returns hash of a small part of the file in the middle.
func getSampleHash(path string, info fs.FileInfo, sampleSize int64) (hash, error) {
	buf, err := readSample(path, info, sampleSize)
	if err != nil {
		return "?", err
	}
	h := calculateHash(buf)
	return h, nil
}

func getFullHash(path string, info fs.FileInfo, sampleSize int64) (hash, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "?", err
	}
	h := calculateHash(buf)
	return h, nil
}

func readSample(path string, info fs.FileInfo, sampleSize int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fileSize := info.Size()

	var offset int64 = 0
	if fileSize > sampleSize {
		offset = (fileSize - sampleSize) / 2
	}
	_, err = f.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, sampleSize)
	nRead, err := f.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:nRead], nil
}

type hash string

func calculateHash(buf []byte) hash {
	h := md5.New()
	h.Write(buf)
	return hash(fmt.Sprintf("%x", h.Sum(nil)))
}
