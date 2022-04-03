package cleanup

import (
	"bufio"
	"bytes"
	"fmt"
	"greasytoad/cleanup/parser"
	"io"
	"strings"
)

func ReadScript(r io.Reader) (Script, error) {
	par := parser.GetMinilangParser()
	scanner := bufio.NewScanner(r)
	script := []instruction{}
	for scanner.Scan() {
		line := strip(scanner.Text())
		if line == "" {
			continue
		}
		ast, err := par.ParseString(line)
		if err != nil {
			return nil, err
		}
		script = append(script, instruction{
			inode: ast.(parser.InstructionNode),
			line:  line,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return script, nil
}

// ProcessManifestWithScript processes the input manifest and returns modified manifest according to the script. The processing is as follows:
//
// - Read the input lines and collect the lines into "hash groups". A hash group is a set of lines from the input manifest with the same hash.
//
// - Infer new actions per hash group.
//
// - Again iterate through input lines, and modity them accorting to the actions inferred in the previous step.
func ProcessManifestWithScript(r io.Reader, script Script, w io.Writer) error {
	inputBytes, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	manifest, err := ReadManifest(bytes.NewReader(inputBytes))
	if err != nil {
		return fmt.Errorf("error when reading manifest: %s", err)
	}

	hg := collectHashGroups(manifest)

	scanner := bufio.NewScanner(bytes.NewReader(inputBytes))
	for scanner.Scan() {
		line := scanner.Text()
		me, err := ParseLineToManifestEntry(strings.TrimSpace(line))
		if err != nil {
			fmt.Fprintln(w, line)
			continue
		}
		me, err = applyScriptToManifestEntry(script, me, hg)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, me.String())
	}

	return nil
}

func collectHashGroups(mes []ManifestEntry) map[string][]ManifestEntry {
	hg := make(map[string][]ManifestEntry)
	for _, me := range mes {
		if _, ok := hg[me.Hash]; ok {
			hg[me.Hash] = append(hg[me.Hash], me)
		} else {
			hg[me.Hash] = []ManifestEntry{me}
		}
	}
	return hg
}

func applyScriptToManifestEntry(script Script, me ManifestEntry, hashGroups map[string][]ManifestEntry) (ManifestEntry, error) {
	entries := hashGroups[me.Hash]

	resultEntry := me
	matchingInstructions := make(map[ManifestOperation]instruction)

	for _, inst := range script {
		if result, err := inst.apply(entries); err == nil {
			modified := getEntryWithSamePath(me, result)
			resultEntry = modified
			matchingInstructions[modified.Operation] = inst
			if len(matchingInstructions) > 1 {
				// TODO add error message with the offending instructions taken from inst
				return me, fmt.Errorf("single manifest entry matched two contradictory instructions: %v", matchingInstructions)
			}
		} else {
			return me, err
		}
	}
	return resultEntry, nil
}

type Script []instruction

type instruction struct {
	inode parser.InstructionNode
	line  string
}

func (s instruction) apply(ments []ManifestEntry) ([]ManifestEntry, error) {
	mentAlias := make(map[ManifestEntry]string)

	// process matches
	for _, matchWithAlias := range s.inode.Matches {
		// all matches must match some path in ManifestEntries
		hasMatch := false
		for _, ment := range ments {
			if strings.Contains(ment.Path, matchWithAlias.Match) {
				hasMatch = true
				if matchWithAlias.Alias != "" {
					if alias := mentAlias[ment]; alias != "" && alias != matchWithAlias.Alias {
						return ments, fmt.Errorf(`single path "%s" matching two different aliases: "%s" and "%s"`, ment.Path, alias, matchWithAlias.Alias)
					}
					mentAlias[ment] = matchWithAlias.Alias
				}
				break
			}
		}
		if !hasMatch {
			// some match string was not a part of any path. this is ok. abort
			return ments, nil
		}
	}

	// apply actions to aliases
	result := []ManifestEntry{}
	for _, me := range ments {
		alias, ok := mentAlias[me]
		if !ok {
			result = append(result, me)
			continue
		}

		for _, action := range s.inode.Actions {
			if action.Alias == alias {
				result = append(result, ManifestEntry{
					Operation: ManifestOperation(action.Action),
					Hash:      me.Hash,
					Path:      me.Path,
				})
				continue
			}
		}

		result = append(result, me)
	}

	return result, nil
}

func getEntryWithSamePath(needle ManifestEntry, haystack []ManifestEntry) ManifestEntry {
	for _, hay := range haystack {
		if needle.Hash != hay.Hash {
			panic(fmt.Sprintf("BUG. Assumed that needle and haystack have the same hash. %v, %v", needle, haystack))
		}
		if needle.Path == hay.Path {
			return hay
		}
	}
	panic(fmt.Sprintf("BUG. No needle in haystack. needle: %v, haystack: %v", needle, haystack))
}

func strip(s string) string {
	s = strings.Trim(s, " \n\t")
	if strings.HasPrefix(s, "#") {
		return ""
	}
	return s
}
