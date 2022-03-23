package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getParser() Parser {
	identifier := Regex(`[a-zA-Z][a-zA-Z_0-9]*`)
	literalAs := Literal("as")
	optionalAlias := Optional{
		Seq{
			Tokenizers: []Tokenizer{
				WhiteSpace,
				literalAs,
				WhiteSpace,
				identifier,
			},
			Evaluator: NilMultiEvaluator,
		},
	}

	pattern := QuotedString()

	matchExpr := Seq{
		Tokenizers: []Tokenizer{
			pattern,
			optionalAlias,
		},
		Evaluator: NilMultiEvaluator,
	}

	literalAnd := Literal("and")

	conditionExprRef := &Ref{}

	conditionExpr := FirstOf{
		Tokenizers: []Tokenizer{
			Seq{
				Tokenizers: []Tokenizer{
					matchExpr,
					WhiteSpace,
					literalAnd,
					WhiteSpace,
					conditionExprRef,
				},
				Evaluator: NilMultiEvaluator,
			},
			matchExpr,
		},
	}

	conditionExprRef.Set(conditionExpr)

	literalIf := Literal("if")

	optionalStartingIf := Optional{
		Seq{
			Tokenizers: []Tokenizer{
				literalIf, WhiteSpace,
			},
			Evaluator: NilMultiEvaluator,
		},
	}

	literalThen := Literal("then")
	literalKeep := Literal("keep")
	literalMove := Literal("move")

	actionSelector := OneOf(literalKeep, literalMove)

	optionalActionAlias := Optional{identifier}

	actionExpr := Seq{
		Tokenizers: []Tokenizer{
			actionSelector,
			WhiteSpace,
			optionalActionAlias,
		},
		Evaluator: NilMultiEvaluator,
	}

	instructionTokenizer := Seq{
		Tokenizers: []Tokenizer{
			optionalStartingIf,
			conditionExpr,
			WhiteSpace,
			literalThen,
			WhiteSpace,
			actionExpr,
		},
		Evaluator: NilMultiEvaluator,
	}

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
