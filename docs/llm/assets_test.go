package llmassets

import (
	"encoding/json"
	"reflect"
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
