package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getParser() Parser {
	identifier := Regex(`[a-zA-Z][a-zA-Z_0-9]*`)
	literalAs := Literal("as")
	optionalAlias := Optional{
		Seq(
			WhiteSpace,
			literalAs,
			WhiteSpace,
			identifier,
		),
	}

	pattern := QuotedString()

	matchExpr := Seq(
		pattern,
		optionalAlias,
	)

	literalAnd := Literal("and")

	conditionExprRef := &Ref{}

	conditionExpr := FirstOf{
		Tokenizers: []Tokenizer{
			Seq(
				matchExpr,
				WhiteSpace,
				literalAnd,
				WhiteSpace,
				conditionExprRef,
			),
			matchExpr,
		},
	}

	conditionExprRef.Set(conditionExpr)

	literalIf := Literal("if")

	optionalStartingIf := Optional{
		Seq(literalIf, WhiteSpace),
	}

	literalThen := Literal("then")
	literalKeep := Literal("keep")
	literalMove := Literal("move")

	actionSelector := OneOf(literalKeep, literalMove)

	optionalActionAlias := Optional{identifier}

	actionExpr := Seq(
		actionSelector,
		WhiteSpace,
		optionalActionAlias,
	)

	instructionTokenizer := Seq(
		optionalStartingIf,
		conditionExpr,
		WhiteSpace,
		literalThen,
		WhiteSpace,
		actionExpr,
	)

	return Parser{instructionTokenizer}
}

func TestParse(t *testing.T) {
	p := getParser()
	//in := `if "foo" and "bar" as x then keep x`
	in := `if "foo" and "bar" as x then keep x`
	root, err := p.ParseString(in)
	assert.NotNil(t, root)
	errString := ""
	if err != nil {
		errString = "'" + err.Cursor().AtPos() + "'"
	}
	assert.Nil(t, err, errString)
	// assert.Equal(t, len(in.Input), cursor.Position)
}
