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
		m := []matchWithAlias{{match: pattern, alias: alias}}
		return m, nil
	}

	matchExpr := Seq(
		QuotedString().Keep(),
		optionalAlias,
	).WithEvaluator(matchEvaluator)

	matchExprRecurRef := Ref()

	matchRecurEvaluator := func(args []any) (AstNode, error) {
		m1 := args[0].([]matchWithAlias)
		m2 := args[4].([]matchWithAlias)
		mm := append(m1, m2...)
		return mm, nil
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
		return actionForAlias{action: action, alias: alias}, nil
	}

	actionExpr := Seq(
		actionSelector,
		optionalActionAlias,
	).WithEvaluator(actionEvaluator)

	actionsEvaluator := func(args []any) (AstNode, error) {
		nodes := []actionForAlias{}
		for _, arg := range args {
			if arg != NilAstNode {
				nodes = append(nodes, arg.(actionForAlias))
			}
		}
		return nodes, nil
	}

	actions := Seq(
		actionExpr,
		ZeroOrMore(
			Seq(WhiteSpace, Literal("and"), WhiteSpace, actionExpr).NonNil()),
	).WithEvaluator(actionsEvaluator)

	instructionEvaluator := func(args []any) (AstNode, error) {
		print(args)
		return instructionNode{
			matches: OneWithType[[]matchWithAlias](args),
			actions: OneWithType[[]actionForAlias](args),
		}, nil
	}
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

type instructionNode struct {
	matches []matchWithAlias
	actions []actionForAlias
}

type actionForAlias struct {
	action string
	alias  string
}

type matchWithAlias struct {
	match string
	alias string
}
