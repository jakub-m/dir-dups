package parser

func getParser() Parser {
	identifier := Regex(`[a-zA-Z][a-zA-Z_0-9]*`).Keep()

	optionalAlias := Optional(
		Seq(
			WhiteSpace,
			Literal("as"),
			WhiteSpace,
			identifier,
		).NonNil(),
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

	matchExpr := Seq(
		QuotedString().Keep(),
		optionalAlias,
	).WithEvaluator(matchEvaluator)

	matchExprRecurRef := Ref()

	matchRecurEvaluator := func(args []any) (AstNode, error) {
		m1 := args[0].(matchNode)
		m2 := args[4].(matchNode)
		return matchNode{mergeMaps(m1.patternToAlias, m2.patternToAlias)}, nil
	}

	matchExprRecur := Seq(
		matchExpr,
		WhiteSpace,
		Literal("and"),
		WhiteSpace,
		matchExprRecurRef,
	).WithEvaluator(matchRecurEvaluator)

	conditionExpr := FirstOf(
		matchExprRecur,
		matchExpr,
	)

	matchExprRecurRef.Set(conditionExpr)

	actionSelector := OneOf(
		Literal("keep").Keep(),
		Literal("move").Keep(),
	)

	optionalActionAlias := Optional(
		Seq(
			WhiteSpace,
			identifier,
		).NonNil())

	actionEvaluator := func(args []any) (AstNode, error) {
		action := args[0].(string)
		alias := ""
		if args[1] != NilAstNode {
			alias = args[1].(string)
		}
		return actionNode{action: action, alias: alias}, nil
	}

	actionExpr := Seq(
		actionSelector,
		optionalActionAlias,
	).WithEvaluator(actionEvaluator)

	actions := Seq(
		actionExpr,
		ZeroOrMore(Seq(Literal("and"), actionExpr).NonNil())).FlattenNonNil()

	instructionTokenizer := Seq(
		Optional(Seq(Literal("if"), WhiteSpace)),
		conditionExpr,
		WhiteSpace,
		Literal("then"),
		WhiteSpace,
		actions,
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
