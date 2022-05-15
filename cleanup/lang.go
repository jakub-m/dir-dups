package cleanup

import (
	"bufio"
	"bytes"
	"fmt"
	"greasytoad/cleanup/parser"
	coll "greasytoad/collections"
	"greasytoad/log"
	"io"
	"strings"
)

const other = "other"

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
	// OK:
	// 1. Many instructions of the Script match the entires
	// 2. Different instructions in the script apply actions to ManifestEntries
	// NOT OK:
	// 3. Contradictory actions for a ManifestEntry

	entriesSameHash := hashGroups[me.Hash]

	resultEntry := me
	matchingInstructions := make(map[parser.ManifestOperation]instruction)

	for _, inst := range script {
		if modifiedEntries, err := inst.apply(entriesSameHash); err == nil {
			if modified, ok := getEntryWithSamePath(me, modifiedEntries); ok {
				resultEntry = modified
				matchingInstructions[modified.Operation] = inst
				if len(matchingInstructions) > 1 {
					lines := []string{}
					for _, mi := range matchingInstructions {
						lines = append(lines, mi.line)
					}
					return me, fmt.Errorf("single manifest entry matched contradictory instructions:\n%s\n%s", me, strings.Join(lines, "\n"))
				}
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

// apply returns those input Manifest entires that an action was executed upon.
func (s instruction) apply(ments []ManifestEntry) ([]ManifestEntry, error) {
	if !coll.All(ments, func(m ManifestEntry) bool { return m.Hash == ments[0].Hash }) {
		log.Fatalf("RATS! Assumed that manifest entries have the same hash: %v", ments)
	}

	// Go through all the parts of the match expression and map the matches to aliases (if there is an alias).
	// All of the parts of the match expression must have a matching ManiestEntry.
	type mentWithAlias struct {
		ment  ManifestEntry
		alias string
	}
	matchingManifestEntries := []mentWithAlias{}
	isMatchOther := false
	for _, instrMatchWithAlias := range s.inode.Matches {
		if instrMatchWithAlias.MatchOther {
			// If there is `if ... and other then...`, here we mark that there is such "other" to evaluate later if there was indeed
			// a manifest entry not matched by other parts of the instruction.
			isMatchOther = true
			continue
		}
		instructionHasSuccessfulMatch := false
		for _, manifestEntry := range ments {
			if strings.Contains(manifestEntry.Path, instrMatchWithAlias.Match) {
				matchingManifestEntries = append(matchingManifestEntries, mentWithAlias{
					ment:  manifestEntry,
					alias: instrMatchWithAlias.Alias,
				})
				instructionHasSuccessfulMatch = true
			}
		}
		if !instructionHasSuccessfulMatch {
			// Some manifest entry did not have a match. It's ok.
			return []ManifestEntry{}, nil
		}
	}

	otherEntries := []mentWithAlias{}
	for _, ment := range ments {
		hadMatch := coll.Any(matchingManifestEntries, func(mewa mentWithAlias) bool { return mewa.ment == ment })
		if !hadMatch {
			otherEntries = append(otherEntries, mentWithAlias{
				ment:  ment,
				alias: other,
			})
		}
	}

	// Here handle "other" match as in `if "foo" and "bar" and other`. Such "other" match means that there must
	// be manifest entries that do not match other matches.
	if isMatchOther && len(otherEntries) == 0 {
		// The instruction wants other and there is no "other", so stop now.
		return []ManifestEntry{}, nil

	}

	matchingManifestEntries = append(matchingManifestEntries, otherEntries...)

	// Now apply actions to aliases
	mentsWithAppliedActions := []ManifestEntry{}
	for _, action := range s.inode.Actions {
		for _, mentWithAlias := range matchingManifestEntries {
			if action.Alias != "" && action.Alias == mentWithAlias.alias {
				m := mentWithAlias.ment
				m.Operation = action.Action
				mentsWithAppliedActions = append(mentsWithAppliedActions, m)
			}
		}
	}

	// Figure if the same manifest entry got two distinct actions. If yes then return an error, if no, then return de-duplicated list of manifest entries.
	result := []ManifestEntry{}
	for _, ment := range ments {
		matchingMents := coll.FilterSlice(mentsWithAppliedActions, func(ma ManifestEntry) bool { return ma.Path == ment.Path })
		matchingMents = coll.Deduplicate(matchingMents)
		if len(matchingMents) > 1 {
			return ments, fmt.Errorf(`single instruction "%s" produced contradictory actions: %v`, s.line, matchingMents)
		}
		if len(matchingMents) == 1 {
			result = append(result, matchingMents[0])
		}
	}
	return result, nil
}

func getEntryWithSamePath(needle ManifestEntry, haystack []ManifestEntry) (ManifestEntry, bool) {
	for _, hay := range haystack {
		if needle.Hash != hay.Hash {
			panic(fmt.Sprintf("BUG. Assumed that needle and haystack have the same hash. %v, %v", needle, haystack))
		}
		if needle.Path == hay.Path {
			return hay, true
		}
	}
	return needle, false
}

func strip(s string) string {
	s = strings.Trim(s, " \n\t")
	if strings.HasPrefix(s, "#") {
		return ""
	}
	return s
}

// TODO when there is only one matching expression, support default (but only then)
