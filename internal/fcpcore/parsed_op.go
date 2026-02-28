package fcpcore

import "strings"

// ParsedOp represents a successfully parsed FCP operation.
type ParsedOp struct {
	Verb         string
	Positionals  []string
	Params       map[string]string
	Selectors    []string
	QuotedParams map[string]bool // Param keys whose values were explicitly quoted
	Raw          string
}

// ParseError represents a parse failure.
type ParseError struct {
	Error string
	Raw   string
}

// ParseResult is the return type of ParseOp — either a *ParsedOp or a *ParseError.
type ParseResult struct {
	Op  *ParsedOp
	Err *ParseError
}

// ParseOp parses an operation string into a ParsedOp or ParseError.
// First token becomes the verb. Remaining tokens are classified:
//
//	@-prefixed  -> selectors
//	key:value   -> params
//	everything else -> positionals (in order)
func ParseOp(input string) ParseResult {
	raw := strings.TrimSpace(input)
	tokens := Tokenize(raw)

	if len(tokens) == 0 {
		return ParseResult{Err: &ParseError{Error: "empty operation", Raw: raw}}
	}

	verb := strings.ToLower(tokens[0])
	positionals := []string{}
	params := map[string]string{}
	selectors := []string{}
	quotedParams := map[string]bool{}

	for i := 1; i < len(tokens); i++ {
		token := tokens[i]
		if IsSelector(token) {
			selectors = append(selectors, token)
		} else if IsKeyValue(token) {
			key, value, wasQuoted := ParseKeyValueWithMeta(token)
			params[key] = value
			if wasQuoted {
				quotedParams[key] = true
			}
		} else {
			positionals = append(positionals, token)
		}
	}

	return ParseResult{
		Op: &ParsedOp{
			Verb:         verb,
			Positionals:  positionals,
			Params:       params,
			Selectors:    selectors,
			QuotedParams: quotedParams,
			Raw:          raw,
		},
	}
}

// IsError returns true if the parse result is an error.
func (r ParseResult) IsError() bool {
	return r.Err != nil
}
