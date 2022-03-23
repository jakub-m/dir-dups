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

func Literal(value string) *LiteralTokenizer {
	return &LiteralTokenizer{
		value:     value,
		evaluator: NilEvaluator,
	}
}

func (l *LiteralTokenizer) WithEvaluator(ev Evaluator) *LiteralTokenizer {
	l.evaluator = ev
	return l
}

type LiteralTokenizer struct {
	// value is the exact value to match
	value     string
	evaluator Evaluator
}

func (t LiteralTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	inputAtPosition := cur.AtPos()
	if strings.HasPrefix(inputAtPosition, t.value) {
		n := len(t.value)
		lexeme := inputAtPosition[:n]
		ast, err := t.evaluator.Evaluate(lexeme)
		if err != nil {
			return cur, nil, NewErrorWithCursor(cur, err.Error())
		}
		return cur.Advance(n), ast, nil
	}
	return cur, nil, NewErrorWithCursor(cur, "expected \"%s\"", t.String())
}

func (t LiteralTokenizer) String() string {
	return t.value
}

var _ Tokenizer = (*LiteralTokenizer)(nil)

type FirstOf struct {
	Tokenizers []Tokenizer
	label      string
}

func (t FirstOf) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	for _, tok := range t.Tokenizers {
		nextCur, ast, err := tok.Tokenize(cur)
		if err == nil {
			return nextCur, ast, nil
		}
	}
	return cur, nil, NewErrorWithCursor(cur, "could not continue")
}

func (t FirstOf) String() string {
	ts := coll.TransformSlice(t.Tokenizers, func(t Tokenizer) string { return t.String() })
	return strings.Join(ts, " or ")
}

var _ Tokenizer = (*FirstOf)(nil)

func OneOf(tt ...Tokenizer) *OneOfTokenizer {
	return &OneOfTokenizer{tt}
}

type OneOfTokenizer struct {
	Tokenizers []Tokenizer
}

func (t OneOfTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
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

func (t OneOfTokenizer) String() string {
	ts := coll.TransformSlice(t.Tokenizers, func(t Tokenizer) string { return t.String() })
	return strings.Join(ts, " or ")
}

var _ Tokenizer = (*OneOfTokenizer)(nil)

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
	ts = coll.FilterSlice(ts, func(s string) bool { return s != "" })
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

func Regex(pattern string) *RegexTokenizer {
	return &RegexTokenizer{
		Matcher:   regexp.MustCompile(pattern),
		Name:      pattern,
		Evaluator: NilEvaluator,
	}
}

type RegexTokenizer struct {
	Matcher   *regexp.Regexp
	Name      string
	Evaluator Evaluator
}

func (t RegexTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
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

func (t RegexTokenizer) String() string {
	return t.Name
}

var _ Tokenizer = (*RegexTokenizer)(nil)

// Ref is used for self-referencing, recurrent expressions.
type Ref struct {
	Tokenizer Tokenizer
}

type foo struct {
	bar string
}

func (t Ref) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	return t.Tokenizer.Tokenize(cur)
}

func (t Ref) String() string {
	return "..."
}

func (t *Ref) Set(tok Tokenizer) {
	if tok == nil {
		panic("must not pass nil to Set")
	}
	t.Tokenizer = tok
}

var _ Tokenizer = (*Ref)(nil)

func QuotedString() *RegexTokenizer {
	return Regex(`"(?:[^"\\]|\\.)*"`)
}

var WhiteSpace = whiteSpace()

func whiteSpace() Tokenizer {
	m := regexp.MustCompile(`[ \t]+`)
	return RegexTokenizer{
		Name:      "",
		Evaluator: NilEvaluator,
		Matcher:   m,
	}
}

var NilEvaluator = nilEvaluator{}

type nilEvaluator struct {
}

func (e nilEvaluator) Evaluate(lexeme string) (AstNode, error) {
	return NilAstNode, nil
}

func NilMultiEvaluator(as []AstNode) (AstNode, error) {
	return NilAstNode, nil
}
