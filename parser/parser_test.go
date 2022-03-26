package parser

import (
	"fmt"
	"greasytoad/collections"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	ALIAS_IDENTIFIER = "ALIAS_IDENTIFIER"
	PATH_PATTERN     = "PATH_PATTERN"
	ACTION_TYPE      = "ACTION_TYPE"
)

func Identity(value any) (AstNode, error) {
	return value, nil
}

func NotNil(values []any) (AstNode, error) {
	nodes := collections.FilterSlice(values, func(t any) bool {
		return t != NilAstNode
	})
	if len(nodes) != 1 {
		panic(fmt.Sprintf("expected exactly one non-nil node, got %d: %v", len(nodes), nodes))
	}
	return nodes[0], nil
}

func getParser() Parser {
	identifier := Regex(`[a-zA-Z][a-zA-Z_0-9]*`).WithLabel(ALIAS_IDENTIFIER).WithEvaluator(Identity)

	optionalAlias := Optional(
		Seq(
			WhiteSpace,
			Literal("as"),
			WhiteSpace,
			identifier,
		).WithEvaluator(NotNil),
	)

	pattern := QuotedString().WithLabel(PATH_PATTERN).WithEvaluator(Identity)

	conditionExprRef := Ref()

	matchExpr := Seq(
		pattern,
		optionalAlias,
	)

	matchEvaluator := func(args []any) (AstNode, error) {
		pattern := args[0].(string)
		alias := ""
		if args[1] != NilAstNode {
			alias = args[1].(string)
		}
		_ = alias
		m := make(map[string]string)
		m[pattern] = alias
		return matchNode{m}, nil
	}

	matchExpr.WithEvaluator(matchEvaluator)

	matchExprRecur := Seq(
		matchExpr,
		WhiteSpace,
		Literal("and"),
		WhiteSpace,
		conditionExprRef,
	)

	matchRecurEvaluator := func(args []any) (AstNode, error) {
		m1 := args[0].(matchNode)
		m2 := args[4].(matchNode)
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

	optionalActionAlias := Optional(
		Seq(
			WhiteSpace,
			identifier,
		).WithEvaluator(NotNil))

	actionExpr := Seq(
		actionSelector,
		optionalActionAlias,
	)

	actionEvaluator := func(args []any) (AstNode, error) {
		action := args[0].(string)
		alias := ""
		if args[1] != NilAstNode {
			alias = args[1].(string)
		}
		return actionNode{action: action, alias: alias}, nil
	}
	actionExpr.WithEvaluator(actionEvaluator)

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

type actionNode struct {
	action string
	alias  string
}

type matchNode struct {
	patternToAlias map[string]string
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
			match: matchNode{
				map[string]string{`fo"o`: "", "bar": "x"},
			},
			action: actionNode{action: "keep", alias: "x"},
		},
		root,
	)
}

func TestParsers(t *testing.T) {
	tcs := []struct {
		in string
		ok bool
	}{
		// {
		// 	in: `if "fo\"o" as y and "bar" as x then keep x and move y`,
		// 	ok: true,
		// },
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
			in: `"foo" as x then move y`,
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
	}
	for _, tc := range tcs {
		p := getParser()
		ast, err := p.ParseString(tc.in)
		if tc.ok {
			assert.Nil(t, err, formatError(err))
			assert.NotNil(t, ast, formatError(err))
		} else {
			assert.NotNil(t, err, formatError(err))
		}
	}
}

func TestZeroOrMore(t *testing.T) {
	x := Literal("x").WithEvaluator(Identity)
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

func formatError(err ErrorWithCursor) string {
	errString := ""
	if err != nil {
		errString = "'" + err.Cursor().AtPos() + "'"
	}
	return errString
}

func IdentitySlice(values []any) (AstNode, error) {
	return values, nil
}

func Flatten(values []any) (AstNode, error) {
	flat := []any{}
	for _, value := range values {
		if arr, ok := value.([]any); ok {
			for _, a := range arr {
				flat = append(flat, a)
			}
		} else {
			flat = append(flat, value)
		}
	}
	return flat, nil
}
