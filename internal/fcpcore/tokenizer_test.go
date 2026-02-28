package fcpcore

import (
	"testing"
)

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTokenize_SimpleTokens(t *testing.T) {
	got := Tokenize("add svc AuthService")
	want := []string{"add", "svc", "AuthService"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(simple) = %v, want %v", got, want)
	}
}

func TestTokenize_QuotedStrings(t *testing.T) {
	got := Tokenize(`add svc "Auth Service" theme:blue`)
	want := []string{"add", "svc", "Auth Service", "theme:blue"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(quoted) = %v, want %v", got, want)
	}
}

func TestTokenize_EscapedQuotes(t *testing.T) {
	got := Tokenize(`label A "say \"hello\""`)
	want := []string{"label", "A", `say "hello"`}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(escaped quotes) = %v, want %v", got, want)
	}
}

func TestTokenize_EmptyInput(t *testing.T) {
	got := Tokenize("")
	if len(got) != 0 {
		t.Errorf("Tokenize(empty) = %v, want []", got)
	}
}

func TestTokenize_WhitespaceOnly(t *testing.T) {
	got := Tokenize("   ")
	if len(got) != 0 {
		t.Errorf("Tokenize(whitespace) = %v, want []", got)
	}
}

func TestTokenize_MultipleSpaces(t *testing.T) {
	got := Tokenize("add   svc   A")
	want := []string{"add", "svc", "A"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(multiple spaces) = %v, want %v", got, want)
	}
}

func TestTokenize_NewlineInUnquoted(t *testing.T) {
	got := Tokenize(`add svc Container\nRegistry`)
	want := []string{"add", "svc", "Container\nRegistry"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(newline unquoted) = %v, want %v", got, want)
	}
}

func TestTokenize_NewlineInQuoted(t *testing.T) {
	got := Tokenize(`add svc "Container\nRegistry"`)
	want := []string{"add", "svc", "Container\nRegistry"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(newline quoted) = %v, want %v", got, want)
	}
}

func TestTokenize_EmbeddedQuotedValue(t *testing.T) {
	got := Tokenize(`label:"Line1\nLine2"`)
	want := []string{"label:\"Line1\nLine2\""}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(embedded quoted) = %v, want %v", got, want)
	}
}

func TestTokenize_MultipleNewlines(t *testing.T) {
	got := Tokenize(`add svc A\nB\nC`)
	want := []string{"add", "svc", "A\nB\nC"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(multi newline) = %v, want %v", got, want)
	}
}

func TestTokenize_SingleToken(t *testing.T) {
	got := Tokenize("add")
	want := []string{"add"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(single) = %v, want %v", got, want)
	}
}

func TestTokenize_EmptyQuotedString(t *testing.T) {
	got := Tokenize(`""`)
	want := []string{""}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(empty quotes) = %v, want %v", got, want)
	}
}

func TestTokenize_UnclosedQuote(t *testing.T) {
	got := Tokenize(`"hello world`)
	want := []string{"hello world"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(unclosed) = %v, want %v", got, want)
	}
}

func TestTokenize_EscapedBackslash(t *testing.T) {
	got := Tokenize(`"path\\dir"`)
	want := []string{`path\dir`}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(escaped backslash) = %v, want %v", got, want)
	}
}

func TestTokenize_ColonsInValue(t *testing.T) {
	got := Tokenize("url:http://example.com")
	want := []string{"url:http://example.com"}
	if !sliceEqual(got, want) {
		t.Errorf("Tokenize(colons in value) = %v, want %v", got, want)
	}
}

func TestIsKeyValue(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"theme:blue", true},
		{"url:http://x", true},
		{"@type:db", false},
		{"->", false},
		{"hello", false},
		{"key:", false},
		{":value", false},
	}
	for _, tt := range tests {
		got := IsKeyValue(tt.input)
		if got != tt.want {
			t.Errorf("IsKeyValue(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseKeyValue(t *testing.T) {
	key, value := ParseKeyValue("theme:blue")
	if key != "theme" || value != "blue" {
		t.Errorf("ParseKeyValue(theme:blue) = %q, %q", key, value)
	}

	key, value = ParseKeyValue("url:http://x:8080")
	if key != "url" || value != "http://x:8080" {
		t.Errorf("ParseKeyValue(url:http://x:8080) = %q, %q", key, value)
	}
}

func TestParseKeyValue_StripsQuotes(t *testing.T) {
	key, value := ParseKeyValue(`label:"Line1` + "\n" + `Line2"`)
	if key != "label" || value != "Line1\nLine2" {
		t.Errorf("ParseKeyValue(quoted) = %q, %q", key, value)
	}
}

func TestParseKeyValueWithMeta(t *testing.T) {
	key, value, wasQuoted := ParseKeyValueWithMeta(`engine_version:"15"`)
	if key != "engine_version" || value != "15" || !wasQuoted {
		t.Errorf("ParseKeyValueWithMeta(quoted) = %q, %q, %v", key, value, wasQuoted)
	}

	key, value, wasQuoted = ParseKeyValueWithMeta("port:80")
	if key != "port" || value != "80" || wasQuoted {
		t.Errorf("ParseKeyValueWithMeta(unquoted) = %q, %q, %v", key, value, wasQuoted)
	}
}

func TestIsArrow(t *testing.T) {
	if !IsArrow("->") {
		t.Error("IsArrow(->) should be true")
	}
	if !IsArrow("<->") {
		t.Error("IsArrow(<->) should be true")
	}
	if !IsArrow("--") {
		t.Error("IsArrow(--) should be true")
	}
	if IsArrow("=>") {
		t.Error("IsArrow(=>) should be false")
	}
	if IsArrow("add") {
		t.Error("IsArrow(add) should be false")
	}
}

func TestIsSelector(t *testing.T) {
	if !IsSelector("@type:db") {
		t.Error("IsSelector(@type:db) should be true")
	}
	if !IsSelector("@all") {
		t.Error("IsSelector(@all) should be true")
	}
	if IsSelector("type:db") {
		t.Error("IsSelector(type:db) should be false")
	}
	if IsSelector("add") {
		t.Error("IsSelector(add) should be false")
	}
}
