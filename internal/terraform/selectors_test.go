package terraform

import (
	"testing"
)

func TestParseSelector_Type(t *testing.T) {
	sel := ParseSelector("@type:aws_instance")
	if sel.Type != "type" || sel.Value != "aws_instance" || sel.Negated {
		t.Errorf("ParseSelector(@type:aws_instance) = %+v", sel)
	}
}

func TestParseSelector_Provider(t *testing.T) {
	sel := ParseSelector("@provider:aws")
	if sel.Type != "provider" || sel.Value != "aws" {
		t.Errorf("ParseSelector(@provider:aws) = %+v", sel)
	}
}

func TestParseSelector_Kind(t *testing.T) {
	sel := ParseSelector("@kind:resource")
	if sel.Type != "kind" || sel.Value != "resource" {
		t.Errorf("ParseSelector(@kind:resource) = %+v", sel)
	}
}

func TestParseSelector_Tag_KeyVal(t *testing.T) {
	sel := ParseSelector("@tag:env=prod")
	if sel.Type != "tag" || sel.Value != "env=prod" {
		t.Errorf("ParseSelector(@tag:env=prod) = %+v", sel)
	}
}

func TestParseSelector_Tag_KeyOnly(t *testing.T) {
	sel := ParseSelector("@tag:env")
	if sel.Type != "tag" || sel.Value != "env" {
		t.Errorf("ParseSelector(@tag:env) = %+v", sel)
	}
}

func TestParseSelector_All(t *testing.T) {
	sel := ParseSelector("@all")
	if sel.Type != "all" || sel.Value != "" {
		t.Errorf("ParseSelector(@all) = %+v", sel)
	}
}

func TestParseSelector_Negated(t *testing.T) {
	sel := ParseSelector("@not:type:aws_instance")
	if sel.Type != "type" || sel.Value != "aws_instance" || !sel.Negated {
		t.Errorf("ParseSelector(@not:type:aws_instance) = %+v", sel)
	}
}

func setupSelectorModel() *TerraformModel {
	m := NewModel("test")
	m.AddResource("aws_instance", "web", nil, nil)
	m.AddResource("aws_instance", "api", nil, nil)
	m.AddResource("aws_vpc", "main", nil, nil)
	m.AddResource("google_compute_instance", "gce", nil, nil)
	m.AddVariable("region", map[string]string{"type": "string"})

	// Add tags
	m.Index.Get("web").Tags["env"] = "prod"
	m.Index.Get("api").Tags["env"] = "staging"
	m.Index.Get("main").Tags["env"] = "prod"

	return m
}

func TestResolveSelector_Type(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@type:aws_instance")
	result := ResolveSelector(sel, m)
	if len(result) != 2 {
		t.Errorf("@type:aws_instance = %d refs, want 2", len(result))
	}
}

func TestResolveSelector_Provider(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@provider:aws")
	result := ResolveSelector(sel, m)
	if len(result) != 3 {
		t.Errorf("@provider:aws = %d refs, want 3", len(result))
	}
}

func TestResolveSelector_Kind(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@kind:resource")
	result := ResolveSelector(sel, m)
	if len(result) != 4 {
		t.Errorf("@kind:resource = %d refs, want 4", len(result))
	}
}

func TestResolveSelector_Kind_Variable(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@kind:variable")
	result := ResolveSelector(sel, m)
	if len(result) != 1 {
		t.Errorf("@kind:variable = %d refs, want 1", len(result))
	}
}

func TestResolveSelector_All(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@all")
	result := ResolveSelector(sel, m)
	if len(result) != 5 {
		t.Errorf("@all = %d refs, want 5", len(result))
	}
}

func TestResolveSelector_Tag_KeyVal(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@tag:env=prod")
	result := ResolveSelector(sel, m)
	if len(result) != 2 {
		t.Errorf("@tag:env=prod = %d refs, want 2", len(result))
	}
}

func TestResolveSelector_Tag_KeyOnly(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@tag:env")
	result := ResolveSelector(sel, m)
	if len(result) != 3 {
		t.Errorf("@tag:env = %d refs, want 3", len(result))
	}
}

func TestResolveSelector_Negated(t *testing.T) {
	m := setupSelectorModel()
	sel := ParseSelector("@not:type:aws_instance")
	result := ResolveSelector(sel, m)
	// Total 5 blocks, 2 are aws_instance, so negated = 3
	if len(result) != 3 {
		t.Errorf("@not:type:aws_instance = %d refs, want 3", len(result))
	}
}

func TestResolveSelectorSet_Intersection(t *testing.T) {
	m := setupSelectorModel()
	// aws provider AND tag env=prod → web and main (not api since staging)
	result := ResolveSelectorSet([]string{"@provider:aws", "@tag:env=prod"}, m)
	if len(result) != 2 {
		t.Errorf("intersection = %d refs, want 2", len(result))
	}
}

func TestResolveSelectorSet_TypeAndTag(t *testing.T) {
	m := setupSelectorModel()
	// aws_instance AND env=prod → only web
	result := ResolveSelectorSet([]string{"@type:aws_instance", "@tag:env=prod"}, m)
	if len(result) != 1 {
		t.Errorf("intersection = %d refs, want 1", len(result))
	}
	if result[0].Label != "web" {
		t.Errorf("expected web, got %q", result[0].Label)
	}
}
