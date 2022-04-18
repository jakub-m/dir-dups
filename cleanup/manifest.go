package cleanup

import (
	"bufio"
	_ "embed"
	"fmt"
	"greasytoad/cleanup/parser"
	"io"
	"regexp"
	gostrings "strings"
)

func ReadManifest(r io.Reader) (Manifest, error) {
	return ReadManifestCallPerLine(r, func(i int, s string, me ManifestEntry) error { return nil })
}

func ReadManifestCallPerLine(r io.Reader, callback func(nLine int, line string, me ManifestEntry) error) (Manifest, error) {
	manifest := Manifest{}
	s := bufio.NewScanner(r)
	nLine := 0
	for s.Scan() {
		nLine++
		line := s.Text()
		if gostrings.HasPrefix(line, "#") {
			continue
		}
		me, err := ParseLineToManifestEntry(line)
		if err != nil {
			return manifest, fmt.Errorf("illegal line %d: '%s'", nLine, line)
		}
		if err = callback(nLine, line, me); err != nil {
			return manifest, err
		}
		manifest = append(manifest, me)
	}
	if err := s.Err(); err != nil {
		return Manifest{}, fmt.Errorf("Error around line %d: %v", nLine, err)
	}
	return manifest, nil
}

func ParseLineToManifestEntry(line string) (ManifestEntry, error) {
	submatches := manifestLineRegex.FindStringSubmatch(line)
	if submatches == nil {
		return ManifestEntry{}, fmt.Errorf("not a manifest entry")
	}
	me := ManifestEntry{
		Operation: parser.ManifestOperation(submatches[1]),
		Hash:      submatches[2],
		Path:      submatches[3],
	}
	return me, nil
}

var manifestLineRegex = regexp.MustCompile(`^(keep|move)\t(\S+)\t(.+)$`)

type Manifest []ManifestEntry

type ManifestEntry struct {
	Operation parser.ManifestOperation
	Hash      string
	Path      string
}

func (me ManifestEntry) String() string {
	return fmt.Sprintf("%s\t%s\t%s", me.Operation, me.Hash, me.Path)
}
