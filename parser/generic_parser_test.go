package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeqFlattenNotNil(t *testing.T) {
	x := Literal("x").Keep()
	y := Literal("y")
	xx := Seq(x, x)
	tok := Seq(xx, y, xx, y).FlattenNonNil()
	ast, err := Parser{tok}.ParseString("xxyxxy")
	assert.NoError(t, err)
	assert.Equal(t, []any{"x", "x", "x", "x"}, ast)
}

func TestZeroOrMore(t *testing.T) {
	x := Literal("x").Keep()
	tok := Seq(
		x,
		ZeroOrMore(
			Seq(WhiteSpace, x).WithEvaluator(NotNil),
		).WithEvaluator(IdentitySlice),
	).WithEvaluator(Flatten)
	p := Parser{tok}
	ast, err := p.ParseString("x x x x x")
	assert.Nil(t, err, formatError(err))
	assert.Equal(t, []any{"x", "x", "x", "x", "x"}, ast)
}

func TestLiteralKeep(t *testing.T) {
	x := Literal("x").Keep()
	y := Literal("y")

	node, err := Parser{x}.ParseString("x")
	assert.NoError(t, err)
	assert.Equal(t, "x", node)

	node, err = Parser{y}.ParseString("y")
	assert.NoError(t, err)
	assert.Equal(t, NilAstNode, node)
}

func formatError(err ErrorWithCursor) string {
	errString := ""
	if err != nil {
		errString = "'" + err.Cursor().AtPos() + "'"
	}
	return errString
}
