package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	ALIAS_IDENTIFIER = "ASIAS_IDENTIFIER"
	PATH_PATTERN     = "PATH_PATTERN"
	MATCH_EXPR       = "MATCH_EXPR"
	ACTION_TYPE      = "ACTION_TYPE"
	ACTION_EXPR      = "ACTION_EXPR"
)

func getParser() Parser {
	identifier := Regex(`[a-zA-Z][a-zA-Z_0-9]*`).WithCategory(ALIAS_IDENTIFIER)
	optionalAlias := Optional(
		Seq(
			WhiteSpace,
			Literal("as"),
			WhiteSpace,
			identifier,
		),
	)

	pattern := QuotedString().WithCategory(PATH_PATTERN)

	conditionExprRef := Ref()

	matchExpr := Seq(
		pattern,
		optionalAlias,
	).WithCategory(MATCH_EXPR)

	matchExprRecur := Seq(
		matchExpr,
		WhiteSpace,
		Literal("and"),
		WhiteSpace,
		conditionExprRef,
	).WithCategory(MATCH_EXPR)

	conditionExpr := FirstOf(
		matchExprRecur,
		matchExpr,
	)

	conditionExprRef.Set(conditionExpr)

	literalKeep := Literal("keep").WithCategory(ACTION_TYPE)
	literalMove := Literal("move").WithCategory(ACTION_TYPE)

	actionSelector := OneOf(
		literalKeep,
		literalMove,
	)

	optionalActionAlias := Optional(identifier)

	actionExpr := Seq(
		actionSelector,
		WhiteSpace,
		optionalActionAlias,
	).WithCategory(ACTION_EXPR)

	instructionTokenizer := Seq(
		Optional(Seq(Literal("if"), WhiteSpace)),
		conditionExpr,
		WhiteSpace,
		Literal("then"),
		WhiteSpace,
		actionExpr,
	)

	return Parser{instructionTokenizer}
}

func TestParse(t *testing.T) {
	p := getParser()
	in := `if "foo" and "bar" as x then keep x`
	root, err := p.ParseString(in)
	assert.NotNil(t, root)
	errString := ""
	if err != nil {
		errString = "'" + err.Cursor().AtPos() + "'"
	}
	assert.Nil(t, err, errString)
	print(root)
	// assert.Equal(t, len(in.Input), cursor.Position)
}
