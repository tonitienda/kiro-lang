package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	llmassets "github.com/kiro-lang/kiro/docs/llm"
)

func TestDescribeBundledLanguageReference(t *testing.T) {
	metadata, err := llmassets.Metadata()
	if err != nil {
		t.Fatal(err)
	}
	content, err := llmassets.Files.ReadFile(metadata.Reference.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"describe", "--json"}, {"describe", "--reference", "--json"}} {
		report, err := captureTool(t, args...)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(report.Data)
		if err != nil {
			t.Fatal(err)
		}
		var data struct {
			GuideVersion string                      `json:"guide_version"`
			Reference    llmassets.ReferenceMetadata `json:"language_reference"`
			Content      string                      `json:"content"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			t.Fatal(err)
		}
		if data.GuideVersion != metadata.GuideVersion || data.Reference != metadata.Reference {
			t.Fatalf("unidentified reference: %+v", data)
		}
		if len(args) == 3 && data.Content != string(content) {
			t.Fatal("describe did not return the exact bundled reference")
		}
	}
	for _, args := range [][]string{{"describe", "json.decode", "--reference", "--json"}, {"describe", "--reference", "--reference", "--json"}} {
		report, err := captureTool(t, args...)
		if err == nil || report.OK || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "KIRO_USAGE" {
			t.Fatalf("invalid reference request: %+v %v", report, err)
		}
	}
	// Exercise ordinary text output and a real scaffold through the native binary.
	binary := buildKiroBinary(t)
	output := referenceCommandOutput(t, "", binary, "describe", "--reference")
	if output != string(content) {
		t.Fatal("text reference differs from bundle")
	}
	root := t.TempDir()
	referenceCommandOutput(t, root, binary, "new", "hello")
	raw, err := os.ReadFile(filepath.Join(root, "hello", ".kiro", "version.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		GuideVersion string                      `json:"guide_version"`
		Reference    llmassets.ReferenceMetadata `json:"language_reference"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if saved.GuideVersion != metadata.GuideVersion || saved.Reference.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("wrong scaffold identity: %+v", saved)
	}
	agents, err := os.ReadFile(filepath.Join(root, "hello", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "references/language.md") {
		t.Fatal("scaffold did not direct agent to the language reference")
	}
}

func referenceCommandOutput(t *testing.T, dir, binary string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kiro %v: %v\n%s", args, err, output)
	}
	return string(output)
}
