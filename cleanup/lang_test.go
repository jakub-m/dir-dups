package cleanup

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinlang(t *testing.T) {
	IN := `#
keep	h111	foo
keep	h111	bar
keep	h111	baz
`
	tcs := []struct {
		name   string
		in     string
		script string
		out    string
	}{
		{
			name: "a",
			in:   IN,
			script: `
if "foo" and "bar" as x then move x
`,
			out: `#
keep	h111	foo
move	h111	bar
keep	h111	baz
`,
		},
		{
			name: "b",
			in:   IN,
			script: `
if "quux" as x then move x
`,
			out: IN,
		},
		{
			name: "c",
			in:   IN,
			script: `
if "bar" as y then move y
`,
			out: `#
keep	h111	foo
move	h111	bar
keep	h111	baz
`,
		},
		{
			name: "d",
			in:   IN,
			script: `
if "foo" as x then move x
if "bar" as y then move y
`,
			out: `#
move	h111	foo
move	h111	bar
keep	h111	baz
`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out := &strings.Builder{}

			script, err := ReadScript(strings.NewReader(tc.script))
			assert.NoError(t, err)

			err = ProcessManifestWithScript(strings.NewReader(tc.in), script, out)
			assert.NoError(t, err)

			assert.Equal(t, tc.out, out.String())
		})
	}
}
