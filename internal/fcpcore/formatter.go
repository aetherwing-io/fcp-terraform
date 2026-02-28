package fcpcore

import "strings"

// FormatResult formats a result with prefix convention.
// If success is false, returns "ERROR: message" (prefix is ignored).
// If success is true and prefix is given, returns "prefix message".
// Otherwise returns just the message.
//
// Prefix conventions:
//
//	+ created
//	~ modified (edge/connection)
//	* changed (property)
//	- removed
//	! meta/group operation
//	@ bulk/layout operation
func FormatResult(success bool, message string, prefix ...string) string {
	if !success {
		return "ERROR: " + message
	}
	if len(prefix) > 0 && prefix[0] != "" {
		return prefix[0] + " " + message
	}
	return message
}

// Suggest finds the closest candidate for a misspelled input using
// Levenshtein distance. Returns empty string if no candidate is
// close enough (distance > 3).
func Suggest(input string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	best := ""
	bestDist := 999

	for _, candidate := range candidates {
		dist := levenshtein(strings.ToLower(input), strings.ToLower(candidate))
		if dist < bestDist {
			bestDist = dist
			best = candidate
		}
	}

	if bestDist <= 3 {
		return best
	}
	return ""
}

// levenshtein computes the Levenshtein distance between two strings.
func levenshtein(a, b string) int {
	m := len(a)
	n := len(b)

	prev := make([]int, n+1)
	for j := 0; j <= n; j++ {
		prev[j] = j
	}

	for i := 1; i <= m; i++ {
		prevDiag := prev[0]
		prev[0] = i
		for j := 1; j <= n; j++ {
			temp := prev[j]
			if a[i-1] == b[j-1] {
				prev[j] = prevDiag
			} else {
				minVal := prevDiag
				if prev[j-1] < minVal {
					minVal = prev[j-1]
				}
				if prev[j] < minVal {
					minVal = prev[j]
				}
				prev[j] = 1 + minVal
			}
			prevDiag = temp
		}
	}

	return prev[n]
}
