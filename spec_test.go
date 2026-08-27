//go:build spec

// Package firezone's OpenAPI conformance check.
//
// Every field this SDK decodes is an assumption about what the server
// sends. Unit tests can't test those assumptions - they assert against
// fixtures written by the same person who wrote the struct tag, so a
// misspelled field passes every test and silently decodes as a zero
// value forever. This test checks the struct tags against Firezone's
// published OpenAPI spec instead, which is the only thing that actually
// knows.
//
// Run it with `mise run spec-check`. It needs the spec:
//
//	FIREZONE_OPENAPI=/path/to/openapi.json mise run spec-check
//
// and skips when it can't find one, so it never blocks a normal build.
// Set FIREZONE_OPENAPI_STRICT=1 to turn "no spec found" into a failure
// (which is what CI should do, once the spec has a stable home).
package firezone

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// schemaFor maps an SDK type to its OpenAPI schema when the names
// differ. Types not listed here are matched by identical name.
var schemaFor = map[string]string{
	"ClientDevice":       "Client",
	"ProvisionedGateway": "Gateway",
	"Condition":          "PolicyCondition",
	"Filter":             "ResourceFilter",
}

// skipTypes are SDK types with no OpenAPI counterpart, each with the
// reason. A type that is neither here nor resolvable to a schema fails
// the test - that is deliberate, so a new resource can't be added
// without someone deciding which bucket it belongs in.
var skipTypes = map[string]string{
	"AuthProvider":        "embedded base type; its fields are checked via each concrete provider",
	"GroupMember":         "no standalone schema - inlined in group membership responses",
	"RotatedGatewayToken": "no standalone schema - inlined in the token rotation response",
}

// TestSpecConformance fails when the SDK declares a JSON field the
// server never sends. Fields the server sends that the SDK omits are
// reported but not fatal: not exposing a field is a deliberate scope
// choice, while decoding one that doesn't exist is always a bug.
func TestSpecConformance(t *testing.T) {
	specPath := findSpec(t)
	schemas := loadSchemas(t, specPath)
	structs := parseStructs(t)

	var missingSchema []string

	for _, name := range sortedKeys(structs) {
		if reason, ok := skipTypes[name]; ok {
			t.Logf("skip %s: %s", name, reason)
			continue
		}
		if !isReadModel(name) {
			continue
		}
		// Types with no JSON tags decode nothing - services, the client
		// itself, errors built by hand. There is no assumption to check.
		if len(structs[name]) == 0 {
			continue
		}

		schemaName := name
		if alias, ok := schemaFor[name]; ok {
			schemaName = alias
		}
		props, ok := schemas[schemaName]
		if !ok {
			missingSchema = append(missingSchema, name)
			continue
		}

		fields := structs[name]
		var ghosts []string
		for _, f := range fields {
			if _, ok := props[f]; !ok {
				ghosts = append(ghosts, f)
			}
		}
		if len(ghosts) > 0 {
			sort.Strings(ghosts)
			t.Errorf("%s (schema %q): declares %d field(s) absent from the spec, which will always decode as zero: %s",
				name, schemaName, len(ghosts), strings.Join(ghosts, ", "))
		}

		// Informational: spec fields this type doesn't expose.
		var uncovered []string
		for p := range props {
			if !contains(fields, p) {
				uncovered = append(uncovered, p)
			}
		}
		if len(uncovered) > 0 {
			sort.Strings(uncovered)
			t.Logf("%s: %d spec field(s) not exposed by the SDK: %s",
				name, len(uncovered), strings.Join(uncovered, ", "))
		}
	}

	if len(missingSchema) > 0 {
		sort.Strings(missingSchema)
		t.Errorf("no OpenAPI schema found for: %s\n"+
			"Add each to schemaFor (if the schema is named differently) or to skipTypes (with a reason).",
			strings.Join(missingSchema, ", "))
	}
}

// findSpec locates the OpenAPI document, preferring an explicit path so
// the check works against a local monorepo checkout.
func findSpec(t *testing.T) string {
	if p := os.Getenv("FIREZONE_OPENAPI"); p != "" {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("FIREZONE_OPENAPI=%s: %v", p, err)
		}
		return p
	}
	// Relative sibling checkouts, so this works regardless of where the
	// projects directory lives.
	const specInRepo = "elixir/priv/static/openapi.json"
	for _, p := range []string{
		"openapi.json",
		"../../firezone-dev/firezone/" + specInRepo,
		"../../firezone/" + specInRepo,
		"../firezone/" + specInRepo,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	msg := "no OpenAPI spec found; set FIREZONE_OPENAPI=/path/to/openapi.json"
	if os.Getenv("FIREZONE_OPENAPI_STRICT") != "" {
		t.Fatal(msg)
	}
	t.Skip(msg)
	return ""
}

// loadSchemas reads components.schemas into schema name -> property set.
func loadSchemas(t *testing.T, path string) map[string]map[string]struct{} {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if len(doc.Components.Schemas) == 0 {
		t.Fatalf("spec %s has no components.schemas", path)
	}
	out := make(map[string]map[string]struct{}, len(doc.Components.Schemas))
	for name, s := range doc.Components.Schemas {
		props := make(map[string]struct{}, len(s.Properties))
		for p := range s.Properties {
			props[p] = struct{}{}
		}
		out[name] = props
	}
	t.Logf("loaded %d schemas from %s", len(out), path)
	return out
}

// parseStructs returns every struct in the package as its list of JSON
// field names, with embedded structs flattened in - without that, a type
// that embeds a shared base looks like it is missing all of the base's
// fields.
func parseStructs(t *testing.T) map[string][]string {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), ".go") && !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}

	own := map[string][]string{}    // type -> its own json tags
	embeds := map[string][]string{} // type -> embedded type names

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				name := ts.Name.Name
				if _, seen := own[name]; !seen {
					own[name] = nil
				}
				for _, f := range st.Fields.List {
					// An embedded field has no names.
					if len(f.Names) == 0 {
						if id, ok := f.Type.(*ast.Ident); ok {
							embeds[name] = append(embeds[name], id.Name)
						}
						continue
					}
					if f.Tag == nil {
						continue
					}
					tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
					jt := tag.Get("json")
					if jt == "" || jt == "-" {
						continue
					}
					if fname := strings.Split(jt, ",")[0]; fname != "" {
						own[name] = append(own[name], fname)
					}
				}
				return true
			})
		}
	}

	// Flatten embedded types (depth-limited; the SDK nests one level).
	resolved := make(map[string][]string, len(own))
	var expand func(string, int) []string
	expand = func(name string, depth int) []string {
		if depth > 8 {
			t.Fatalf("embedding cycle at %s", name)
		}
		fields := append([]string(nil), own[name]...)
		for _, e := range embeds[name] {
			if _, ok := own[e]; ok {
				fields = append(fields, expand(e, depth+1)...)
			}
		}
		return fields
	}
	for name := range own {
		resolved[name] = expand(name, 0)
	}
	t.Logf("parsed %d struct types from package source", len(resolved))
	return resolved
}

// isReadModel reports whether a type decodes a server response. Request
// bodies and option structs are encoded, not decoded, so a field the
// spec doesn't list is a different (and much louder) kind of problem.
func isReadModel(name string) bool {
	if name == "" || !ast.IsExported(name) {
		return false
	}
	for _, p := range []string{"Create", "Update", "Provision", "Patch", "Replace"} {
		if strings.HasPrefix(name, p) {
			return false
		}
	}
	for _, s := range []string{"Request", "Options", "Metadata"} {
		if strings.HasSuffix(name, s) {
			return false
		}
	}
	return true
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
