package generator

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sladkoelllka/struct-codogen-ts/internal/parser"
)

func schemaFile(name string, structs ...parser.Struct) *parser.File {
	return &parser.File{Path: name + ".go", Entity: "offers", Type: name, Generate: true, Structs: structs}
}

func TestSchemasPreserveEmbeddedAndNestedValidation(t *testing.T) {
	file := schemaFile("request",
		parser.Struct{Name: "CreateRequest", Fields: []parser.Field{
			{Name: "RequestID", Type: "int64", JSONTag: "request_id", Binding: "required,gt=0"},
			{Type: "SnapshotRequest", Embedded: true},
		}},
		parser.Struct{Name: "SnapshotRequest", Fields: []parser.Field{
			{Name: "Selections", Type: "[]Selection", JSONTag: "selections", Binding: "required,min=1,max=500,dive"},
			{Name: "Terms", Type: "Terms", JSONTag: "terms"},
		}},
		parser.Struct{Name: "Selection", Fields: []parser.Field{
			{Name: "ID", Type: "int64", JSONTag: "id", Binding: "required,gt=0"},
		}},
		parser.Struct{Name: "Terms", Fields: []parser.Field{
			{Name: "Comment", Type: "string", JSONTag: "comment", Binding: "max=100"},
		}},
	)
	output := generateSchemaFileContent(file)
	for _, expected := range []string{
		`"request_id": z.number().int().gt(0)`,
		`"selections": z.array(z.lazy(() => SelectionSchema)).min(1).max(500)`,
		`"terms": z.lazy(() => TermsSchema)`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing %q: %s", expected, output)
		}
	}

	if strings.Count(output, `"selections":`) != 2 || strings.Contains(output, "z.unknown()") {
		t.Fatalf("embedded fields lost or validation weakened: %s", output)
	}
}

func TestSchemasIncludeInheritedOnlyAndUnconstrainedNestedTypes(t *testing.T) {
	file := schemaFile("request",
		parser.Struct{Name: "Nested", Fields: []parser.Field{{Name: "Name", Type: "string", JSONTag: "name"}}},
		parser.Struct{Name: "Base", Fields: []parser.Field{
			{Name: "ID", Type: "int64", JSONTag: "id", Binding: "gt=0"},
			{Name: "Nested", Type: "*Nested", JSONTag: "nested"},
		}},
		parser.Struct{Name: "Request", Fields: []parser.Field{{Type: "Base", Embedded: true}}},
	)
	output := generateSchemaFileContent(file)
	if !strings.Contains(output, "export const RequestSchema") || !strings.Contains(output, "export const NestedSchema") || !strings.Contains(output, `"nested": z.lazy(() => NestedSchema).optional()`) {
		t.Fatalf("inherited or nested schema missing: %s", output)
	}
}

func TestSchemasResolveTypesFromAnotherFile(t *testing.T) {
	nested := schemaFile("terms", parser.Struct{Name: "Terms", Fields: []parser.Field{{Name: "Comment", Type: "string", JSONTag: "comment", Binding: "max=100"}}})
	parent := schemaFile("request", parser.Struct{Name: "Request", Fields: []parser.Field{{Name: "Terms", Type: "Terms", JSONTag: "terms"}}})
	outputs := NewGenerator(GeneratorConfig{}).GeneratePerFile([]*parser.File{parent, nested}, "output")
	var schema string
	for _, output := range outputs {
		if filepath.Base(output.Path) == "request.schema.ts" {
			schema = output.Content
		}
	}

	if !strings.Contains(schema, "import { TermsSchema as TermsSchema } from './terms.schema';") || !strings.Contains(schema, `"terms": z.lazy(() => TermsSchema)`) {
		t.Fatalf("cross-file reference lost: %s", schema)
	}
}

func TestSchemasHandleRecursiveTypes(t *testing.T) {
	file := schemaFile("request", parser.Struct{Name: "Node", Fields: []parser.Field{
		{Name: "Name", Type: "string", JSONTag: "name", Binding: "required"},
		{Name: "Children", Type: "[]Node", JSONTag: "children"},
	}})
	output := generateSchemaFileContent(file)
	if !strings.Contains(output, `"children": z.array(z.lazy((): z.ZodType => NodeSchema))`) {
		t.Fatalf("recursive schema missing: %s", output)
	}
}

func TestSchemasIgnoreHiddenFieldsAndPreferExplicitFields(t *testing.T) {
	file := schemaFile("request",
		parser.Struct{Name: "Base", Fields: []parser.Field{{Name: "ID", Type: "int64", JSONTag: "id", Binding: "gt=0"}}},
		parser.Struct{Name: "Request", Fields: []parser.Field{
			{Type: "Base", Embedded: true},
			{Name: "ID", Type: "string", JSONTag: "id", Binding: "max=10"},
			{Name: "Hidden", Type: "Base", JSONTag: "-", Binding: "required"},
		}},
	)
	output := generateSchemaFileContent(file)
	request := output[strings.Index(output, "export const RequestSchema"):]
	if strings.Count(request, `"id":`) != 1 || !strings.Contains(request, `"id": z.string().max(10)`) || strings.Contains(request, "Hidden") {
		t.Fatalf("incorrect field precedence: %s", request)
	}
}
