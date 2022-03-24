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

func Identity(value any) (AstNode, error) {
	return value, nil
}

func getParser() Parser {
	identifier := Regex(`[a-zA-Z][a-zA-Z_0-9]*`).WithLabel(ALIAS_IDENTIFIER)
	optionalAlias := Optional(
		Seq(
			WhiteSpace,
			Literal("as"),
			WhiteSpace,
			identifier,
		),
	)

	pattern := QuotedString().WithLabel(PATH_PATTERN).WithEvaluator(Identity)

	conditionExprRef := Ref()

	matchExpr := Seq(
		pattern,
		optionalAlias,
	).WithLabel(MATCH_EXPR)

	matchEvaluator := func(args []any) (AstNode, error) {
		pattern := args[0].(string)
		m := make(map[string]string)
		m[pattern] = "" // TODO handle alias here
		return matchNode{m}, nil
	}

	matchExpr.WithEvaluator(matchEvaluator)

	matchExprRecur := Seq(
		matchExpr,
		WhiteSpace,
		Literal("and"),
		WhiteSpace,
		conditionExprRef,
	).WithLabel(MATCH_EXPR)

	matchRecurEvaluator := func(args []any) (AstNode, error) {
		// TODO handle the other args here!
		m1 := args[0].(matchNode)
		m2 := args[4].(matchNode)
		// m.patternToAlias
		// TODO this is actuall wrong. we might want to have a single evaluator with all the patterns.
		// m := make(map[string]string)
		// m[pattern] = "" // TODO handle alias here
		return matchNode{mergeMaps(m1.patternToAlias, m2.patternToAlias)}, nil
	}

	matchExprRecur.WithEvaluator(matchRecurEvaluator)

	conditionExpr := FirstOf(
		matchExprRecur,
		matchExpr,
	)

	conditionExprRef.Set(conditionExpr)

	literalKeep := Literal("keep").WithLabel(ACTION_TYPE).WithEvaluator(Identity)
	literalMove := Literal("move").WithLabel(ACTION_TYPE).WithEvaluator(Identity)

	actionSelector := OneOf(
		literalKeep,
		literalMove,
	)

	optionalActionAlias := Optional(identifier)

	actionExpr := Seq(
		actionSelector,
		WhiteSpace,
		optionalActionAlias,
	).WithLabel(ACTION_EXPR).WithEvaluator(actionEvaluator)

	instructionTokenizer := Seq(
		Optional(Seq(Literal("if"), WhiteSpace)),
		conditionExpr,
		WhiteSpace,
		Literal("then"),
		WhiteSpace,
		actionExpr,
	).WithEvaluator(instructionEvaluator)

	return Parser{instructionTokenizer}
}

func actionEvaluator(args []any) (AstNode, error) {
	action := args[0].(string)
	return actionNode{action: action}, nil
}

type actionNode struct {
	action string
	// TODO add alias here
}

type matchNode struct {
	patternToAlias map[string]string
	// TODO add optional alias
}

func instructionEvaluator(args []any) (AstNode, error) {
	return instructionNode{
		match:  OneWithType[matchNode](args),
		action: OneWithType[actionNode](args),
	}, nil
}

type instructionNode struct {
	match  matchNode
	action actionNode
}

func mergeMaps[K comparable, V any](m1, m2 map[K]V) map[K]V {
	out := make(map[K]V)
	for k, v := range m1 {
		out[k] = v
	}
	for k, v := range m2 {
		out[k] = v
	}
	return out
}

func TestParse(t *testing.T) {
	p := getParser()
	in := `if "fo\"o" and "bar" as x then keep x`
	root, err := p.ParseString(in)
	assert.NotNil(t, root)
	errString := ""
	if err != nil {
		errString = "'" + err.Cursor().AtPos() + "'"
	}
	assert.Nil(t, err, errString)
	// assert.Equal(t, len(in.Input), cursor.Position)

	assert.Equal(t,
		instructionNode{
			match: matchNode{
				map[string]string{`fo"o`: "", "bar": ""},
				//map[string]string{`fo"o`: "", "bar": "x"},
			},
			action: actionNode{`keep`},
		},
		root,
	)
}
