package parser

const (
	ALIAS_IDENTIFIER = "ALIAS_IDENTIFIER"
	PATH_PATTERN     = "PATH_PATTERN"
	ACTION_TYPE      = "ACTION_TYPE"
)

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

	// actions := Seq(
	// 	actionExpr,
	// 	ZeroOrMore(Seq(Literal("and"), actionExpr).WithEvaluator(NotNil)))

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

type actionNode struct {
	action string
	alias  string
}

type matchNode struct {
	patternToAlias map[string]string
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
