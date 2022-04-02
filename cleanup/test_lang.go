package cleanup

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinlang(t *testing.T) {
	input := strings.NewReader(`#
keep	h111	foo
keep	h111	bar
keep	h111	baz
#
`)
	minilang := strings.NewReader(`#
if "foo" and "bar" as x then keep x
`)
	output, err := ProcessManifest(minilang, input)
	assert.NoError(t, err)
	expected := strings.TrimSpace(``)
	assert.Equal(t, expected, output)
}
