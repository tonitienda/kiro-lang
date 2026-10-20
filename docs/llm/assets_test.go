package llmassets

import (
	"encoding/json"
	"github.com/kiro-lang/kiro/internal/project"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
)

func TestManifestCallSignaturesMatchRuntime(t *testing.T) {
	data, err := Files.ReadFile("references/kiro.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Calls map[string][]string `json:"call_signatures"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.Calls, runtimekit.BuiltinSignatures()) {
		t.Fatal("compact agent manifest call signatures differ from checker/runtime registry")
	}
}

func TestBundledGuideExamplesCheck(t *testing.T) {
	entries, err := Files.ReadDir("references/examples")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ki") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			root := t.TempDir()
			data, err := Files.ReadFile("references/examples/" + entry.Name())
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "main.ki"), data, 0644); err != nil {
				t.Fatal(err)
			}
			err = fs.WalkDir(Files, "references/examples/internal", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				dest := filepath.Join(root, strings.TrimPrefix(path, "references/examples/"))
				if d.IsDir() {
					return os.MkdirAll(dest, 0755)
				}
				data, err := Files.ReadFile(path)
				if err != nil {
					return err
				}
				return os.WriteFile(dest, data, 0644)
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := project.Load(root); err != nil {
				t.Fatal(err)
			}
		})
	}
}
