package analyze

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindSimilarSimple(t *testing.T) {
	left := loadNodeFromString(t, `
/a1/b1/c1 1
/a1/b1/c2 2
/a1/b1/c3 3
`)

	right := loadNodeFromString(t, `
/a1/b1/c1 1
/a1/b1/c2 2
/a1/b1/c3 3
`)

	similar := FindSimilar(left, right)

	assert.Equal(t, len(similar), 1)
	assert.Equal(t, similar[0], left)
}

func TestFindSimilarOneFileInDifferentFolder(t *testing.T) {
	left := loadNodeFromString(t, `
/a1/b1/c1 1
/a1/b1/c2 2
`)

	right := loadNodeFromString(t, `
/a1/b1/c1 1
/a1/b1/c2 2
/a1/b2/c3 3
`)

	similar := FindSimilar(left, right)

	assert.Equal(t, len(similar), 1)
	assert.Equal(t, similar[0], left.Children["a1"].Children["b1"])
	assert.Equal(t, similar[0].FullPath(), "/a1/b1")
}

func TestLoadLines(t *testing.T) {
	r := bytes.NewBufferString("/foo/bar/baz\t1\n/foo/quux\t2")
	root, err := LoadNodesFromFileList(r)
	if j, err := json.MarshalIndent(root, "", " "); err == nil {
		fmt.Println(string(j))
	} else {
		fmt.Println(err)
	}

	assert.NoError(t, err)
	assert.Equal(t, "", root.Name)
	assert.Equal(t, 3, root.Size)
	assert.Equal(t, 2, root.FileCount)

	foo := root.Children["foo"]
	assert.Equal(t, "foo", foo.Name)
	assert.Equal(t, 3, foo.Size)
	assert.Equal(t, 2, foo.FileCount)

	bar := foo.Children["bar"]
	assert.Equal(t, "bar", bar.Name)
	assert.Equal(t, 1, bar.Size)
	assert.Equal(t, 1, bar.FileCount)

	baz := bar.Children["baz"]
	assert.Equal(t, "baz", baz.Name)
	assert.Equal(t, 1, baz.Size)
	assert.Equal(t, 1, baz.FileCount)

	quux := foo.Children["quux"]
	assert.Equal(t, "quux", quux.Name)
	assert.Equal(t, 2, quux.Size)
	assert.Equal(t, 1, quux.FileCount)
}

// test many slashes
// test starting with root being non-empty

func loadNodeFromString(t *testing.T, s string) *Node {
	s = strings.Trim(s, " \n")
	s = strings.ReplaceAll(s, " ", "\t")
	n, err := LoadNodesFromFileList(bytes.NewBufferString(s))
	if err != nil {
		t.Fatal(err)
	}
	return n
}
