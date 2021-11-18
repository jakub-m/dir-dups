package hash

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
)

const (
	sampleHashSize = 1024
	nilHash        = "?"
)

type FileHashFunc func(filePath string, fileInfo fs.FileInfo) (HashString, error)

type HashString string

func GetFullContentHash(filePath string, fileInfo fs.FileInfo) (HashString, error) {
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nilHash, err
	}
	h, err := calculateHash(buf)
	if err != nil {
		return nilHash, err
	}
	return HashString(fmt.Sprintf("h%s", h)), nil
}

// GetSampleHash returns hash of a small part of the file in the middle.
func GetSampleHash(path string, info fs.FileInfo) (HashString, error) {
	buf, err := readSample(path, info, sampleHashSize)
	if err != nil {
		return nilHash, err
	}
	h, err := calculateHash(buf)
	if err != nil {
		return nilHash, err
	}
	return HashString(fmt.Sprintf("s%s", h)), nil
}

func GetNameSizeHash(path string, info fs.FileInfo) (HashString, error) {
	s := fmt.Sprintf("%s+%d", info.Name(), info.Size())
	h, err := calculateHash([]byte(s))
	if err != nil {
		return nilHash, err
	}
	return HashString(fmt.Sprintf("n%s", h)), nil
}

func readSample(path string, info fs.FileInfo, sampleSize int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fileSize := info.Size()
	if fileSize == 0 {
		return []byte{}, nil
	}

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

func calculateHash(buf []byte) (HashString, error) {
	h := md5.New()
	if _, err := h.Write(buf); err != nil {
		return nilHash, err
	}
	return HashString(fmt.Sprintf("%x", h.Sum(nil))), nil
}
