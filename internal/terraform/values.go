package terraform

import (
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// referencePattern matches Terraform references: aws_*, google_*, azurerm_*, var.*, local.*, data.*, module.*
var referencePattern = regexp.MustCompile(`^(aws_|google_|azurerm_|var\.|local\.|data\.|module\.)`)

// numberPattern matches integers and decimals
var numberPattern = regexp.MustCompile(`^\d+(\.\d+)?$`)

// SetAttribute sets a single attribute on a block body, coercing the string value
// to the appropriate cty/HCL type.
//
// Value type inference:
//   - forceString=true -> always cty.StringVal
//   - "s:VALUE" prefix -> cty.StringVal (explicit string coercion)
//   - "true"/"false" -> cty.Bool
//   - numeric string -> cty.Number
//   - [a,b,c] -> list (raw tokens, elements individually typed)
//   - {key:val} with ":" -> raw expression via jsonencode()
//   - {expr} without ":" -> raw expression
//   - aws_xxx.name.attr, var.xxx etc -> reference (SetAttributeTraversal)
//   - everything else -> cty.StringVal
func SetAttribute(body *hclwrite.Body, key, value string, forceString bool) {
	// String type prefix: s:VALUE forces string type
	if !forceString && strings.HasPrefix(value, "s:") {
		body.SetAttributeValue(key, cty.StringVal(value[2:]))
		return
	}

	if forceString {
		body.SetAttributeValue(key, cty.StringVal(value))
		return
	}

	// Bool
	if value == "true" {
		body.SetAttributeValue(key, cty.True)
		return
	}
	if value == "false" {
		body.SetAttributeValue(key, cty.False)
		return
	}

	// Number
	if numberPattern.MatchString(value) {
		body.SetAttributeRaw(key, rawTokens(value))
		return
	}

	// List
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		setListAttribute(body, key, value)
		return
	}

	// Map/Expression: starts with {
	if strings.HasPrefix(value, "{") {
		if strings.Contains(value, ":") {
			// JSON object -> jsonencode()
			body.SetAttributeRaw(key, rawTokens("jsonencode("+value+")"))
		} else {
			body.SetAttributeRaw(key, rawTokens(value))
		}
		return
	}

	// Reference
	if referencePattern.MatchString(value) {
		// If value contains brackets (index notation), use raw tokens
		if strings.Contains(value, "[") {
			body.SetAttributeRaw(key, rawTokens(value))
			return
		}
		traversal := parseTraversal(value)
		if traversal != nil {
			body.SetAttributeTraversal(key, traversal)
			return
		}
		// Fallback: raw tokens if traversal parsing fails
		body.SetAttributeRaw(key, rawTokens(value))
		return
	}

	// Interpolated string: contains ${...}
	if strings.Contains(value, "${") {
		// Write as raw quoted string: "...${expr}..."
		body.SetAttributeRaw(key, rawTokens(`"`+value+`"`))
		return
	}

	// Default: string
	body.SetAttributeValue(key, cty.StringVal(value))
}

// setListAttribute handles [a,b,c] list values with proper element typing.
func setListAttribute(body *hclwrite.Body, key, value string) {
	inner := strings.TrimSpace(value[1 : len(value)-1])
	if inner == "" {
		body.SetAttributeRaw(key, rawTokens("[]"))
		return
	}

	elements := splitListElements(inner)
	var tokenGroups []hclwrite.Tokens
	for _, elem := range elements {
		elem = strings.TrimSpace(elem)
		tokenGroups = append(tokenGroups, tokensForElement(elem))
	}

	body.SetAttributeRaw(key, hclwrite.TokensForTuple(tokenGroups))
}

// tokensForElement returns hclwrite.Tokens for a single list element,
// inferring whether it's a reference, number, bool, or string.
func tokensForElement(elem string) hclwrite.Tokens {
	// Raw identifier (force unquoted) — prefixed with !
	if strings.HasPrefix(elem, "!") {
		return rawTokens(elem[1:])
	}
	// Reference
	if referencePattern.MatchString(elem) {
		traversal := parseTraversal(elem)
		if traversal != nil {
			return hclwrite.TokensForTraversal(traversal)
		}
		return rawTokens(elem)
	}
	// Number
	if numberPattern.MatchString(elem) {
		return rawTokens(elem)
	}
	// Bool
	if elem == "true" || elem == "false" {
		return rawTokens(elem)
	}
	// Already quoted
	if len(elem) >= 2 && elem[0] == '"' && elem[len(elem)-1] == '"' {
		return hclwrite.TokensForValue(cty.StringVal(elem[1 : len(elem)-1]))
	}
	// Default: string
	return hclwrite.TokensForValue(cty.StringVal(elem))
}

// parseTraversal converts a dotted reference string into an hcl.Traversal.
// e.g., "var.vpc_id" -> Traversal{TraverseRoot{var}, TraverseAttr{vpc_id}}
// e.g., "aws_vpc.main.id" -> Traversal{TraverseRoot{aws_vpc}, TraverseAttr{main}, TraverseAttr{id}}
func parseTraversal(ref string) hcl.Traversal {
	parts := strings.Split(ref, ".")
	if len(parts) == 0 {
		return nil
	}

	traversal := hcl.Traversal{
		hcl.TraverseRoot{Name: parts[0]},
	}
	for _, part := range parts[1:] {
		traversal = append(traversal, hcl.TraverseAttr{Name: part})
	}
	return traversal
}

// rawTokens creates a simple token sequence from a raw string.
func rawTokens(s string) hclwrite.Tokens {
	return hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(s)},
	}
}

// splitListElements splits comma-separated list elements, respecting nested brackets.
func splitListElements(s string) []string {
	var elements []string
	depth := 0
	current := strings.Builder{}

	for _, ch := range s {
		switch ch {
		case '[', '(':
			depth++
			current.WriteRune(ch)
		case ']', ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				elements = append(elements, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		elements = append(elements, current.String())
	}
	return elements
}

// RemoveAttribute removes an attribute from a block body.
func RemoveAttribute(body *hclwrite.Body, key string) {
	body.RemoveAttribute(key)
}
