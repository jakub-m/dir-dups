package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func getParser() Parser {

	// if "foo" and "bar" as x then keep x

	// identifier := RegexExpression(`[a-zA-Z][a-zA-Z_0-9]+`)

	// path := Tokenizer{
	// 	Lexer: QuotedString,
	// 	Name: "part of path"
	// 	Evaluator: func(lexeme string) {return PathToken{lexem}}
	// }{}

	// refExpression := &RefExpression{}
	// refExpression.Set(conditionExpr)

	// _ = Or(Literal("move"), Literal("keep"))

	// // line := Sequence(
	// // Optional(Literal("if")),
	// // conditionExpr,
	// // Literal("then"),
	// // actionExpr,
	// // )

	matchExpr := Seq{
		Tokenizers: []Tokenizer{
			QuotedString("part_of_path", NilEvaluator),
			// 	Optional(Sequence(WhiteSpace, Literal("as"), WhiteSpace, identifier)))
		},
		Evaluator: NilMultiEvaluator,
	}

	literalAnd := Literal{
		Value:     "and",
		Name:      "and",
		Evaluator: NilEvaluator,
	}

	_ = literalAnd

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

	literalIf := Literal{
		Value:     "if",
		Name:      "if",
		Evaluator: NilEvaluator,
	}

	optionalStartingIf := Optional{
		Seq{
			Tokenizers: []Tokenizer{
				literalIf, WhiteSpace,
			},
			Evaluator: NilMultiEvaluator,
		},
	}

	instructionTokenizer := Seq{
		Tokenizers: []Tokenizer{
			optionalStartingIf,
			conditionExpr,
			// 	//WhiteSpace,
			// 	//Literal("then"),
		},
		Evaluator: NilMultiEvaluator,
	}

	return Parser{instructionTokenizer}
}

func TestParse(t *testing.T) {
	p := getParser()
	//in := `if "foo" and "bar" as x then keep x`
	in := `if "foo" and "bar" and "baz"`
	root, err := p.ParseString(in)
	assert.NotNil(t, root)
	errString := ""
	if err != nil {
		errString = "'" + err.Cursor().AtPos() + "'"
	}
	assert.Nil(t, err, errString)
	// assert.Equal(t, len(in.Input), cursor.Position)
}
