package llmassets

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kiro-lang/kiro/internal/buildsys/runtimekit"
	"github.com/kiro-lang/kiro/internal/project"
)

// Examples are read from the shipped document, never a second copy of its code.
func TestLanguageReferenceConformance(t *testing.T) {
	raw, err := Files.ReadFile("references/language.md")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	seen := map[string]bool{}
	outcomes := map[string]int{}
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "```ki") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(lines[i], "```"))
		if len(fields) == 0 || fields[0] != "ki" {
			t.Fatalf("invalid Kiro fence: %s", lines[i])
		}
		attrs := map[string]string{}
		for _, field := range fields[1:] {
			pair := strings.SplitN(field, "=", 2)
			if len(pair) != 2 || attrs[pair[0]] != "" {
				t.Fatalf("invalid metadata: %s", lines[i])
			}
			switch pair[0] {
			case "example", "expect", "diagnostic", "message":
			default:
				t.Fatalf("unknown metadata: %s", field)
			}
			attrs[pair[0]] = pair[1]
		}
		id, outcome := attrs["example"], attrs["expect"]
		if id == "" || seen[id] {
			t.Fatalf("missing or repeated example ID: %s", id)
		}
		seen[id] = true
		start := i + 1
		for i = start; i < len(lines) && lines[i] != "```"; i++ {
		}
		if i == len(lines) {
			t.Fatalf("unclosed fence: %s", id)
		}
		source := strings.Join(lines[start:i], "\n") + "\n"
		outcomes[outcome]++
		t.Run(id, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "main.ki"), []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			p, err := project.Load(root)
			if outcome == "reject" {
				if err == nil {
					t.Fatal("documented rejection was accepted")
				}
				code := attrs["diagnostic"]
				if code == "" {
					t.Fatal("rejection requires an explicit diagnostic")
				}
				d := runtimekit.DiagnosticOf(err, "KIRO_CHECK_FAILED")
				if d.Code != code {
					t.Fatalf("expected %s, got %s: %v", code, d.Code, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if outcome != "pass" && outcome != "runtime-error" {
				t.Fatalf("unknown outcome %s", outcome)
			}
			report, err := runtimekit.RunCoverage(p.RuntimeSpec(), 10000)
			if err != nil {
				t.Fatal(err)
			}
			for _, observation := range report.Tests {
				if observation.Status == "budget_exhausted" {
					t.Fatal("example exceeded execution budget")
				}
			}
			if outcome == "pass" {
				if !report.Passed || !report.Complete || len(report.Tests) == 0 {
					t.Fatalf("example did not pass: %+v", report.Tests)
				}
			} else {
				if report.Passed || !report.Complete || len(report.Tests) != 1 || attrs["message"] == "" || !strings.Contains(report.Tests[0].Error, attrs["message"]) {
					t.Fatalf("expected runtime error containing %q: %+v", attrs["message"], report.Tests)
				}
			}
		})
	}
	for _, outcome := range []string{"pass", "reject", "runtime-error"} {
		if outcomes[outcome] == 0 {
			t.Fatalf("missing %s examples", outcome)
		}
	}
	t.Logf("checked %d reference examples: %v", len(seen), outcomes)
}

func TestReferenceBuiltinContracts(t *testing.T) {
	raw, err := Files.ReadFile("references/language.md")
	if err != nil {
		t.Fatal(err)
	}
	table := strings.SplitN(string(raw), "<!-- builtin-contracts:start -->", 2)
	if len(table) != 2 {
		t.Fatal("missing builtin table")
	}
	rows := strings.Split(strings.SplitN(table[1], "<!-- builtin-contracts:end -->", 2)[0], "\n")
	actual := map[string]string{}
	for _, row := range rows {
		if !strings.HasPrefix(row, "| `") {
			continue
		}
		fields := strings.Split(row, "|")
		if len(fields) != 4 {
			t.Fatalf("invalid builtin row: %s", row)
		}
		name := strings.Trim(strings.TrimSpace(fields[1]), "`")
		if actual[name] != "" {
			t.Fatalf("repeated builtin %s", name)
		}
		actual[name] = strings.Trim(strings.TrimSpace(fields[2]), "`")
	}
	expected := map[string]string{}
	for name, signature := range runtimekit.BuiltinContracts() {
		expected[name] = signature.String()
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("reference API differs from executable registry\nactual: %v\nexpected: %v", actual, expected)
	}
}

func TestBundledMultiModuleExamplesCheckAndRun(t *testing.T) {
	for _, name := range []string{"typed_errors", "validated_results"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			prefix := "references/examples/" + name
			if err := copyExampleTree(prefix, root); err != nil {
				t.Fatal(err)
			}
			p, err := project.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			report, err := runtimekit.RunCoverage(p.RuntimeSpec(), 10000)
			if err != nil || !report.Passed || !report.Complete {
				t.Fatalf("%s: %+v %v", name, report.Tests, err)
			}
		})
	}
}

func copyExampleTree(prefix, dest string) error {
	entries, err := Files.ReadDir(prefix)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			if err := copyExampleTree(prefix+"/"+entry.Name(), path); err != nil {
				return err
			}
		} else {
			data, err := Files.ReadFile(prefix + "/" + entry.Name())
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				return fmt.Errorf("copy example: %w", err)
			}
		}
	}
	return nil
}

func TestLanguageReferenceIdentity(t *testing.T) {
	metadata, err := Metadata()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Files.ReadFile(metadata.Reference.Path)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Reference.SHA256 == "" || !strings.Contains(string(raw), "Guide revision: `"+metadata.GuideVersion+"`") || !strings.Contains(string(raw), "Reference schema: `"+metadata.Reference.Schema+"`") {
		t.Fatalf("reference identity disagrees with compact manifest: %+v", metadata)
	}
}
