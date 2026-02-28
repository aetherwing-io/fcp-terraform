package fcpcore

import (
	"strings"
	"testing"
)

func TestVerbRegistry_RegisterAndLookup(t *testing.T) {
	reg := NewVerbRegistry()
	spec := VerbSpec{Name: "add", Syntax: "add TYPE LABEL", Category: "create"}
	reg.Register(spec)
	got, ok := reg.Lookup("add")
	if !ok {
		t.Fatal("expected to find 'add'")
	}
	if got.Name != "add" || got.Syntax != "add TYPE LABEL" || got.Category != "create" {
		t.Errorf("Lookup(add) = %+v", got)
	}
}

func TestVerbRegistry_LookupUnknown(t *testing.T) {
	reg := NewVerbRegistry()
	_, ok := reg.Lookup("nonexistent")
	if ok {
		t.Error("expected Lookup to return false for unknown verb")
	}
}

func TestVerbRegistry_RegisterMany(t *testing.T) {
	reg := createTestRegistry()
	for _, name := range []string{"add", "remove", "connect", "style"} {
		_, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("expected to find %q", name)
		}
	}
}

func TestVerbRegistry_Verbs(t *testing.T) {
	reg := createTestRegistry()
	verbs := reg.Verbs()
	if len(verbs) != 4 {
		t.Errorf("len(Verbs) = %d, want 4", len(verbs))
	}
	names := make(map[string]bool)
	for _, v := range verbs {
		names[v.Name] = true
	}
	for _, name := range []string{"add", "remove", "connect", "style"} {
		if !names[name] {
			t.Errorf("missing verb %q", name)
		}
	}
}

func TestVerbRegistry_ReferenceCard(t *testing.T) {
	reg := createTestRegistry()
	card := reg.GenerateReferenceCard(nil)
	if !strings.Contains(card, "CREATE:") {
		t.Error("card missing CREATE:")
	}
	if !strings.Contains(card, "MODIFY:") {
		t.Error("card missing MODIFY:")
	}
	if !strings.Contains(card, "  add TYPE LABEL [key:value]") {
		t.Error("card missing add syntax")
	}
	if !strings.Contains(card, "  connect SRC -> TGT") {
		t.Error("card missing connect syntax")
	}
	if !strings.Contains(card, "  remove SELECTOR") {
		t.Error("card missing remove syntax")
	}
	if !strings.Contains(card, "  style REF [fill:#HEX]") {
		t.Error("card missing style syntax")
	}
}

func TestVerbRegistry_ReferenceCard_ExtraSections(t *testing.T) {
	reg := createTestRegistry()
	card := reg.GenerateReferenceCard(map[string]string{
		"Themes": "  blue  #dae8fc\n  red   #f8cecc",
	})
	if !strings.Contains(card, "THEMES:") {
		t.Error("card missing THEMES:")
	}
	if !strings.Contains(card, "  blue  #dae8fc") {
		t.Error("card missing theme content")
	}
}

func TestVerbRegistry_ReferenceCard_Empty(t *testing.T) {
	reg := NewVerbRegistry()
	card := reg.GenerateReferenceCard(nil)
	if card != "" {
		t.Errorf("expected empty card, got %q", card)
	}
}

func createTestRegistry() *VerbRegistry {
	reg := NewVerbRegistry()
	reg.RegisterMany([]VerbSpec{
		{Name: "add", Syntax: "add TYPE LABEL [key:value]", Category: "create"},
		{Name: "remove", Syntax: "remove SELECTOR", Category: "modify"},
		{Name: "connect", Syntax: "connect SRC -> TGT", Category: "create"},
		{Name: "style", Syntax: "style REF [fill:#HEX]", Category: "modify"},
	})
	return reg
}
