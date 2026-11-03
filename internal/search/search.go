// Package search implements catalog search and ranking. It contains no TUI
// code and no JSON parsing.
package search

import (
	"sort"
	"strconv"
	"strings"

	"ntexplorer/internal/catalog"
)

// Result is one ranked search hit.
type Result struct {
	API     catalog.API
	Score   int
	Reasons []string
}

// StructureResult is one ranked structure search hit.
type StructureResult struct {
	Structure catalog.Structure
	Score     int
	Reasons   []string
}

// ConstantResult is one ranked constant search hit.
type ConstantResult struct {
	Constant catalog.Constant
	Score    int
	Reasons  []string
}

// StatusResult is one ranked NTSTATUS search hit.
type StatusResult struct {
	Status  catalog.NTStatus
	Score   int
	Reasons []string
}

// Query ranks APIs against a case-insensitive query. Every whitespace-separated
// term must match at least one field for an API to be included.
//
// Ranking order follows the specification:
//  1. exact API name
//  2. API name prefix
//  3. API name substring
//  4. alias or related symbol
//  5. category or tag
//  6. description and parameter text
func Query(apis []catalog.API, query string, limit int) []Result {
	query = strings.TrimSpace(query)
	if query == "" {
		results := make([]Result, 0, len(apis))
		for _, api := range apis {
			results = append(results, Result{API: api})
		}
		return limitResults(results, limit)
	}

	terms := strings.Fields(strings.ToLower(query))
	full := strings.ToLower(query)
	results := make([]Result, 0, len(apis))
	for _, api := range apis {
		total := 0
		var reasons []string
		matchedAll := true
		for _, term := range terms {
			score, reason := scoreTerm(api, term)
			if score == 0 {
				matchedAll = false
				break
			}
			total += score
			reasons = appendUnique(reasons, reason)
		}
		if !matchedAll {
			continue
		}

		name := strings.ToLower(api.Name)
		if name == full {
			total += 2000
		} else if strings.HasPrefix(name, full) {
			total += 1200
		}
		// A full-phrase category match should outrank incidental word matches
		// elsewhere in a description.
		category := strings.ToLower(api.Category)
		if category == full {
			total += 900
			reasons = appendUnique(reasons, "category "+api.Category)
		} else if strings.Contains(category, full) {
			total += 500
		}
		for _, tag := range api.Tags {
			if strings.ToLower(tag) == full {
				total += 600
				reasons = appendUnique(reasons, "tag "+tag)
				break
			}
		}

		results = append(results, Result{API: api, Score: total, Reasons: reasons})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		a, b := strings.ToLower(results[i].API.Name), strings.ToLower(results[j].API.Name)
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})

	return limitResults(results, limit)
}

// QueryStructures ranks structures against a case-insensitive query. Every
// whitespace-separated term must match at least one field for a structure to be
// included. Names rank above member names, which rank above descriptions.
func QueryStructures(structures []catalog.Structure, query string, limit int) []StructureResult {
	query = strings.TrimSpace(query)
	if query == "" {
		results := make([]StructureResult, 0, len(structures))
		for _, structure := range structures {
			results = append(results, StructureResult{Structure: structure})
		}
		return limitStructures(results, limit)
	}

	terms := strings.Fields(strings.ToLower(query))
	full := strings.ToLower(query)
	results := make([]StructureResult, 0, len(structures))
	for _, structure := range structures {
		total := 0
		var reasons []string
		matchedAll := true
		for _, term := range terms {
			score, reason := scoreStructureTerm(structure, term)
			if score == 0 {
				matchedAll = false
				break
			}
			total += score
			reasons = appendUnique(reasons, reason)
		}
		if !matchedAll {
			continue
		}

		name := strings.ToLower(structure.Name)
		if name == full {
			total += 2000
		} else if strings.HasPrefix(name, full) {
			total += 1200
		}
		for _, tag := range structure.Tags {
			if strings.ToLower(tag) == full {
				total += 600
				reasons = appendUnique(reasons, "tag "+tag)
				break
			}
		}

		results = append(results, StructureResult{Structure: structure, Score: total, Reasons: reasons})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		a, b := strings.ToLower(results[i].Structure.Name), strings.ToLower(results[j].Structure.Name)
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})

	return limitStructures(results, limit)
}

