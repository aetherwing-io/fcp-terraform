package fcpcore

import "strings"

// Tokenize splits an operation string into tokens, respecting quoted strings.
// Double-quoted strings are treated as single tokens with quotes stripped.
// Escape sequences within quotes: \" produces literal ", \\ produces \, \n produces newline.
// Embedded quotes in unquoted tokens (e.g., key:"value") preserve quotes in the token
// for downstream detection.
func Tokenize(input string) []string {
	var tokens []string
	i := 0
	n := len(input)

	for i < n {
		// Skip whitespace
		for i < n && input[i] == ' ' {
			i++
		}
		if i >= n {
			break
		}

		if input[i] == '"' {
			// Quoted string — strip outer quotes
			i++ // skip opening quote
			var token strings.Builder
			for i < n && input[i] != '"' {
				if input[i] == '\\' && i+1 < n {
					next := input[i+1]
					if next == 'n' {
						token.WriteByte('\n')
						i += 2
					} else {
						i++
						token.WriteByte(input[i])
						i++
					}
				} else {
					token.WriteByte(input[i])
					i++
				}
			}
			if i < n {
				i++ // skip closing quote
			}
			tokens = append(tokens, token.String())
		} else {
			// Unquoted token — preserve embedded quotes (e.g., key:"value")
			var token strings.Builder
			for i < n && input[i] != ' ' {
				if input[i] == '"' {
					// Embedded quoted value — preserve quotes in token
					token.WriteByte('"')
					i++ // skip opening quote
					for i < n && input[i] != '"' {
						if input[i] == '\\' && i+1 < n {
							next := input[i+1]
							if next == 'n' {
								token.WriteByte('\n')
								i += 2
							} else {
								i++
								token.WriteByte(input[i])
								i++
							}
						} else {
							token.WriteByte(input[i])
							i++
						}
					}
					if i < n {
						token.WriteByte('"')
						i++ // skip closing quote
					}
				} else {
					token.WriteByte(input[i])
					i++
				}
			}
			// Convert literal \n in unquoted tokens to actual newlines
			tokens = append(tokens, strings.ReplaceAll(token.String(), "\\n", "\n"))
		}
	}

	return tokens
}

// IsKeyValue checks if a token is a key:value pair.
// Must contain ":" but not start with "@" (selectors) and not be an arrow.
// The colon must not be at position 0 or at the end.
// The key portion (before the first colon) must be a valid identifier: [a-zA-Z0-9_-] only.
func IsKeyValue(token string) bool {
	if strings.HasPrefix(token, "@") {
		return false
	}
	if IsArrow(token) {
		return false
	}
	idx := strings.Index(token, ":")
	if idx <= 0 || idx >= len(token)-1 {
		return false
	}
	key := token[:idx]
	for _, ch := range key {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

// ParseKeyValue parses a key:value token. The value may include colons.
// Strips surrounding quotes from the value for backwards compatibility.
func ParseKeyValue(token string) (key, value string) {
	idx := strings.Index(token, ":")
	key = token[:idx]
	value = token[idx+1:]
	// Strip surrounding quotes preserved by tokenizer
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	return key, value
}

// ParseKeyValueWithMeta parses a key:value token with metadata about quoting.
// Returns the unquoted value plus a wasQuoted flag.
func ParseKeyValueWithMeta(token string) (key, value string, wasQuoted bool) {
	idx := strings.Index(token, ":")
	key = token[:idx]
	value = token[idx+1:]
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
		wasQuoted = true
	}
	return key, value, wasQuoted
}

// IsArrow checks if a token is an arrow operator.
func IsArrow(token string) bool {
	return token == "->" || token == "<->" || token == "--"
}

// IsSelector checks if a token is a selector (@-prefixed).
func IsSelector(token string) bool {
	return strings.HasPrefix(token, "@")
}
