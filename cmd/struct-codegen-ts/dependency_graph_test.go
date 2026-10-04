package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sladkoelllka/struct-codogen-ts/internal/generator"
	"github.com/sladkoelllka/struct-codogen-ts/internal/parser"
)

func TestImportedDependenciesIncludeOnlyReachableTypes(t *testing.T) {
	root := t.TempDir()
	write := func(name, contents string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.local/app\n\ngo 1.25\n")
	write("handler/order/dto/response.go", "package dto\nimport \"example.local/app/readmodel\"\ntype Response = readmodel.Order\n")
	write("readmodel/order.go", "package readmodel\nimport \"example.local/app/value\"\ntype Order struct { Items []Item `json:\"items\"`; Currency value.Currency `json:\"currency\"` }\ntype Item = SavedItem\n")
	write("readmodel/item.go", "package readmodel\ntype SavedItem struct { Name string `json:\"name\"` }\n")
	write("readmodel/unrelated.go", "package readmodel\nimport \"example.local/app/pagination\"\ntype User struct { ID int `json:\"id\"` }\ntype Users = pagination.ListResult[User]\n")
	write("pagination/page.go", "package pagination\ntype ListResult[T any] struct { Items []T `json:\"items\"` }\n")
	write("value/currency.go", "package value\ntype Currency string\nconst RUB Currency = \"RUB\"\n")

	primary, err := parser.ParseFile(filepath.Join(root, "handler/order/dto/response.go"))
	if err != nil {
		t.Fatal(err)
	}

	dependencies := collectImportedStructFiles([]*parser.File{primary})
	names := make(map[string]bool)
	for _, file := range dependencies {
		for _, strct := range file.Structs {
			names[strct.Name] = true
		}
	}

	for _, expected := range []string{"Order", "Item", "SavedItem", "Currency"} {
		if !names[expected] {
			t.Fatalf("reachable type %s was omitted", expected)
		}
	}

	for _, excluded := range []string{"User", "Users", "ListResult"} {
		if names[excluded] {
			t.Fatalf("unrelated type %s was included", excluded)
		}
	}

	outputs := generator.NewGenerator(generator.GeneratorConfig{ExportType: "interface"}).GeneratePerFile(append([]*parser.File{primary}, dependencies...), filepath.Join(root, "generated"))
	var content strings.Builder
	for _, output := range outputs {
		content.WriteString(output.Content)
	}

	for _, expected := range []string{"export type Response = Order;", "export type Item = SavedItem;", "items: Item[];"} {
		if !strings.Contains(content.String(), expected) {
			t.Fatalf("missing generated declaration %q", expected)
		}
	}
}
