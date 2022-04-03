package cleanup

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinlang(t *testing.T) {
	manifestString := `#
keep	h111	foo
keep	h111	bar
keep	h111	baz
`
	scriptString := `
if "foo" and "bar" as x then move x
`
	out := &strings.Builder{}

	script, err := ReadScript(strings.NewReader(scriptString))
	assert.NoError(t, err)

	err = ProcessManifestWithScript(strings.NewReader(manifestString), script, out)
	assert.NoError(t, err)

	expected := `#
keep	h111	foo
move	h111	bar
keep	h111	baz
`
	assert.Equal(t, expected, out.String())
}