// QueryConstants ranks constants against a case-insensitive query. Every
// whitespace-separated term must match at least one field for a constant to be
// included. Names rank above values, which rank above categories and text.
//
// Numeric terms are compared by value, so a query such as 0x1000, 4096, or
// 0x00001000 all match MEM_COMMIT.
func QueryConstants(constants []catalog.Constant, query string, limit int) []ConstantResult {
	query = strings.TrimSpace(query)
	if query == "" {
		results := make([]ConstantResult, 0, len(constants))
		for _, constant := range constants {
			results = append(results, ConstantResult{Constant: constant})
		}
		return limitConstants(results, limit)
	}

	terms := strings.Fields(strings.ToLower(query))
	full := strings.ToLower(query)
	results := make([]ConstantResult, 0, len(constants))
	for _, constant := range constants {
		total := 0
		var reasons []string
		matchedAll := true
		for _, term := range terms {
			score, reason := scoreConstantTerm(constant, term)
			if score == 0 {
				matchedAll = false
				break
			}
			total += score
			reasons = appendUnique(reasons, reason)
		}
		if !matchedAll {
			continue
		}

		name := strings.ToLower(constant.Name)
		if name == full {
			total += 2000
		} else if strings.HasPrefix(name, full) {
			total += 1200
		}
		if valueMatches(constant.Value, full) {
			total += 1500
			reasons = appendUnique(reasons, "value")
		}
		category := strings.ToLower(constant.Category)
		if category == full {
			total += 900
			reasons = appendUnique(reasons, "category "+constant.Category)
		}
		for _, tag := range constant.Tags {
			if strings.ToLower(tag) == full {
				total += 600
				reasons = appendUnique(reasons, "tag "+tag)
				break
			}
		}

		results = append(results, ConstantResult{Constant: constant, Score: total, Reasons: reasons})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		a, b := strings.ToLower(results[i].Constant.Name), strings.ToLower(results[j].Constant.Name)
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})

	return limitConstants(results, limit)
}

// QueryStatuses ranks NTSTATUS records against a case-insensitive query. Every
// whitespace-separated term must match at least one field for a record to be
// included. Names rank above values, which rank above severity and text.
//
// Numeric terms are compared by value, so 0xC0000022, c0000022, and
// 3221225506 all match STATUS_ACCESS_DENIED.
func QueryStatuses(statuses []catalog.NTStatus, query string, limit int) []StatusResult {
	query = strings.TrimSpace(query)
	if query == "" {
		results := make([]StatusResult, 0, len(statuses))
		for _, status := range statuses {
			results = append(results, StatusResult{Status: status})
		}
		return limitStatuses(results, limit)
	}

	terms := strings.Fields(strings.ToLower(query))
	full := strings.ToLower(query)
	results := make([]StatusResult, 0, len(statuses))
	for _, status := range statuses {
		total := 0
		var reasons []string
		matchedAll := true
		for _, term := range terms {
			score, reason := scoreStatusTerm(status, term)
			if score == 0 {
				matchedAll = false
				break
			}
			total += score
			reasons = appendUnique(reasons, reason)
		}
		if !matchedAll {
			continue
		}

		name := strings.ToLower(status.Name)
		if name == full {
			total += 2000
		} else if strings.HasPrefix(name, full) {
			total += 1200
		}
		if valueMatches(status.Value, full) {
			total += 1500
			reasons = appendUnique(reasons, "value")
		}
		if strings.EqualFold(status.Severity, full) {
			total += 900
			reasons = appendUnique(reasons, "severity "+status.Severity)
		}

		results = append(results, StatusResult{Status: status, Score: total, Reasons: reasons})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		a, b := strings.ToLower(results[i].Status.Name), strings.ToLower(results[j].Status.Name)
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})

	return limitStatuses(results, limit)
}

// Filter returns the APIs that match query without ranking metadata.
func Filter(apis []catalog.API, query string) []catalog.API {
	results := Query(apis, query, 0)
	out := make([]catalog.API, len(results))
	for i, r := range results {
		out[i] = r.API
	}
	return out
}

