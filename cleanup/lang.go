package cleanup

import (
	"bufio"
	"greasytoad/cleanup/parser"
	"io"
	"strings"
)

type Script []parser.InstructionNode

func ReadScript(r io.Reader) (Script, error) {
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

// func ProcessManifest(minilangReader io.Reader, inputReader io.Reader) (string, error) {
// 	mil, err := parserMinilang(minilangReader)
// 	inputBytes, err := ioutil.ReadAll(inputReader)
// 	if err != nil {
// 		return "", err
// 	}
// 	input := string(inputBytes)

// 	_ = mil
// 	_ = input
// 	return "", nil
// 	// hashGroups := collectHashGroups(input)

// 	// The processing is as follows:
// 	// - Read the input lines and collect the lines into "hash groups". A hash group is a set of lines from the input manifest with the same hash.
// 	// - Infer new actions per hash group.
// 	// - Again iterate through input lines, and modity them accorting to the actions inferred in the previous step.
// 	//
// 	// Inferring actions per hash group works as follows:
// 	// - check if hashgroup matches all the expressions. If so, apply the mapping.
// }

// func (mi minilangInstruction) process(r io.Reader) (string, error) {
// 	out := &strings.Builder{}
// 	scanner := bufio.NewScanner(r)
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if strings.HasPrefix(line, "#") {
// 			fmt.Fprintf(out, "%s\n", line)
// 			continue
// 		}
// 		// // HERE duplicated logic
// 		// parts := strings.Split(line, "\t")
// 		// lineAction, lineHash, linePath := parts[0], parts[1], parts[2]
// 		// lineko
// 	}
// 	if err := scanner.Err(); err != nil {
// 		return "", err
// 	}
// 	return out.String(), nil
// }

func strip(s string) string {
	s = strings.Trim(s, " \n\t")
	if strings.HasPrefix(s, "#") {
		return ""
	}
	return s
}
