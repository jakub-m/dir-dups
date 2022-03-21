package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getParser() Expression {

	// if "foo" and "bar" as x then keep x

	identifier := RegexExpression(`[a-zA-Z][a-zA-Z_0-9]+`)

	matchExpr := Sequence(
		QuotedString,
		Optional(Sequence(Literal("as"), identifier)))

	var conditionExpr Expression
	conditionExpr = Or(
		matchExpr,
		Sequence(matchExpr, Literal("and"), conditionExpr))

	actionExpr := Or(Literal("move"), Literal("keep"))

	line := Sequence(
		Optional(Literal("if")),
		conditionExpr,
		Literal("then"),
		actionExpr,
	)

	return line
}

func TestParse(t *testing.T) {
	p := getParser()
	in := Cursor{
		Input:    `if "foo" and "bar" as x then keep x`,
		Position: 0,
	}
	root, cursor, err := p.Parse(in)
	assert.NotNil(t, root)
	assert.Equal(t, len(in.Input), cursor.Position)
	assert.Nil(t, err)
}
