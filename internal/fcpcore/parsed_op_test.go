package fcpcore

import (
	"testing"
)

func TestParseOp_SimpleVerb(t *testing.T) {
	r := ParseOp("add svc AuthService")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "add" {
		t.Errorf("verb = %q, want %q", r.Op.Verb, "add")
	}
	if !sliceEqual(r.Op.Positionals, []string{"svc", "AuthService"}) {
		t.Errorf("positionals = %v, want [svc AuthService]", r.Op.Positionals)
	}
	if len(r.Op.Params) != 0 {
		t.Errorf("params = %v, want empty", r.Op.Params)
	}
	if len(r.Op.Selectors) != 0 {
		t.Errorf("selectors = %v, want empty", r.Op.Selectors)
	}
}

func TestParseOp_KeyValueParams(t *testing.T) {
	r := ParseOp("add svc AuthService theme:blue near:Gateway")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "add" {
		t.Errorf("verb = %q, want %q", r.Op.Verb, "add")
	}
	if !sliceEqual(r.Op.Positionals, []string{"svc", "AuthService"}) {
		t.Errorf("positionals = %v", r.Op.Positionals)
	}
	if r.Op.Params["theme"] != "blue" {
		t.Errorf("params[theme] = %q, want %q", r.Op.Params["theme"], "blue")
	}
	if r.Op.Params["near"] != "Gateway" {
		t.Errorf("params[near] = %q, want %q", r.Op.Params["near"], "Gateway")
	}
}

func TestParseOp_Selectors(t *testing.T) {
	r := ParseOp("remove @type:db @recent:3")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "remove" {
		t.Errorf("verb = %q", r.Op.Verb)
	}
	if !sliceEqual(r.Op.Selectors, []string{"@type:db", "@recent:3"}) {
		t.Errorf("selectors = %v", r.Op.Selectors)
	}
	if len(r.Op.Positionals) != 0 {
		t.Errorf("positionals = %v, want empty", r.Op.Positionals)
	}
}

func TestParseOp_MixedTypes(t *testing.T) {
	r := ParseOp("style @type:svc fill:#ff0000 bold")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "style" {
		t.Errorf("verb = %q", r.Op.Verb)
	}
	if !sliceEqual(r.Op.Selectors, []string{"@type:svc"}) {
		t.Errorf("selectors = %v", r.Op.Selectors)
	}
	if r.Op.Params["fill"] != "#ff0000" {
		t.Errorf("params[fill] = %q", r.Op.Params["fill"])
	}
	if !sliceEqual(r.Op.Positionals, []string{"bold"}) {
		t.Errorf("positionals = %v", r.Op.Positionals)
	}
}

func TestParseOp_LowercasesVerb(t *testing.T) {
	r := ParseOp("ADD svc Test")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "add" {
		t.Errorf("verb = %q, want %q", r.Op.Verb, "add")
	}
}

func TestParseOp_PreservesRaw(t *testing.T) {
	r := ParseOp("  add svc Test  ")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Raw != "add svc Test" {
		t.Errorf("raw = %q, want %q", r.Op.Raw, "add svc Test")
	}
}

func TestParseOp_QuotedPositionals(t *testing.T) {
	r := ParseOp(`label Gateway "API Gateway v2"`)
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if !sliceEqual(r.Op.Positionals, []string{"Gateway", "API Gateway v2"}) {
		t.Errorf("positionals = %v", r.Op.Positionals)
	}
}

func TestParseOp_EmptyInput(t *testing.T) {
	r := ParseOp("")
	if !r.IsError() {
		t.Fatal("expected error for empty input")
	}
	if r.Err.Error != "empty operation" {
		t.Errorf("error = %q, want %q", r.Err.Error, "empty operation")
	}
}

func TestParseOp_WhitespaceOnly(t *testing.T) {
	r := ParseOp("   ")
	if !r.IsError() {
		t.Fatal("expected error for whitespace-only input")
	}
}

func TestParseOp_ArrowsAsPositionals(t *testing.T) {
	r := ParseOp("connect A -> B")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if !sliceEqual(r.Op.Positionals, []string{"A", "->", "B"}) {
		t.Errorf("positionals = %v, want [A -> B]", r.Op.Positionals)
	}
}

func TestParseOp_VerbOnly(t *testing.T) {
	r := ParseOp("undo")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "undo" {
		t.Errorf("verb = %q", r.Op.Verb)
	}
	if len(r.Op.Positionals) != 0 {
		t.Errorf("positionals = %v, want empty", r.Op.Positionals)
	}
	if len(r.Op.Params) != 0 {
		t.Errorf("params = %v, want empty", r.Op.Params)
	}
	if len(r.Op.Selectors) != 0 {
		t.Errorf("selectors = %v, want empty", r.Op.Selectors)
	}
}

func TestParseOp_TerraformStyle(t *testing.T) {
	r := ParseOp(`add resource aws_instance web ami:ami-123 instance_type:t2.micro`)
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "add" {
		t.Errorf("verb = %q", r.Op.Verb)
	}
	if !sliceEqual(r.Op.Positionals, []string{"resource", "aws_instance", "web"}) {
		t.Errorf("positionals = %v", r.Op.Positionals)
	}
	if r.Op.Params["ami"] != "ami-123" {
		t.Errorf("params[ami] = %q", r.Op.Params["ami"])
	}
	if r.Op.Params["instance_type"] != "t2.micro" {
		t.Errorf("params[instance_type] = %q", r.Op.Params["instance_type"])
	}
}

func TestParseOp_SelectorWithNot(t *testing.T) {
	r := ParseOp("remove @type:aws_instance @tag:env=prod")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if !sliceEqual(r.Op.Selectors, []string{"@type:aws_instance", "@tag:env=prod"}) {
		t.Errorf("selectors = %v", r.Op.Selectors)
	}
}

func TestParseOp_ConnectWithArrowAndParams(t *testing.T) {
	r := ParseOp("connect AuthService -> UserDB label:queries")
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "connect" {
		t.Errorf("verb = %q", r.Op.Verb)
	}
	if !sliceEqual(r.Op.Positionals, []string{"AuthService", "->", "UserDB"}) {
		t.Errorf("positionals = %v", r.Op.Positionals)
	}
	if r.Op.Params["label"] != "queries" {
		t.Errorf("params[label] = %q", r.Op.Params["label"])
	}
}

func TestParseOp_QuotedNew(t *testing.T) {
	r := ParseOp(`new "My Project"`)
	if r.IsError() {
		t.Fatalf("unexpected error: %s", r.Err.Error)
	}
	if r.Op.Verb != "new" {
		t.Errorf("verb = %q", r.Op.Verb)
	}
	if !sliceEqual(r.Op.Positionals, []string{"My Project"}) {
		t.Errorf("positionals = %v", r.Op.Positionals)
	}
}
