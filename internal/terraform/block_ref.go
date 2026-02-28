package terraform

import (
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

// BlockRef is a thin index entry pointing into hclwrite.File.
type BlockRef struct {
	Label    string            // user-facing label (e.g., "web_server")
	Kind     string            // "resource", "data", "variable", "output", "module", "provider"
	FullType string            // e.g., "aws_instance", "aws_vpc"
	Provider string            // derived from type prefix ("aws", "google", "azurerm")
	Block    *hclwrite.Block   // pointer INTO hclwrite.File
	Tags     map[string]string // for @tag: selector
}

// QualifiedName returns "fullType.label" for resource/data blocks, or just the label.
func (ref *BlockRef) QualifiedName() string {
	if ref.Kind == "resource" || ref.Kind == "data" {
		return ref.FullType + "." + ref.Label
	}
	return ref.Label
}

// DeriveProvider extracts the provider name from a Terraform resource type.
// "aws_s3_bucket" -> "aws", "google_compute_instance" -> "google", "azurerm_resource_group" -> "azurerm"
func DeriveProvider(fullType string) string {
	idx := strings.Index(fullType, "_")
	if idx > 0 {
		return fullType[:idx]
	}
	return fullType
}
