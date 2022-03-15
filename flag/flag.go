package flag

import (
	goflag "flag"
	"strings"
)

func NewStringList(dest *[]string) *StringList {
	return &StringList{Values: dest}
}

type StringList struct {
	Values *[]string
}

func (s StringList) Set(input string) error {
	*s.Values = append(*s.Values, input)
	return nil
}

func (s StringList) String() string {
	return "StringList"
}

var _ goflag.Value = (*StringList)(nil)

type CommaSepValue struct {
	Value *[]string
}

var _ goflag.Value = (*CommaSepValue)(nil)

func (s CommaSepValue) Set(input string) error {
	*s.Value = strings.Split(input, ",")
	return nil
}

func (s CommaSepValue) String() string {
	return "CommaSplitter"
}
