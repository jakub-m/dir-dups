package parser

import (
	"testing"

	par "greasytoad/parser"

	"github.com/stretchr/testify/assert"
)

func TestConcreteParser(t *testing.T) {
	p := getParser()
	in := `if "fo\"o" and "bar" as x then keep x`
	root, err := p.ParseString(in)
	assert.NotNil(t, root)
	errString := formatError(err)
	assert.Nil(t, err, errString)
	// assert.Equal(t, len(in.Input), cursor.Position)

	assert.Equal(t,
		instructionNode{
			matches: []matchWithAlias{{match: `fo"o`, alias: ""}, {match: "bar", alias: "x"}},
			actions: []actionForAlias{{action: "keep", alias: "x"}},
		},
		root,
	)
}

func TestParsers(t *testing.T) {
	tcs := []struct {
		in string
		ok bool
	}{
		{
			in: `if "fo\"o" and "bar" as x then keep x`,
			ok: true,
		},
		{
			in: `"foo" then move`,
			ok: true,
		},
		{
			in: `"foo then move`,
			ok: false,
		},
		{
			in: `"foo" as y then move y`,
			ok: true,
		},
		{
			in: `"foo" and "bar" and "quux" then move`,
			ok: true,
		},
		{
			in: `"foo" as x then mov y`,
			ok: false,
		},
		{
			in: `foo then move`,
			ok: false,
		},
		{
			in: `if "foo" as x and "bar" as y then keep x and move y`,
			ok: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.in, func(t *testing.T) {
			p := getParser()
			ast, err := p.ParseString(tc.in)
			if tc.ok {
				assert.Nil(t, err, formatError(err))
				assert.NotNil(t, ast, formatError(err))
			} else {
				assert.NotNil(t, err, formatError(err))
			}
		})
	}
}

func formatError(err par.ErrorWithCursor) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