func limitResults(results []Result, limit int) []Result {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func limitStructures(results []StructureResult, limit int) []StructureResult {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func limitConstants(results []ConstantResult, limit int) []ConstantResult {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func limitStatuses(results []StatusResult, limit int) []StatusResult {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func scoreTerm(api catalog.API, term string) (int, string) {
	name := strings.ToLower(api.Name)
	switch {
	case name == term:
		return 1000, "name"
	case strings.HasPrefix(name, term):
		return 850, "name prefix"
	case strings.Contains(name, term):
		return 700, "name"
	}

	for _, alias := range api.Aliases {
		aliasLower := strings.ToLower(alias)
		if aliasLower == term {
			return 650, "alias " + alias
		}
		if strings.Contains(aliasLower, term) {
			return 500, "alias " + alias
		}
	}

	category := strings.ToLower(api.Category)
	if category == term {
		return 450, "category " + api.Category
	}
	if strings.Contains(category, term) {
		return 400, "category"
	}

	for _, tag := range api.Tags {
		tagLower := strings.ToLower(tag)
		if tagLower == term {
			return 380, "tag " + tag
		}
	}
	for _, tag := range api.Tags {
		if strings.Contains(strings.ToLower(tag), term) {
			return 300, "tag " + tag
		}
	}

	if strings.Contains(strings.ToLower(api.Summary), term) {
		return 260, "summary"
	}
	if strings.Contains(strings.ToLower(api.Description), term) {
		return 240, "description"
	}
	for _, p := range api.Parameters {
		if strings.Contains(strings.ToLower(p.Name), term) {
			return 220, "parameter " + p.Name
		}
	}
	for _, p := range api.Parameters {
		if strings.Contains(strings.ToLower(p.Description), term) {
			return 180, "parameter " + p.Name
		}
	}
	if strings.Contains(strings.ToLower(api.DLL), term) || strings.Contains(strings.ToLower(api.Header), term) {
		return 150, "module"
	}
	if strings.Contains(strings.ToLower(api.MalDevNotes), term) || strings.Contains(strings.ToLower(api.DefensiveNotes), term) {
		return 120, "notes"
	}
	if strings.Contains(strings.ToLower(api.Prototype), term) {
		return 100, "prototype"
	}
	return 0, ""
}

func scoreStructureTerm(structure catalog.Structure, term string) (int, string) {
	name := strings.ToLower(structure.Name)
	switch {
	case name == term:
		return 1000, "name"
	case strings.HasPrefix(name, term):
		return 850, "name prefix"
	case strings.Contains(name, term):
		return 700, "name"
	}

	for _, member := range structure.Members {
		memberName := strings.ToLower(member.Name)
		switch {
		case memberName == term:
			return 600, "member " + member.Name
		case strings.HasPrefix(memberName, term):
			return 550, "member " + member.Name
		case strings.Contains(memberName, term):
			return 500, "member " + member.Name
		}
	}
	for _, member := range structure.Members {
		if strings.Contains(strings.ToLower(member.Type), term) {
			return 350, "member type " + member.Name
		}
	}
	for _, member := range structure.Members {
		if strings.Contains(strings.ToLower(member.Description), term) {
			return 250, "member " + member.Name
		}
	}

	for _, tag := range structure.Tags {
		tagLower := strings.ToLower(tag)
		if tagLower == term {
			return 380, "tag " + tag
		}
	}
	for _, tag := range structure.Tags {
		if strings.Contains(strings.ToLower(tag), term) {
			return 300, "tag " + tag
		}
	}

	if strings.Contains(strings.ToLower(structure.Summary), term) {
		return 260, "summary"
	}
	if strings.Contains(strings.ToLower(structure.Notes), term) {
		return 200, "notes"
	}
	if strings.Contains(strings.ToLower(structure.Definition), term) {
		return 150, "definition"
	}
	for _, ref := range structure.References {
		if strings.Contains(strings.ToLower(ref.Title), term) {
			return 100, "reference"
		}
	}
	return 0, ""
}

func scoreConstantTerm(constant catalog.Constant, term string) (int, string) {
	name := strings.ToLower(constant.Name)
	switch {
	case name == term:
		return 1000, "name"
	case strings.HasPrefix(name, term):
		return 850, "name prefix"
	case strings.Contains(name, term):
		return 700, "name"
	}

	if valueMatches(constant.Value, term) {
		return 900, "value"
	}
	if strings.Contains(strings.ToLower(constant.Value), term) {
		return 500, "value"
	}

	category := strings.ToLower(constant.Category)
	if category == term {
		return 450, "category " + constant.Category
	}
	if strings.Contains(category, term) {
		return 400, "category"
	}

	for _, tag := range constant.Tags {
		tagLower := strings.ToLower(tag)
		if tagLower == term {
			return 380, "tag " + tag
		}
	}
	for _, tag := range constant.Tags {
		if strings.Contains(strings.ToLower(tag), term) {
			return 300, "tag " + tag
		}
	}

	if strings.Contains(strings.ToLower(constant.Description), term) {
		return 260, "description"
	}
	for _, ref := range constant.References {
		if strings.Contains(strings.ToLower(ref.Title), term) {
			return 100, "reference"
		}
	}
	return 0, ""
}

func scoreStatusTerm(status catalog.NTStatus, term string) (int, string) {
	name := strings.ToLower(status.Name)
	switch {
	case name == term:
		return 1000, "name"
	case strings.HasPrefix(name, term):
		return 850, "name prefix"
	case strings.Contains(name, term):
		return 700, "name"
	}

	if valueMatches(status.Value, term) {
		return 900, "value"
	}
	if strings.Contains(strings.ToLower(status.Value), term) {
		return 500, "value"
	}

	if strings.EqualFold(status.Severity, term) {
		return 450, "severity " + status.Severity
	}

	if strings.Contains(strings.ToLower(status.Description), term) {
		return 260, "description"
	}
	for _, ref := range status.References {
		if strings.Contains(strings.ToLower(ref.Title), term) {
			return 100, "reference"
		}
	}
	return 0, ""
}

// valueMatches compares a constant value and a query term numerically when
// both parse, falling back to a case-insensitive string comparison.
func valueMatches(value, term string) bool {
	if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(term)) {
		return true
	}
	a, okA := parseNumericValue(value)
	b, okB := parseNumericValue(term)
	return okA && okB && a == b
}

func parseNumericValue(text string) (uint64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	if text == "" {
		return 0, false
	}
	value, err := strconv.ParseUint(text, base, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func appendUnique(items []string, item string) []string {
	if item == "" {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
