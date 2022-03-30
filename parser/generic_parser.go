// General purpose parser. Consider moving it to a separate package.
package parser

import (
	"fmt"
	"greasytoad/collections"
	coll "greasytoad/collections"
	"regexp"
	"strconv"
	"strings"
)

// Ast is a node in abstract syntax tree.
type AstNode any

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

type Evaluator = func(any) (AstNode, error)
type MultiEvaluator = func([]any) (AstNode, error)

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

// Keep makes the tokenizer return the matched literal. By default, LiteralTokenizer returns NilAstNode.
func (t *LiteralTokenizer) Keep() *LiteralTokenizer {
	t.evaluator = Identity
	return t
}

type LiteralTokenizer struct {
	// value is the exact value to match
	value     string
	evaluator Evaluator
	label     string
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

// TODO do we need label for anything
func (t *LiteralTokenizer) WithLabel(c string) *LiteralTokenizer {
	t.label = c
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
		evaluator:  IdentitySlice,
	}
}

type SeqTokenizer struct {
	tokenizers []Tokenizer
	evaluator  MultiEvaluator
}

func (t SeqTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	nodes := []any{}
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

func (t *SeqTokenizer) WithEvaluator(ev MultiEvaluator) *SeqTokenizer {
	t.evaluator = ev
	return t
}

// FlattenNonNil makes the tokenizer filter-out NilAstToken values, and then flatten the embedded slices.
func (t *SeqTokenizer) FlattenNonNil() *SeqTokenizer {
	t.evaluator = flattenNonNilEvaluator
	return t
}

// NonNil makes the tokenizer filter-out NilAstToken values.
func (t *SeqTokenizer) NonNil() *SeqTokenizer {
	t.evaluator = NotNil
	return t
}

var _ Tokenizer = (*SeqTokenizer)(nil)

func flattenNonNilEvaluator(input []any) (AstNode, error) {
	flatten := []any{}
	for _, v := range input {
		if v != NilAstNode {
			if arr, ok := v.([]any); ok {
				for _, w := range arr {
					flatten = append(flatten, w)
				}
			} else {
				flatten = append(flatten, v)

			}
		}
	}
	return flatten, nil
}

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

func ZeroOrMore(t Tokenizer) *ZeroOrMoreTokenizer {
	return &ZeroOrMoreTokenizer{
		tokenizer: t,
		evaluator: NilMultiEvaluator,
	}
}

type ZeroOrMoreTokenizer struct {
	tokenizer Tokenizer
	evaluator MultiEvaluator
}

func (t ZeroOrMoreTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	results := []any{}
	for !cur.IsEnd() {
		nextCur, ast, err := t.tokenizer.Tokenize(cur)
		if err != nil {
			break
		}
		results = append(results, ast)
		cur = nextCur
	}
	ast, err := t.evaluator(results)
	if err != nil {
		return cur, ast, NewErrorWithCursor(cur, err.Error())
	}
	return cur, ast, nil
}

func (t ZeroOrMoreTokenizer) String() string {
	return fmt.Sprintf("(%s)*", t.tokenizer)
}

func (t *ZeroOrMoreTokenizer) WithEvaluator(e MultiEvaluator) *ZeroOrMoreTokenizer {
	t.evaluator = e
	return t
}

var _ Tokenizer = (*ZeroOrMoreTokenizer)(nil)

var NilAstNode = &nilAstNode{}

type nilAstNode struct{}

func Regex(pattern string) *RegexTokenizer {
	return &RegexTokenizer{
		matcher:   regexp.MustCompile(pattern),
		name:      pattern,
		evaluator: NilEvaluator,
	}
}

type RegexTokenizer struct {
	label     string
	matcher   *regexp.Regexp
	name      string
	evaluator Evaluator
}

func (t *RegexTokenizer) WithLabel(label string) *RegexTokenizer {
	t.label = label
	return t
}

func (t *RegexTokenizer) WithEvaluator(ev Evaluator) *RegexTokenizer {
	t.evaluator = ev
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

func QuotedString() *QuotedStringTokenizer {
	rt := Regex(`"(?:[^"\\]|\\.)*"`).WithEvaluator(func(s any) (AstNode, error) {
		return strconv.Unquote(s.(string))
	})
	return &QuotedStringTokenizer{rt: rt, ev: NilEvaluator}
}

type QuotedStringTokenizer struct {
	rt *RegexTokenizer
	ev Evaluator
}

func (t *QuotedStringTokenizer) Tokenize(cur Cursor) (Cursor, AstNode, ErrorWithCursor) {
	nextCur, ast, errWithCursor := t.rt.Tokenize(cur)
	if errWithCursor != nil {
		return cur, nil, errWithCursor
	}
	newAst, err := t.ev(ast)
	if err != nil {
		return cur, nil, NewErrorWithCursor(cur, err.Error())
	}
	return nextCur, newAst, nil

}

func (t *QuotedStringTokenizer) String() string {
	return t.String()
}

func (t *QuotedStringTokenizer) WithLabel(label string) *QuotedStringTokenizer {
	t.rt.WithLabel(label)
	return t
}

func (t *QuotedStringTokenizer) WithEvaluator(ev Evaluator) *QuotedStringTokenizer {
	t.ev = ev
	return t
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

var NilEvaluator = func(value any) (AstNode, error) {
	return NilAstNode, nil
}

var NilMultiEvaluator = func(values []any) (AstNode, error) {
	return NilAstNode, nil
}

func OneWithType[T any](values []any) T {
	results := AllWithType[T](values)
	if len(results) != 1 {
		panic(fmt.Sprintf("expected exactly 1 value of type %+v, got %d", (*T)(nil), len(results)))
	}
	return results[0]
}

func AllWithType[T any](values []any) []T {
	results := []T{}
	for _, v := range values {
		if t, ok := v.(T); ok {
			results = append(results, t)
		}
	}
	return results
}

func Identity(value any) (AstNode, error) {
	return value, nil
}

func IdentitySlice(values []any) (AstNode, error) {
	return values, nil
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
