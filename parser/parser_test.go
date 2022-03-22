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
			QuotedString("part of path", NilEvaluator),
			// 	Optional(Sequence(WhiteSpace, Literal("as"), WhiteSpace, identifier)))
		},
		Evaluator: NilMultiEvaluator,
	}

	conditionalExpr := OneOf{
		Tokenizers: []Tokenizer{
			matchExpr,
			// 	Sequence(matchExpr, WhiteSpace, Literal("and"), WhiteSpace, refExpression),
		},
	}

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
			conditionalExpr,
			// 	Optional(Sequence(Literal("if"), WhiteSpace)),
			// 	conditionExpr,
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
	in := `if "foo" and "bar"`
	root, err := p.ParseString(in)
	assert.NotNil(t, root)
	assert.Nil(t, err)
	// assert.Equal(t, len(in.Input), cursor.Position)
}
