package fcpcore

import "strings"

// VerbSpec defines a single verb in an FCP protocol.
type VerbSpec struct {
	Name     string
	Syntax   string
	Category string
}

// VerbRegistry is a registry of verb specifications that supports
// lookup by verb name and reference card generation grouped by category.
type VerbRegistry struct {
	specs      map[string]VerbSpec
	categories []categoryGroup
	catIndex   map[string]int
}

type categoryGroup struct {
	name  string
	specs []VerbSpec
}

// NewVerbRegistry creates a new empty VerbRegistry.
func NewVerbRegistry() *VerbRegistry {
	return &VerbRegistry{
		specs:    make(map[string]VerbSpec),
		catIndex: make(map[string]int),
	}
}

// Register adds a single verb specification.
func (r *VerbRegistry) Register(spec VerbSpec) {
	r.specs[spec.Name] = spec
	if idx, ok := r.catIndex[spec.Category]; ok {
		r.categories[idx].specs = append(r.categories[idx].specs, spec)
	} else {
		r.catIndex[spec.Category] = len(r.categories)
		r.categories = append(r.categories, categoryGroup{
			name:  spec.Category,
			specs: []VerbSpec{spec},
		})
	}
}

// RegisterMany adds multiple verb specifications at once.
func (r *VerbRegistry) RegisterMany(specs []VerbSpec) {
	for _, spec := range specs {
		r.Register(spec)
	}
}

// Lookup finds a verb specification by name.
// Returns the spec and true if found, or zero value and false if not.
func (r *VerbRegistry) Lookup(name string) (VerbSpec, bool) {
	spec, ok := r.specs[name]
	return spec, ok
}

// Verbs returns all registered verb specifications.
func (r *VerbRegistry) Verbs() []VerbSpec {
	result := make([]VerbSpec, 0, len(r.specs))
	for _, spec := range r.specs {
		result = append(result, spec)
	}
	return result
}

// GenerateReferenceCard generates a reference card string grouped by category.
// Optional extraSections adds extra static sections appended after the verb listing.
func (r *VerbRegistry) GenerateReferenceCard(extraSections map[string]string) string {
	var lines []string

	for _, cat := range r.categories {
		lines = append(lines, strings.ToUpper(cat.name)+":")
		for _, spec := range cat.specs {
			lines = append(lines, "  "+spec.Syntax)
		}
		lines = append(lines, "")
	}

	if extraSections != nil {
		for title, content := range extraSections {
			lines = append(lines, strings.ToUpper(title)+":")
			lines = append(lines, content)
			lines = append(lines, "")
		}
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}
