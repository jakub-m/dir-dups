package cleanup

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"regexp"
	gostrings "strings"
)

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

var manifestLineRegex = regexp.MustCompile(`^(keep|move)\t(\S+)\t(.+)$`)

func ParseManifest(r io.Reader) (Manifest, error) {
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
		return Manifest{}, fmt.Errorf("Error around line %d: %v", nLine, err)
	}
	return manifest, nil
}
