package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Index provides O(1) label lookup and multi-key indexing for BlockRefs.
type Index struct {
	ByLabel     map[string]*BlockRef           // label -> ref (only if unambiguous)
	ByQualified map[string]*BlockRef           // "fulltype.label" -> ref (always populated)
	Ambiguous   map[string]bool                // labels used by multiple types
	ByType      map[string][]*BlockRef         // fullType -> refs
	ByKind      map[string][]*BlockRef         // kind -> refs
	ByProvider  map[string][]*BlockRef         // provider -> refs
	Order       []string                       // insertion order of labels
	Connections map[string]map[string]string   // src_label -> dst_label -> edge_label
}

// NewIndex creates a new empty Index.
func NewIndex() *Index {
	return &Index{
		ByLabel:     make(map[string]*BlockRef),
		ByQualified: make(map[string]*BlockRef),
		Ambiguous:   make(map[string]bool),
		ByType:      make(map[string][]*BlockRef),
		ByKind:      make(map[string][]*BlockRef),
		ByProvider:  make(map[string][]*BlockRef),
		Connections: make(map[string]map[string]string),
	}
}

// qualifiedKey returns the qualified map key for a BlockRef: "fulltype.label" lowercased.
func qualifiedKey(ref *BlockRef) string {
	return strings.ToLower(ref.FullType + "." + ref.Label)
}

// Add adds a BlockRef to the index. Returns an error if the exact qualified name already exists.
func (idx *Index) Add(ref *BlockRef) error {
	key := strings.ToLower(ref.Label)
	qKey := qualifiedKey(ref)

	// Check for duplicate: same fullType AND same label
	if _, ok := idx.ByQualified[qKey]; ok {
		return fmt.Errorf("%s %q already exists", ref.FullType, ref.Label)
	}

	// Always add to ByQualified
	idx.ByQualified[qKey] = ref

	// Handle ByLabel: if label is already taken by a DIFFERENT type, mark as ambiguous
	if existing, ok := idx.ByLabel[key]; ok {
		if existing.FullType != ref.FullType {
			// Different type using same label -> ambiguous
			idx.Ambiguous[key] = true
			delete(idx.ByLabel, key)
		}
	} else if idx.Ambiguous[key] {
		// Already ambiguous, don't add to ByLabel
	} else {
		// First use of this label
		idx.ByLabel[key] = ref
	}

	idx.ByType[ref.FullType] = append(idx.ByType[ref.FullType], ref)
	idx.ByKind[ref.Kind] = append(idx.ByKind[ref.Kind], ref)
	if ref.Provider != "" {
		idx.ByProvider[ref.Provider] = append(idx.ByProvider[ref.Provider], ref)
	}
	idx.Order = append(idx.Order, ref.Label)
	return nil
}

// Remove removes a block by label and returns it, or nil if not found.
func (idx *Index) Remove(label string) *BlockRef {
	key := strings.ToLower(label)

	// Try plain label first
	ref, ok := idx.ByLabel[key]
	if ok {
		qKey := qualifiedKey(ref)
		delete(idx.ByLabel, key)
		delete(idx.ByQualified, qKey)
	} else {
		// Try qualified lookup
		ref = idx.ByQualified[strings.ToLower(label)]
		if ref == nil {
			// Not found by qualified either, try to find in ByQualified by scanning for this label
			// (used when the label is ambiguous)
			for qk, r := range idx.ByQualified {
				if strings.ToLower(r.Label) == key {
					ref = r
					delete(idx.ByQualified, qk)
					break
				}
			}
		} else {
			delete(idx.ByQualified, strings.ToLower(label))
		}
		if ref == nil {
			return nil
		}
		key = strings.ToLower(ref.Label)
	}

	// If the label was ambiguous, check if it's still ambiguous after removal
	if idx.Ambiguous[key] {
		// Count how many different types still use this label
		typesSeen := make(map[string]bool)
		var lastRef *BlockRef
		for _, r := range idx.ByQualified {
			if strings.ToLower(r.Label) == key {
				typesSeen[r.FullType] = true
				lastRef = r
			}
		}
		if len(typesSeen) <= 1 {
			delete(idx.Ambiguous, key)
			if lastRef != nil {
				idx.ByLabel[key] = lastRef
			}
		}
	}

	idx.ByType[ref.FullType] = removeFromSlice(idx.ByType[ref.FullType], ref)
	idx.ByKind[ref.Kind] = removeFromSlice(idx.ByKind[ref.Kind], ref)
	if ref.Provider != "" {
		idx.ByProvider[ref.Provider] = removeFromSlice(idx.ByProvider[ref.Provider], ref)
	}

	// Remove from order
	for i, l := range idx.Order {
		if strings.EqualFold(l, ref.Label) {
			idx.Order = append(idx.Order[:i], idx.Order[i+1:]...)
			break
		}
	}

	// Remove connections involving this block
	delete(idx.Connections, key)
	for src, targets := range idx.Connections {
		delete(targets, key)
		if len(targets) == 0 {
			delete(idx.Connections, src)
		}
	}

	return ref
}

