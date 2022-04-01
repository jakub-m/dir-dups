package minilang

import (
	"bufio"
	"greasytoad/cleanup/parser"
	"io"
	"strings"
)

func ProcessManifest(minilang io.Reader, input io.Reader) (string, error) {
	_, err := parserMinilang(minilang)
	return "", err
}

func parserMinilang(r io.Reader) (minilangInstruction, error) {
	par := parser.GetMinilangParser()
	scanner := bufio.NewScanner(r)
	mi := []parser.InstructionNode{}
	for scanner.Scan() {
		line := strip(scanner.Text())
		if line == "" {
			continue
		}
		ast, err := par.ParseString(line)
		if err != nil {
			return nil, err
		}
		instr := ast.(parser.InstructionNode)
		mi = append(mi, instr)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mi, nil
}

type minilangInstruction []parser.InstructionNode

func strip(s string) string {
	s = strings.Trim(s, " \n\t")
	if strings.HasPrefix(s, "#") {
		return ""
	}
	return s
}
