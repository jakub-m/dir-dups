// General purpose parser. Consider moving it to a separate package.
package parser

import (
	"fmt"
	coll "greasytoad/collections"
	"strings"
)

// Parser takes a text and retuns a meaningful abstract syntaxt tree
type Parser struct {
	Tokenizer Tokenizer
}

func (p Parser) ParseString(s string) (AstNode, ErrorWithCursor) {
	startCur := Cursor{s, 0}
	cur, node, err := p.Tokenizer.Tokenize(startCur)
	if !cur.IsEnd() {
		return nil, NewErrorWithCursor(cur, "did not parse whole input")
	}
	return node, err
}

type Tokenizer interface {
	Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor)
	String() string
}

type AstNode any

type Cursor struct {
	Input    string
	Position int
}

func (c Cursor) IsEnd() bool {
	return len(c.Input) == c.Position
}

func (c Cursor) AtPos() string {
	return c.Input[c.Position:]
}

func (c Cursor) Advance(n int) Cursor {
	c.Position += n
	if c.Position > len(c.Input) {
		c.Position = len(c.Input)
	}
	return c
}

type ErrorWithCursor interface {
	Cursor() Cursor
	Error() string
}

var _ error = (ErrorWithCursor)(nil)

func NewErrorWithCursor(cur Cursor, format string, args ...any) ErrorWithCursor {
	return errorWithCursor{cur, fmt.Sprintf(format, args...)}
}

type errorWithCursor struct {
	cur     Cursor
	message string
}

func (e errorWithCursor) Cursor() Cursor {
	return e.cur
}

func (e errorWithCursor) Error() string {
	return e.message
}

type Evaluator interface {
	Evaluate(lexeme string) (AstNode, error)
}

type MultiEvaluator interface {
	Evaluate(lexeme []string) (AstNode, error)
}

// Below are the concrete tokenizers

type Literal struct {
	// Value is the exact value to match
	Value     string
	Evaluator Evaluator
	// Name is a displayable, context specific name
	Name string
}

func (t Literal) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	inputAtPosition := cur.AtPos()
	if strings.HasPrefix(inputAtPosition, t.Value) {
		n := len(t.Value)
		lexeme := inputAtPosition[:n]
		ast, err := t.Evaluator.Evaluate(lexeme)
		if err != nil {
			return cur, nil, NewErrorWithCursor(cur, err.Error())
		}
		return cur.Advance(n), ast, nil
	}
	return cur, nil, NewErrorWithCursor(cur, "expected \"%s\"", t.String())
}

func (t Literal) String() string {
	return t.Name
}

var _ Tokenizer = (*Literal)(nil)

type OneOf struct {
	Tokenizers []Tokenizer
}

func (t OneOf) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	type result struct {
		tok    Tokenizer
		ast    AstNode
		cursor Cursor
	}
	results := []result{}
	for _, tok := range t.Tokenizers {
		nextCur, ast, err := tok.Tokenize(cur)
		if err != nil {
			continue
		}
		results = append(results, result{
			tok:    tok,
			ast:    ast,
			cursor: nextCur,
		})
	}
	if len(results) == 0 {
		parserStrings := coll.TransformSlice(t.Tokenizers, func(tok Tokenizer) string { return tok.String() })
		return cur, nil, NewErrorWithCursor(cur, "failed to parse any of: %s", strings.Join(parserStrings, ", "))
	}
	if len(results) > 1 {
		resultStrings := coll.TransformSlice(results, func(r result) string { return r.tok.String() })
		return cur, nil, NewErrorWithCursor(cur, "more than one match: %s", strings.Join(resultStrings, ", "))
	}
	return results[0].cursor, results[0].ast, nil
}

func (t OneOf) String() string {
	ts := coll.TransformSlice(t.Tokenizers, func(t Tokenizer) string { return t.String() })
	return strings.Join(ts, " or ")
}

var _ Tokenizer = (*OneOf)(nil)

type Seq struct {
	Tokenizers []Tokenizer
	Evaluator  func([]AstNode) (AstNode, error)
}

func (t Seq) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	nodes := []AstNode{}
	for _, tok := range t.Tokenizers {
		nextCur, ast, err := tok.Tokenize(cur)
		if err != nil {
			return cur, nil, err
		}
		nodes = append(nodes, ast)
		cur = nextCur
	}
	ast, err := t.Evaluator(nodes)
	if err != nil {
		return cur, nil, NewErrorWithCursor(cur, err.Error())
	}
	return cur, ast, nil
}

func (t Seq) String() string {
	ts := coll.TransformSlice(t.Tokenizers, func(t Tokenizer) string { return t.String() })
	return strings.Join(ts, " ")
}

// func Optional(expr Expression) Expression {
// 	return optionalExpression{expr}
// }

// type optionalExpression struct {
// 	expr Expression
// }

// func (e optionalExpression) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
// 	if ast, newCur, err := e.expr.Parse(cursor); err == nil {
// 		return ast, newCur, err
// 	} else {
// 		return NilAstNode, cursor, nil
// 	}
// }

// func (e optionalExpression) String() string {
// 	return fmt.Sprintf("(%s)?", e.expr.String())
// }

// var NilAstNode = &EmptyAstNode{}

// type EmptyAstNode struct {
// }

// func Literal(value string) Expression {
// 	return literalExpression{value: value}
// }

// type literalExpression struct {
// 	value string
// }

// type LiteralAstNode struct {
// 	Value string
// }

// func (e literalExpression) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
// 	inputAtPosition := cursor.inputAtPosition()
// 	if strings.HasPrefix(inputAtPosition, e.value) {
// 		ast := LiteralAstNode{e.value}
// 		cur := cursor.movePos(len(e.value))
// 		return &ast, cur, nil
// 	}
// 	return nil, cursor, NewParseError(cursor, "expected \"%s\"", e.value)
// }

// func (e literalExpression) String() string {
// 	return e.value
// }

// func Or(expressions ...Expression) Expression {
// 	return oneOfExpr{expressions}
// }

// type oneOfExpr struct {
// 	expressions []Expression
// }

// func (e oneOfExpr) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
// }

// func (e oneOfExpr) String() string {
// 	es := coll.TransformSlice(e.expressions, func(e Expression) string { return e.String() })
// 	return strings.Join(es, "|")
// }

// var QuotedString = RegexExpression(`"(?:[^"\\]|\\.)*"`)

// var WhiteSpace = RegexExpression(`[ \n\t]+`)

// func RegexExpression(pattern string) Expression {
// 	return regexExpression{regexp.MustCompile(pattern)}
// }

// type regexExpression struct {
// 	re *regexp.Regexp
// }

// func (e regexExpression) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
// 	in := cursor.inputAtPosition()
// 	loc := e.re.FindStringIndex(in)
// 	if loc == nil || loc[0] != 0 {
// 		return nil, cursor, NewParseError(cursor, "expected regex %v", e.re)
// 	}
// 	return &RegexAstNode{in[loc[0]:loc[1]]}, cursor.movePos(loc[1] - loc[0]), nil
// }

// func (e regexExpression) String() string {
// 	return e.re.String()
// }

// type RefExpression struct {
// 	ref Expression
// }

// func (e RefExpression) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
// 	return e.ref.Parse(cursor)
// }

// func (e RefExpression) String() string {
// 	return fmt.Sprint("...")
// }

// func (e *RefExpression) Set(expr Expression) {
// 	e.ref = expr
// }

// type RegexAstNode struct {
// 	match string
// }
