package terraform

import "strings"

// Selector represents a parsed @-prefixed selector.
type Selector struct {
	Type    string // e.g., "type", "provider", "kind", "tag", "all"
	Value   string // e.g., "aws_instance", "aws", "prod"
	Negated bool
	Raw     string // original token
}

// ParseSelector parses a raw selector string like "@type:aws_instance" into a Selector.
func ParseSelector(raw string) Selector {
	s := raw
	if !strings.HasPrefix(s, "@") {
		return Selector{Raw: raw}
	}
	s = s[1:] // strip @

	negated := false
	if strings.HasPrefix(s, "not:") {
		negated = true
		s = s[4:]
	}

	sel := Selector{Raw: raw, Negated: negated}
	colonIdx := strings.Index(s, ":")
	if colonIdx >= 0 {
		sel.Type = s[:colonIdx]
		sel.Value = s[colonIdx+1:]
	} else {
		sel.Type = s
	}
	return sel
}

// ResolveSelector returns all BlockRefs matching a single selector.
func ResolveSelector(sel Selector, model *TerraformModel) []*BlockRef {
	var result []*BlockRef

	switch sel.Type {
	case "all":
		for _, ref := range model.Index.ByQualified {
			result = append(result, ref)
		}
	case "type":
		result = model.Index.FindByType(sel.Value)
	case "kind":
		result = model.Index.FindByKind(sel.Value)
	case "provider":
		result = model.Index.FindByProvider(sel.Value)
	case "tag":
		key, value := parseTagSelector(sel.Value)
		result = model.Index.FindByTag(key, value)
	}

	if sel.Negated {
		return negateRefs(result, model)
	}
	return result
}

// ResolveSelectorSet resolves multiple selector strings and returns their intersection.
// The selectors are raw strings from ParsedOp.Selectors.
func ResolveSelectorSet(rawSelectors []string, model *TerraformModel) []*BlockRef {
	if len(rawSelectors) == 0 {
		return nil
	}

	// Start with all blocks for the first selector
	first := ParseSelector(rawSelectors[0])
	result := ResolveSelector(first, model)

	// Intersect with each subsequent selector
	for i := 1; i < len(rawSelectors); i++ {
		sel := ParseSelector(rawSelectors[i])
		filtered := ResolveSelector(sel, model)
		result = intersectRefs(result, filtered)
	}

	return result
}

// parseTagSelector splits a tag selector value into key and optional value.
// "env=prod" -> ("env", "prod"), "env" -> ("env", "")
func parseTagSelector(s string) (key, value string) {
	eqIdx := strings.Index(s, "=")
	if eqIdx >= 0 {
		return s[:eqIdx], s[eqIdx+1:]
	}
	return s, ""
}

// negateRefs returns all blocks NOT in the excluded set.
func negateRefs(excluded []*BlockRef, model *TerraformModel) []*BlockRef {
	excludeSet := make(map[string]bool)
	for _, ref := range excluded {
		excludeSet[strings.ToLower(ref.FullType+"."+ref.Label)] = true
	}

	var result []*BlockRef
	for _, ref := range model.Index.ByQualified {
		qKey := strings.ToLower(ref.FullType + "." + ref.Label)
		if !excludeSet[qKey] {
			result = append(result, ref)
		}
	}
	return result
}

// intersectRefs returns BlockRefs present in both slices.
func intersectRefs(a, b []*BlockRef) []*BlockRef {
	set := make(map[string]bool)
	for _, ref := range b {
		set[strings.ToLower(ref.FullType+"."+ref.Label)] = true
	}

	var result []*BlockRef
	for _, ref := range a {
		if set[strings.ToLower(ref.FullType+"."+ref.Label)] {
			result = append(result, ref)
		}
	}
	return result
}
