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

type Evaluator = func(AstNode) (AstNode, error)
type MultiEvaluator = func([]AstNode) (AstNode, error)

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
	category  string
}

func (t LiteralTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	inputAtPosition := cur.AtPos()
	if strings.HasPrefix(inputAtPosition, t.value) {
		n := len(t.value)
		lexeme := inputAtPosition[:n]
		ast, err := t.evaluator(lexeme)
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

func (t *LiteralTokenizer) WithLabel(c string) *LiteralTokenizer {
	t.category = c
	return t
}

var _ Tokenizer = (*LiteralTokenizer)(nil)

func FirstOf(tt ...Tokenizer) *FirstOfTokenizer {
	return &FirstOfTokenizer{tt}
}

type FirstOfTokenizer struct {
	Tokenizers []Tokenizer
}

func (t FirstOfTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	for _, tok := range t.Tokenizers {
		nextCur, ast, err := tok.Tokenize(cur)
		if err == nil {
			return nextCur, ast, nil
		}
	}
	return cur, nil, NewErrorWithCursor(cur, "could not continue")
}

func (t FirstOfTokenizer) String() string {
	ts := coll.TransformSlice(t.Tokenizers, func(t Tokenizer) string { return t.String() })
	return strings.Join(ts, " or ")
}

var _ Tokenizer = (*FirstOfTokenizer)(nil)

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

func Seq(tt ...Tokenizer) *SeqTokenizer {
	return &SeqTokenizer{
		tokenizers: tt,
		evaluator:  NilMultiEvaluator,
	}
}

type SeqTokenizer struct {
	tokenizers []Tokenizer
	evaluator  MultiEvaluator
	category   string
}

func (t SeqTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	nodes := []AstNode{}
	for _, tok := range t.tokenizers {
		nextCur, ast, err := tok.Tokenize(cur)
		if err != nil {
			return cur, nil, err
		}
		nodes = append(nodes, ast)
		cur = nextCur
	}
	ast, err := t.evaluator(nodes)
	if err != nil {
		return cur, nil, NewErrorWithCursor(cur, err.Error())
	}
	return cur, ast, nil
}

func (t SeqTokenizer) String() string {
	ts := coll.TransformSlice(t.tokenizers, func(t Tokenizer) string { return t.String() })
	ts = coll.FilterSlice(ts, func(s string) bool { return s != "" })
	return strings.Join(ts, " ")
}

func (t *SeqTokenizer) WithLabel(category string) *SeqTokenizer {
	t.category = category
	return t
}

func (t *SeqTokenizer) WithEvaluator(ev MultiEvaluator) *SeqTokenizer {
	t.evaluator = ev
	return t
}

var _ Tokenizer = (*SeqTokenizer)(nil)

func Optional(t Tokenizer) *OptionalTokenizer {
	return &OptionalTokenizer{
		tokenizer: t,
	}
}

type OptionalTokenizer struct {
	tokenizer Tokenizer
}

func (t OptionalTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	if nextCur, ast, err := t.tokenizer.Tokenize(cur); err == nil {
		return nextCur, ast, nil
	} else {
		return cur, NilAstNode, nil
	}
}

func (t OptionalTokenizer) String() string {
	return t.String() + "?"
}

var _ Tokenizer = (*OptionalTokenizer)(nil)

var NilAstNode = &nilAstNode{}

type nilAstNode struct{}

var _ AstNode = (*nilAstNode)(nil)

type AstNode any

func Regex(pattern string) *RegexTokenizer {
	return &RegexTokenizer{
		matcher:   regexp.MustCompile(pattern),
		name:      pattern,
		evaluator: NilEvaluator,
	}
}

type RegexTokenizer struct {
	category  string
	matcher   *regexp.Regexp
	name      string
	evaluator Evaluator
}

func (t *RegexTokenizer) WithLabel(category string) *RegexTokenizer {
	t.category = category
	return t
}

func (t RegexTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	in := cur.AtPos()
	loc := t.matcher.FindStringIndex(in)
	if loc == nil || loc[0] != 0 {
		return cur, nil, NewErrorWithCursor(cur, "expected regex %v", t.matcher)
	}
	ast, err := t.evaluator(in[loc[0]:loc[1]])
	if err != nil {
		return cur, nil, NewErrorWithCursor(cur, err.Error())
	}
	return cur.Advance(loc[1] - loc[0]), ast, nil
}

func (t RegexTokenizer) String() string {
	return t.name
}

var _ Tokenizer = (*RegexTokenizer)(nil)

func Ref() *RefTokenizer {
	return &RefTokenizer{}
}

// RefTokenizer is used for self-referencing, recurrent expressions.
type RefTokenizer struct {
	tokenizer Tokenizer
}

func (t RefTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	return t.tokenizer.Tokenize(cur)
}

func (t RefTokenizer) String() string {
	return "..."
}

func (t *RefTokenizer) Set(tok Tokenizer) {
	if tok == nil {
		panic("must not pass nil to Set")
	}
	t.tokenizer = tok
}

var _ Tokenizer = (*RefTokenizer)(nil)

func QuotedString() *RegexTokenizer {
	return Regex(`"(?:[^"\\]|\\.)*"`)
}

var WhiteSpace = whiteSpace()

func whiteSpace() Tokenizer {
	m := regexp.MustCompile(`[ \t]+`)
	return RegexTokenizer{
		name:      "",
		evaluator: NilEvaluator,
		matcher:   m,
	}
}

var NilEvaluator = func(lexeme AstNode) (AstNode, error) {
	return NilAstNode, nil
}

var NilMultiEvaluator = func(nodes []AstNode) (AstNode, error) {
	return NilAstNode, nil
}

func OnlyWithType[T any](values ...any) T {
	results := []T{}
	for _, v := range values {
		if t, ok := v.(T); ok {
			results = append(results, t)
		}
	}
	if len(results) != 1 {
		panic(fmt.Sprintf("expected exactly 1 value of type %+v, got %d", (*T)(nil), len(results)))
	}
	return results[0]
}
