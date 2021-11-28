package analyze

import (
	"bytes"
	"encoding/json"
	"fmt"
	"greasytoad/log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	log.DebugEnabled = true
}

func TestFindSimilarSimple(t *testing.T) {
	t.Skip()
	loadNodeFromString(t, `
/a1/b1/c1 1 c11
/a1/b1/c2 2 c22
/a1/b1/c3 3 c33
`)

	loadNodeFromString(t, `
/a1/b1/c1 1 c11
/a1/b1/c2 2 c22
/a1/b1/c3 3 c33
`)

	// AnalyzeDuplicates(left, right)

	// assert.Equal(t, 3, len(left.Similar))
	// assert.Contains(t, left.Similar, right)
	// assert.Contains(t, left.Similar, right.Children["a1"])
	// assert.Contains(t, left.Similar, right.Children["a1"].Children["b1"])
	// assert.Equal(t, FullDuplicate, left.SimilarityType)
}

func TestHash(t *testing.T) {
	node := loadNodeFromString(t, `
/a1/b1/c1 1 c11
/a1/b1/c2 2 c22
/a2/b2/c1 1 c11
`)
	// printNode(t, "node", node)

	assert.Equal(t,
		node.Children["a1"].Hash,
		node.Children["a1"].Children["b1"].Hash)
	assert.NotEqual(t,
		node.Children["a1"].Children["b1"].Children["c1"].Hash,
		node.Children["a1"].Children["b1"].Children["c2"].Hash)
	assert.Equal(t,
		node.Children["a1"].Children["b1"].Children["c1"].Hash,
		node.Children["a2"].Children["b2"].Children["c1"].Hash)
}

func TestFindSimilarOneFileInDifferentFolder(t *testing.T) {
	t.Skip()
	loadNodeFromString(t, `
/a1/b1/c1 1 c11
/a1/b1/c2 2 c22
`)

	loadNodeFromString(t, `
/a1/b1/c1 1 c11
/a1/b1/c2 2 c22
/a1/b2/c3 3 c33
`)

	// // printNode(t, "left", left)
	// // printNode(t, "right", left)
	// AnalyzeDuplicates(left, right)
	// assert.Equal(t, 1, len(left.Similar))
	// // assert.Equal(t, right.Children["a1"].Children["b1"], left.Similar[0])
	// // assert.Equal(t, similar[0], left.Children["a1"].Children["b1"])
	// // assert.Equal(t, similar[0].FullPath(), "/a1/b1")
}

func TestLoadLines(t *testing.T) {
	r := bytes.NewBufferString("/foo/bar/baz\t1\tb1\n/foo/quux\t2\tq2")
	root, err := LoadNodesFromFileList(r)
	// printNode(t, "node", root)

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

func TestNoUnknownSimilarity(t *testing.T) {
	t.Skip()
	loadNodeFromString(t, `
/a1/b1/c1 1 c11
/a1/b1/c2 1 c21
/a1/b2/c1 1 c11
/a1/b2/c10 10 c1010
`)

	loadNodeFromString(t, `
/a2/b1/c1 1 c11
/a2/b1/c2 1 c21
/a2/b2/c1 1 c11
/a2/b2/c2 1 c21
/a2/b2/c3 1 c31
`)

	// AnalyzeDuplicates(left, right)
	// Walk(left, func(n *Node) bool {
	// 	t.Logf("%s `%s`", n.SimilarityType, n.FullPath())
	// 	return true
	// })

	// Walk(left, func(n *Node) bool {
	// 	assert.NotEqual(t, Unknown, n.SimilarityType, "Unknown similarity type for: `%s`", n.FullPath())
	// 	return true
	// })
}

func loadNodeFromString(t *testing.T, s string) *Node {
	s = strings.Trim(s, " \n")
	s = strings.ReplaceAll(s, " ", "\t")
	n, err := LoadNodesFromFileList(bytes.NewBufferString(s))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func printNode(t *testing.T, label string, n *Node) {
	if j, err := json.MarshalIndent(n, "", " "); err == nil {
		t.Log(string(j))
		fmt.Printf("%s:\n%s\n", label, string(j))
	} else {
		t.Log(err)
	}
}