// Get returns a BlockRef by label, or nil if not found or ambiguous.
func (idx *Index) Get(label string) *BlockRef {
	key := strings.ToLower(label)
	if idx.Ambiguous[key] {
		return nil // force qualified lookup
	}
	return idx.ByLabel[key]
}

// GetByQualified finds a block by "fullType.label" format (e.g., "aws_vpc.main").
func (idx *Index) GetByQualified(input string) *BlockRef {
	return idx.ByQualified[strings.ToLower(input)]
}

// FindByType returns all blocks of a given type.
func (idx *Index) FindByType(fullType string) []*BlockRef {
	return idx.ByType[fullType]
}

// FindByKind returns all blocks of a given kind (resource, data, variable, etc.).
func (idx *Index) FindByKind(kind string) []*BlockRef {
	return idx.ByKind[kind]
}

// FindByProvider returns all blocks from a given provider.
func (idx *Index) FindByProvider(provider string) []*BlockRef {
	return idx.ByProvider[provider]
}

// FindByTag returns all blocks matching a tag key and optionally a value.
// If value is empty, matches any block with that tag key.
func (idx *Index) FindByTag(key, value string) []*BlockRef {
	var result []*BlockRef
	for _, ref := range idx.ByQualified {
		if ref.Tags == nil {
			continue
		}
		if tagVal, ok := ref.Tags[key]; ok {
			if value == "" || tagVal == value {
				result = append(result, ref)
			}
		}
	}
	return result
}

// Rebuild rescans an hclwrite.File and repopulates the index.
func (idx *Index) Rebuild(file *hclwrite.File) {
	// Clear all maps
	idx.ByLabel = make(map[string]*BlockRef)
	idx.ByQualified = make(map[string]*BlockRef)
	idx.Ambiguous = make(map[string]bool)
	idx.ByType = make(map[string][]*BlockRef)
	idx.ByKind = make(map[string][]*BlockRef)
	idx.ByProvider = make(map[string][]*BlockRef)
	idx.Order = nil
	// Preserve connections — they are not stored in HCL

	for _, block := range file.Body().Blocks() {
		ref := blockToRef(block)
		if ref != nil {
			// Use Add() to properly handle ambiguity
			idx.Add(ref)
		}
	}
}

// blockToRef converts an hclwrite.Block to a BlockRef.
func blockToRef(block *hclwrite.Block) *BlockRef {
	blockType := block.Type()
	labels := block.Labels()

	ref := &BlockRef{
		Block: block,
		Tags:  make(map[string]string),
	}

	switch blockType {
	case "resource":
		if len(labels) < 2 {
			return nil
		}
		ref.Kind = "resource"
		ref.FullType = labels[0]
		ref.Label = labels[1]
		ref.Provider = DeriveProvider(labels[0])
	case "data":
		if len(labels) < 2 {
			return nil
		}
		ref.Kind = "data"
		ref.FullType = labels[0]
		ref.Label = labels[1]
		ref.Provider = DeriveProvider(labels[0])
	case "variable":
		if len(labels) < 1 {
			return nil
		}
		ref.Kind = "variable"
		ref.FullType = "variable"
		ref.Label = labels[0]
	case "output":
		if len(labels) < 1 {
			return nil
		}
		ref.Kind = "output"
		ref.FullType = "output"
		ref.Label = labels[0]
	case "provider":
		if len(labels) < 1 {
			return nil
		}
		ref.Kind = "provider"
		ref.FullType = labels[0]
		ref.Label = labels[0]
		ref.Provider = labels[0]
	case "module":
		if len(labels) < 1 {
			return nil
		}
		ref.Kind = "module"
		ref.FullType = "module"
		ref.Label = labels[0]
	default:
		return nil
	}

	return ref
}

// removeFromSlice removes a specific BlockRef from a slice.
func removeFromSlice(slice []*BlockRef, ref *BlockRef) []*BlockRef {
	for i, r := range slice {
		if r == ref {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
