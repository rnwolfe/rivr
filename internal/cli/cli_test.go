package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(""), &out, &errb)
	return out.String(), errb.String(), code
}

func runStdin(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), errb.String(), code
}

func noColor(t *testing.T) { t.Helper(); t.Setenv("NO_COLOR", "1") }

func TestSearchJSON(t *testing.T) {
	noColor(t)
	out, _, code := run(t, "search", "usb-c cable", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", err, out)
	}
	if res["schemaVersion"] != "1" {
		t.Fatalf("missing/wrong schemaVersion: %v", res["schemaVersion"])
	}
	items, _ := res["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("want items, got none")
	}
}

func TestItemGetJSON(t *testing.T) {
	noColor(t)
	out, _, code := run(t, "item", "get", "B0TEST0001", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var it map[string]any
	if err := json.Unmarshal([]byte(out), &it); err != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", err, out)
	}
	if it["asin"] != "B0TEST0001" {
		t.Fatalf("asin mismatch: %v", it["asin"])
	}
}

func TestUntrustedFencingDefaultOn(t *testing.T) {
	noColor(t)
	out, _, code := run(t, "reviews", "B0TEST0001", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "‹untrusted›") {
		t.Fatalf("review text not fenced by default:\n%s", out)
	}
}

func TestUntrustedFencingCanBeDisabled(t *testing.T) {
	noColor(t)
	out, _, code := run(t, "reviews", "B0TEST0001", "--no-wrap-untrusted", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.Contains(out, "‹untrusted›") {
		t.Fatalf("--no-wrap-untrusted should disable fencing:\n%s", out)
	}
}

func TestAssociateTagDecoratesLinks(t *testing.T) {
	noColor(t)
	// Off by default: undecorated /dp/ links.
	out, _, _ := run(t, "search", "x", "--json")
	if strings.Contains(out, "tag=") {
		t.Fatalf("links should be undecorated by default:\n%s", out)
	}
	// With the tag: every product URL carries tag=<id>.
	out, _, code := run(t, "search", "x", "--associate-tag", "mytag-20", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "tag=mytag-20") {
		t.Fatalf("product links not decorated with associate tag:\n%s", out)
	}
}

func TestSelectProjection(t *testing.T) {
	noColor(t)
	out, _, code := run(t, "search", "thing", "--select", "query,provider", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if _, ok := res["items"]; ok {
		t.Fatalf("--select should drop unselected fields, got items: %s", out)
	}
}

func TestUnknownProvider(t *testing.T) {
	noColor(t)
	_, errb, code := run(t, "--provider", "nope", "search", "x", "--json")
	if code != 10 {
		t.Fatalf("exit = %d, want 10 (config)", code)
	}
	if !strings.Contains(errb, "PROVIDER_UNKNOWN") {
		t.Fatalf("missing PROVIDER_UNKNOWN: %s", errb)
	}
}

func TestReadOnlyGate(t *testing.T) {
	// amzn has no mutating command, but the gate must still default-deny.
	rt := &Runtime{Cfg: &CLI{}}
	err := rt.Guard("hypothetical mutation")
	if err == nil {
		t.Fatalf("Guard should block by default")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("unexpected guard error: %v", err)
	}
	rt.Cfg.AllowMutations = true
	if err := rt.Guard("op"); err != nil {
		t.Fatalf("Guard should allow with --allow-mutations: %v", err)
	}
}

func TestAuthLoginNoInputHardFails(t *testing.T) {
	noColor(t)
	_, errb, code := runStdin(t, "", "auth", "login", "--provider", "serpapi", "--no-input", "--json")
	if code != 13 {
		t.Fatalf("exit = %d, want 13 (input required)", code)
	}
	if !strings.Contains(errb, "INPUT_REQUIRED") {
		t.Fatalf("missing INPUT_REQUIRED: %s", errb)
	}
}

func TestAuthLoginReadsStdin(t *testing.T) {
	noColor(t)
	out, _, code := runStdin(t, "secret-key-123\n", "auth", "login", "--provider", "serpapi", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.Contains(out, "secret-key-123") {
		t.Fatalf("secret leaked to stdout: %s", out)
	}
}

func TestDidYouMean(t *testing.T) {
	noColor(t)
	_, errb, code := run(t, "searc", "x")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage)", code)
	}
	if !strings.Contains(errb, "did you mean") || !strings.Contains(errb, "search") {
		t.Fatalf("missing suggestion: %s", errb)
	}
}

func TestSchemaHasSafetyAndExitCodes(t *testing.T) {
	noColor(t)
	out, _, code := run(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit = %d, want 0", code)
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("schema not valid JSON: %v", err)
	}
	if _, ok := s["safety"]; !ok {
		t.Fatalf("schema missing safety state")
	}
	if _, ok := s["exit_codes"]; !ok {
		t.Fatalf("schema missing exit_codes")
	}
	if _, ok := s["providers"]; !ok {
		t.Fatalf("schema missing providers")
	}
}

// TestSchemaSnapshot is the required CI gate (contract §10): any change to the command
// tree / exit-code table / providers is a reviewed golden diff, not a silent break.
// Run `AMZN_UPDATE_GOLDEN=1 go test ./internal/cli/...` to regenerate after an intended change.
func TestSchemaSnapshot(t *testing.T) {
	noColor(t)
	out, _, code := run(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit = %d, want 0", code)
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("schema not JSON: %v", err)
	}
	// Strip volatile fields so the snapshot tracks only the stable contract surface.
	delete(s, "version")
	if safety, ok := s["safety"].(map[string]any); ok {
		// runtime toggles, not contract shape
		for _, k := range []string{"allow_mutations", "dry_run", "no_input", "wrap_untrusted"} {
			delete(safety, k)
		}
	}
	stable, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "schema.golden.json")
	if os.Getenv("AMZN_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, append(stable, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("updated golden")
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("missing golden (run AMZN_UPDATE_GOLDEN=1 go test ./...): %v", err)
	}
	if strings.TrimSpace(string(want)) != strings.TrimSpace(string(stable)) {
		t.Fatalf("schema drift — review and regenerate with AMZN_UPDATE_GOLDEN=1\n--- got ---\n%s", stable)
	}
}
