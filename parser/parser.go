// General purpose parser. Consider moving it to a separate package.
package parser

import (
	"fmt"
	coll "greasytoad/collections"
	"regexp"
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

var _ Tokenizer = (*Seq)(nil)

type Optional struct {
	Tokenizer Tokenizer
}

func (t Optional) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	if nextCur, ast, err := t.Tokenizer.Tokenize(cur); err == nil {
		return nextCur, ast, nil
	} else {
		return cur, NilAstNode, nil
	}
}

func (t Optional) String() string {
	return t.String() + "?"
}

var _ Tokenizer = (*Optional)(nil)

var NilAstNode = &nilAstNode{}

type nilAstNode struct{}

var _ AstNode = (*nilAstNode)(nil)

type AstNode any

type Regex struct {
	Matcher   *regexp.Regexp
	Name      string
	Evaluator Evaluator
}

func (t Regex) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	in := cur.AtPos()
	loc := t.Matcher.FindStringIndex(in)
	if loc == nil || loc[0] != 0 {
		return cur, nil, NewErrorWithCursor(cur, "expected regex %v", t.Matcher)
	}
	ast, err := t.Evaluator.Evaluate(in[loc[0]:loc[1]])
	if err != nil {
		return cur, nil, NewErrorWithCursor(cur, err.Error())
	}
	return cur.Advance(loc[1] - loc[0]), ast, nil
}

func (t Regex) String() string {
	return t.Name
}

var _ Tokenizer = (*Regex)(nil)

// Ref is used for self-referencing, recurrent expressions.
type Ref struct {
	Tokenizer Tokenizer
}

func (t Ref) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	return t.Tokenizer.Tokenize(cur)
}

func (t Ref) String() string {
	return "..."
}

var _ Tokenizer = (*Ref)(nil)

func QuotedString(name string, evaluator Evaluator) Tokenizer {
	m := regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)
	return Regex{
		Name:      name,
		Evaluator: evaluator,
		Matcher:   m,
	}
}

func WhiteSpace(name string, evaluator Evaluator) Tokenizer {
	m := regexp.MustCompile(`[ \n\t]+`)
	return Regex{
		Name:      name,
		Evaluator: evaluator,
		Matcher:   m,
	}
}
