package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

func makeRef(kind, fullType, label string) *BlockRef {
	return &BlockRef{
		Kind:     kind,
		FullType: fullType,
		Label:    label,
		Provider: DeriveProvider(fullType),
		Block:    hclwrite.NewBlock("resource", []string{fullType, label}),
		Tags:     map[string]string{},
	}
}

func TestIndex_Add_And_Get(t *testing.T) {
	idx := NewIndex()
	ref := makeRef("resource", "aws_instance", "web")
	if err := idx.Add(ref); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got := idx.Get("web")
	if got != ref {
		t.Error("Get(web) did not return the added ref")
	}
}

func TestIndex_Add_Duplicate(t *testing.T) {
	idx := NewIndex()
	ref1 := makeRef("resource", "aws_instance", "web")
	ref2 := makeRef("resource", "aws_instance", "web")
	if err := idx.Add(ref1); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	if err := idx.Add(ref2); err == nil {
		t.Error("expected error for duplicate label+type")
	}
}

func TestIndex_Add_SameLabel_DifferentType(t *testing.T) {
	idx := NewIndex()
	ref1 := makeRef("resource", "aws_instance", "main")
	ref2 := makeRef("resource", "aws_vpc", "main")
	if err := idx.Add(ref1); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	// Different type, same label — currently overwrites (simple index)
	if err := idx.Add(ref2); err != nil {
		t.Fatalf("Add second (different type): %v", err)
	}
}

func TestIndex_Remove(t *testing.T) {
	idx := NewIndex()
	ref := makeRef("resource", "aws_instance", "web")
	idx.Add(ref)
	removed := idx.Remove("web")
	if removed != ref {
		t.Error("Remove did not return the ref")
	}
	if idx.Get("web") != nil {
		t.Error("Get(web) should return nil after remove")
	}
	if len(idx.Order) != 0 {
		t.Errorf("Order should be empty after remove, got %v", idx.Order)
	}
}

func TestIndex_Remove_NotFound(t *testing.T) {
	idx := NewIndex()
	if idx.Remove("nonexistent") != nil {
		t.Error("Remove(nonexistent) should return nil")
	}
}

func TestIndex_GetByQualified(t *testing.T) {
	idx := NewIndex()
	ref := makeRef("resource", "aws_vpc", "main")
	idx.Add(ref)
	got := idx.GetByQualified("aws_vpc.main")
	if got != ref {
		t.Error("GetByQualified(aws_vpc.main) did not return the ref")
	}
}

func TestIndex_GetByQualified_NotFound(t *testing.T) {
	idx := NewIndex()
	ref := makeRef("resource", "aws_vpc", "main")
	idx.Add(ref)
	got := idx.GetByQualified("aws_instance.main")
	if got != nil {
		t.Error("GetByQualified should return nil for wrong type")
	}
}

func TestIndex_FindByType(t *testing.T) {
	idx := NewIndex()
	idx.Add(makeRef("resource", "aws_instance", "web"))
	idx.Add(makeRef("resource", "aws_instance", "api"))
	idx.Add(makeRef("resource", "aws_vpc", "main"))

	got := idx.FindByType("aws_instance")
	if len(got) != 2 {
		t.Errorf("FindByType(aws_instance) = %d refs, want 2", len(got))
	}
}

func TestIndex_FindByKind(t *testing.T) {
	idx := NewIndex()
	idx.Add(makeRef("resource", "aws_instance", "web"))
	idx.Add(&BlockRef{Kind: "variable", FullType: "variable", Label: "region", Tags: map[string]string{}})
	idx.Add(&BlockRef{Kind: "variable", FullType: "variable", Label: "env", Tags: map[string]string{}})

	got := idx.FindByKind("variable")
	if len(got) != 2 {
		t.Errorf("FindByKind(variable) = %d refs, want 2", len(got))
	}
}

func TestIndex_FindByProvider(t *testing.T) {
	idx := NewIndex()
	idx.Add(makeRef("resource", "aws_instance", "web"))
	idx.Add(makeRef("resource", "aws_vpc", "main"))
	idx.Add(makeRef("resource", "google_compute_instance", "gce"))

	got := idx.FindByProvider("aws")
	if len(got) != 2 {
		t.Errorf("FindByProvider(aws) = %d refs, want 2", len(got))
	}
}

func TestIndex_FindByTag(t *testing.T) {
	idx := NewIndex()
	ref1 := makeRef("resource", "aws_instance", "web")
	ref1.Tags["env"] = "prod"
	ref2 := makeRef("resource", "aws_instance", "api")
	ref2.Tags["env"] = "staging"
	idx.Add(ref1)
	idx.Add(ref2)

	got := idx.FindByTag("env", "prod")
	if len(got) != 1 {
		t.Errorf("FindByTag(env, prod) = %d refs, want 1", len(got))
	}
	if got[0].Label != "web" {
		t.Errorf("FindByTag returned %q, want web", got[0].Label)
	}

	// Match any value for key
	all := idx.FindByTag("env", "")
	if len(all) != 2 {
		t.Errorf("FindByTag(env, '') = %d refs, want 2", len(all))
	}
}

func TestIndex_Order(t *testing.T) {
	idx := NewIndex()
	idx.Add(makeRef("resource", "aws_vpc", "vpc"))
	idx.Add(makeRef("resource", "aws_subnet", "subnet"))
	idx.Add(makeRef("resource", "aws_instance", "web"))

	if len(idx.Order) != 3 {
		t.Fatalf("Order len = %d, want 3", len(idx.Order))
	}
	if idx.Order[0] != "vpc" || idx.Order[1] != "subnet" || idx.Order[2] != "web" {
		t.Errorf("Order = %v", idx.Order)
	}
}

func TestIndex_Rebuild(t *testing.T) {
	f := hclwrite.NewEmptyFile()
	body := f.Body()
	body.AppendNewBlock("resource", []string{"aws_instance", "web"})
	body.AppendNewline()
	body.AppendNewBlock("variable", []string{"region"})

	idx := NewIndex()
	idx.Rebuild(f)

	if idx.Get("web") == nil {
		t.Error("expected web in index after rebuild")
	}
	if idx.Get("region") == nil {
		t.Error("expected region in index after rebuild")
	}
	if len(idx.Order) != 2 {
		t.Errorf("Order len = %d, want 2", len(idx.Order))
	}
}

func TestIndex_Connections(t *testing.T) {
	idx := NewIndex()
	idx.Add(makeRef("resource", "aws_instance", "web"))
	idx.Add(makeRef("resource", "aws_vpc", "vpc"))

	idx.Connections["web"] = map[string]string{"vpc": "depends"}

	if idx.Connections["web"]["vpc"] != "depends" {
		t.Error("connection not stored")
	}

	// Remove web should clean up connections
	idx.Remove("web")
	if _, ok := idx.Connections["web"]; ok {
		t.Error("connections should be removed with block")
	}
}

func TestDeriveProvider(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"aws_instance", "aws"},
		{"google_compute_instance", "google"},
		{"azurerm_resource_group", "azurerm"},
		{"random_id", "random"},
		{"kubernetes", "kubernetes"},
	}
	for _, tt := range tests {
		got := DeriveProvider(tt.input)
		if got != tt.want {
			t.Errorf("DeriveProvider(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
