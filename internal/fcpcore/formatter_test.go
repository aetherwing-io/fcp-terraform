package fcpcore

import "testing"

func TestFormatResult_SuccessWithPrefix(t *testing.T) {
	got := FormatResult(true, "svc AuthService", "+")
	if got != "+ svc AuthService" {
		t.Errorf("FormatResult(+) = %q", got)
	}
}

func TestFormatResult_SuccessWithoutPrefix(t *testing.T) {
	got := FormatResult(true, "done")
	if got != "done" {
		t.Errorf("FormatResult(no prefix) = %q", got)
	}
}

func TestFormatResult_Error(t *testing.T) {
	got := FormatResult(false, "something broke")
	if got != "ERROR: something broke" {
		t.Errorf("FormatResult(error) = %q", got)
	}
}

func TestFormatResult_ErrorIgnoresPrefix(t *testing.T) {
	got := FormatResult(false, "bad input", "+")
	if got != "ERROR: bad input" {
		t.Errorf("FormatResult(error+prefix) = %q", got)
	}
}

func TestFormatResult_PrefixConventions(t *testing.T) {
	tests := []struct {
		prefix  string
		message string
		want    string
	}{
		{"+", "AuthService", "+ AuthService"},
		{"~", "edge A->B", "~ edge A->B"},
		{"*", "styled A", "* styled A"},
		{"-", "A", "- A"},
		{"!", "group Backend", "! group Backend"},
		{"@", "layout", "@ layout"},
	}
	for _, tt := range tests {
		got := FormatResult(true, tt.message, tt.prefix)
		if got != tt.want {
			t.Errorf("FormatResult(%q, %q) = %q, want %q", tt.prefix, tt.message, got, tt.want)
		}
	}
}

func TestSuggest(t *testing.T) {
	candidates := []string{"add", "remove", "connect", "style", "label", "badge"}

	tests := []struct {
		input string
		want  string
	}{
		{"add", "add"},
		{"ad", "add"},
		{"styel", "style"},
		{"labek", "label"},
		{"zzzzzzz", ""},
		{"bade", "badge"},
	}

	for _, tt := range tests {
		got := Suggest(tt.input, candidates)
		if got != tt.want {
			t.Errorf("Suggest(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSuggest_EmptyCandidates(t *testing.T) {
	got := Suggest("test", []string{})
	if got != "" {
		t.Errorf("Suggest(empty candidates) = %q, want empty", got)
	}
}
