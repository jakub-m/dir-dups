package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getParser() Expression {

	// if "foo" and "bar" as x then keep x

	identifier := RegexExpression(`[a-zA-Z][a-zA-Z_0-9]+`)

	matchExpr := Sequence(
		QuotedString,
		Optional(Sequence(WhiteSpace, Literal("as"), WhiteSpace, identifier)))

	var conditionExpr Expression
	conditionExpr = Or(
		matchExpr,
		Sequence(matchExpr, WhiteSpace, Literal("and"), WhiteSpace, conditionExpr),
	)

	_ = Or(Literal("move"), Literal("keep"))

	// line := Sequence(
	// Optional(Literal("if")),
	// conditionExpr,
	// Literal("then"),
	// actionExpr,
	// )

	return Sequence(
		Optional(Sequence(Literal("if"), WhiteSpace)),
		conditionExpr,
		//WhiteSpace,
		//Literal("then"),
	)
}

func TestParse(t *testing.T) {
	p := getParser()
	in := Cursor{
		Input:    `if "foo" and "bar" as x then keep x`,
		Position: 0,
	}
	root, cursor, err := p.Parse(in)
	assert.NotNil(t, root)
	assert.Nil(t, err, fmt.Sprintf("|%s|", formatParseError(err)))
	_ = cursor
	// assert.Equal(t, len(in.Input), cursor.Position)
}

func formatParseError(err *ParseError) string {
	if err == nil {
		return ""
	}
	return err.Cursor.inputAtPosition()
}
