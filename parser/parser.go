// General purpose parser. Consider moving it to a separate package.
package parser

import (
	"fmt"
	coll "greasytoad/collections"
	"regexp"
	"strings"
)

type Expression interface {
	Parse(Cursor) (AstNode, Cursor, *ParseError)
	String() string
}

type Cursor struct {
	Input    string
	Position int
}

func (c Cursor) inputAtPosition() string {
	return c.Input[c.Position:]
}

func (c Cursor) movePos(shift int) Cursor {
	c.Position += shift
	return c
}

type AstNode interface {
}

func Sequence(expressions ...Expression) Expression {
	return sequenceExpr{expressions}
}

type sequenceExpr struct {
	expressions []Expression
}

func (e sequenceExpr) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
	nodes := []AstNode{}
	for _, expr := range e.expressions {
		ast, nextCur, err := expr.Parse(cursor)
		if err != nil {
			return nil, cursor, err
		}
		nodes = append(nodes, ast)
		cursor = nextCur
	}
	return &SequenceAstNode{nodes}, cursor, nil
}

func (e sequenceExpr) String() string {
	es := coll.TransformSlice(e.expressions, func(e Expression) string { return e.String() })
	return strings.Join(es, " ")
}

type SequenceAstNode struct {
	Nodes []AstNode
}

func Optional(expr Expression) Expression {
	return optionalExpression{expr}
}

type optionalExpression struct {
	expr Expression
}

func (e optionalExpression) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
	if ast, newCur, err := e.expr.Parse(cursor); err == nil {
		return ast, newCur, err
	} else {
		return NilAstNode, cursor, nil
	}
}

func (e optionalExpression) String() string {
	return fmt.Sprintf("(%s)?", e.expr.String())
}

var NilAstNode = &EmptyAstNode{}

type EmptyAstNode struct {
}

func Literal(value string) Expression {
	return literalExpression{value: value}
}

type literalExpression struct {
	value string
}

type LiteralAstNode struct {
	Value string
}

func (e literalExpression) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
	inputAtPosition := cursor.inputAtPosition()
	if strings.HasPrefix(inputAtPosition, e.value) {
		ast := LiteralAstNode{e.value}
		cur := cursor.movePos(len(e.value))
		return &ast, cur, nil
	}
	return nil, cursor, NewParseError(cursor, "expected \"%s\"", e.value)
}

func (e literalExpression) String() string {
	return e.value
}

func Or(expressions ...Expression) Expression {
	return oneOfExpr{expressions}
}

type oneOfExpr struct {
	expressions []Expression
}

func (e oneOfExpr) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
	type result struct {
		expr   Expression
		ast    AstNode
		cursor Cursor
	}
	results := []result{}
	for _, expr := range e.expressions {
		ast, nextCur, err := expr.Parse(cursor)
		if err != nil {
			continue
		}
		results = append(results, result{
			expr:   expr,
			ast:    ast,
			cursor: nextCur,
		})
	}
	if len(results) == 0 {
		parserStrings := coll.TransformSlice(e.expressions, func(e Expression) string { return fmt.Sprint(e) })
		return NilAstNode, cursor, NewParseError(cursor, "failed to parse any of: %s", strings.Join(parserStrings, ", "))
	}
	if len(results) > 1 {
		resultStrings := coll.TransformSlice(results, func(e result) string { return fmt.Sprint(e) })
		return NilAstNode, cursor, NewParseError(cursor, "more than one match: %s", strings.Join(resultStrings, ", "))
	}
	return results[0].ast, results[0].cursor, nil
}

func (e oneOfExpr) String() string {
	es := coll.TransformSlice(e.expressions, func(e Expression) string { return e.String() })
	return strings.Join(es, "|")
}

var QuotedString = RegexExpression(`"(?:[^"\\]|\\.)*"`)

var WhiteSpace = RegexExpression(`[ \n\t]+`)

func RegexExpression(pattern string) Expression {
	return regexExpression{regexp.MustCompile(pattern)}
}

type regexExpression struct {
	re *regexp.Regexp
}

func (e regexExpression) Parse(cursor Cursor) (AstNode, Cursor, *ParseError) {
	in := cursor.inputAtPosition()
	loc := e.re.FindStringIndex(in)
	if loc == nil || loc[0] != 0 {
		return nil, cursor, NewParseError(cursor, "expected regex %v", e.re)
	}
	return &RegexAstNode{in[loc[0]:loc[1]]}, cursor.movePos(loc[1] - loc[0]), nil
}

func (e regexExpression) String() string {
	return e.re.String()
}

type RegexAstNode struct {
	match string
}

func NewParseError(cur Cursor, format string, args ...any) *ParseError {
	return &ParseError{
		Cursor:  cur,
		Message: fmt.Sprintf(format, args...),
	}
}

type ParseError struct {
	Cursor  Cursor
	Message string
}

func (e ParseError) Error() string {
	return e.Message
}
