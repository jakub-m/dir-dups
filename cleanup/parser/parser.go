package parser

import (
	"fmt"
	par "greasytoad/parser"
)

func GetMinilangParser() par.Parser {
	identifier := par.Regex(`[a-zA-Z][a-zA-Z_0-9]*`).Keep()

	optionalAlias := par.Optional(
		par.Seq(
			par.WhiteSpace,
			par.Literal("as"),
			par.WhiteSpace,
			identifier,
		).NonNil(),
	)

	matchEvaluator := func(args []any) (par.AstNode, error) {
		pattern := ""
		switch p := args[0].(type) {
		case string:
			pattern = p
		case par.QuotedStringAstNode:
			pattern = string(p)
		default:
			panic(fmt.Sprintf("unexpected type for value: %+v", p))
		}
		alias := ""
		if args[1] != par.NilAstNode {
			alias = args[1].(string)
		}
		_ = alias
		m := []MatchWithAlias{{Match: pattern, Alias: alias}} // TODO add handling "other"
		return m, nil
	}

	matchExpr := par.Seq(
		par.FirstOf(
			par.QuotedString().Keep(),
			par.Literal("other").Keep(),
		),
		optionalAlias,
	).WithEvaluator(matchEvaluator)

	matchExprRecurRef := par.Ref()

	matchRecurEvaluator := func(args []any) (par.AstNode, error) {
		m1 := args[0].([]MatchWithAlias)
		m2 := args[4].([]MatchWithAlias)
		mm := append(m1, m2...)
		return mm, nil
	}

	matchExprRecur := par.Seq(
		matchExpr,
		par.WhiteSpace,
		par.Literal("and"),
		par.WhiteSpace,
		matchExprRecurRef,
	).WithEvaluator(matchRecurEvaluator)

	conditionExpr := par.FirstOf(
		matchExprRecur,
		matchExpr,
	)

	matchExprRecurRef.Set(conditionExpr)

	actionSelector := par.OneOf(
		par.Literal("keep").Keep(),
		par.Literal("move").Keep(),
	)

	optionalActionAlias := par.Optional(
		par.Seq(
			par.WhiteSpace,
			identifier,
		).NonNil())

	actionEvaluator := func(args []any) (par.AstNode, error) {
		action := args[0].(string)
		alias := ""
		if args[1] != par.NilAstNode {
			alias = args[1].(string)
		}
		return ActionForAlias{Action: ManifestOperation(action), Alias: alias}, nil
	}

	actionExpr := par.Seq(
		actionSelector,
		optionalActionAlias,
	).WithEvaluator(actionEvaluator)

	actionsEvaluator := func(args []any) (par.AstNode, error) {
		nodes := []ActionForAlias{}
		for _, arg := range args {
			if arg != par.NilAstNode {
				nodes = append(nodes, arg.(ActionForAlias))
			}
		}
		return nodes, nil
	}

	actions := par.Seq(
		actionExpr,
		par.ZeroOrMore(
			par.Seq(par.WhiteSpace, par.Literal("and"), par.WhiteSpace, actionExpr).NonNil()),
	).WithEvaluator(actionsEvaluator)

	instructionEvaluator := func(args []any) (par.AstNode, error) {
		return InstructionNode{
			Matches: par.OneWithType[[]MatchWithAlias](args),
			Actions: par.OneWithType[[]ActionForAlias](args),
		}, nil
	}
	instructionTokenizer := par.Seq(
		par.Optional(par.Seq(par.Literal("if"), par.WhiteSpace)),
		conditionExpr,
		par.WhiteSpace,
		par.Literal("then"),
		par.WhiteSpace,
		actions,
	).WithEvaluator(instructionEvaluator)

	return par.Parser{instructionTokenizer}
}

type InstructionNode struct {
	Matches []MatchWithAlias
	Actions []ActionForAlias
}

type ActionForAlias struct {
	Action ManifestOperation
	Alias  string
}

type MatchWithAlias struct {
	Match string
	Alias string
}

type ManifestOperation string

const (
	Keep ManifestOperation = "keep"
	Move                   = "move"
)
