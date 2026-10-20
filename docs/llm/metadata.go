package llmassets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type ReferenceMetadata struct {
	Schema string `json:"schema"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type BundleMetadata struct {
	GuideVersion string            `json:"guide_version"`
	Reference    ReferenceMetadata `json:"language_reference"`
}

// Metadata identifies the exact reference bundled with this compiler, including
// development builds whose release version is only "dev".
func Metadata() (BundleMetadata, error) {
	raw, err := Files.ReadFile("references/kiro.json")
	if err != nil {
		return BundleMetadata{}, err
	}
	var manifest struct {
		Version   string            `json:"version"`
		Reference ReferenceMetadata `json:"language_reference"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return BundleMetadata{}, err
	}
	if manifest.Version == "" || manifest.Reference.Schema != "kiro.language-reference.v1" || manifest.Reference.Path != "references/language.md" {
		return BundleMetadata{}, fmt.Errorf("invalid bundled language reference metadata")
	}
	content, err := Files.ReadFile(manifest.Reference.Path)
	if err != nil {
		return BundleMetadata{}, err
	}
	sum := sha256.Sum256(content)
	manifest.Reference.SHA256 = hex.EncodeToString(sum[:])
	return BundleMetadata{GuideVersion: manifest.Version, Reference: manifest.Reference}, nil
}
