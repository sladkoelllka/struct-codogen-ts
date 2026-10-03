package generator

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sladkoelllka/struct-codogen-ts/internal/parser"
)

type schemaDefinition struct {
	file  *parser.File
	strct parser.Struct
}

type schemaGenerator struct {
	definitions map[string]schemaDefinition
	required    map[string]bool
}

func newSchemaGenerator(files []*parser.File) *schemaGenerator {
	g := &schemaGenerator{
		definitions: make(map[string]schemaDefinition),
		required:    make(map[string]bool),
	}

	for _, file := range files {
		if !file.Generate {
			continue
		}

		for _, strct := range file.Structs {
			if strct.AliasType == "" {
				g.definitions[file.Entity+"/"+strct.Name] = schemaDefinition{file: file, strct: strct}
			}
		}
	}

	for key, definition := range g.definitions {
		if structNeedsZod(definition.strct) {
			g.require(key)
		}
	}

	// Ограничения вложенных и встроенных типов распространяются на содержащие их структуры.
	for changed := true; changed; {
		changed = false
		for key, definition := range g.definitions {
			if g.required[key] {
				continue
			}

			for _, field := range definition.strct.Fields {
				if resolveFieldName(field) != "-" && g.required[resolveStructFileKey(definition.file, schemaElementType(field.Type))] {
					g.require(key)
					changed = true
					break
				}
			}
		}
	}

	return g
}

func (g *schemaGenerator) require(key string) {
	if g.required[key] {
		return
	}

	definition, ok := g.definitions[key]
	if !ok {
		return
	}

	g.required[key] = true
	for _, field := range definition.strct.Fields {
		if resolveFieldName(field) == "-" {
			continue
		}

		g.require(resolveStructFileKey(definition.file, schemaElementType(field.Type)))
	}
}

func schemaElementType(rawType string) string {
	rawType = strings.TrimPrefix(rawType, "*")
	for strings.HasPrefix(rawType, "[]") {
		rawType = strings.TrimPrefix(strings.TrimPrefix(rawType, "[]"), "*")
	}

	return rawType
}

func (g *schemaGenerator) needsFile(file *parser.File) bool {
	for _, strct := range file.Structs {
		if g.required[file.Entity+"/"+strct.Name] {
			return true
		}
	}

	return false
}

// Встроенные поля сохраняют исходный файл, чтобы корректно разрешать импорты типов.
type schemaField struct {
	file  *parser.File
	field parser.Field
}

func (g *schemaGenerator) fields(definition schemaDefinition, visited map[string]bool) []schemaField {
	key := definition.file.Entity + "/" + definition.strct.Name
	if visited[key] {
		return nil
	}

	visited[key] = true
	defer delete(visited, key)
	var fields []schemaField
	directNames := make(map[string]bool)
	for _, field := range definition.strct.Fields {
		if !field.Embedded {
			directNames[resolveFieldName(field)] = true
		}
	}

	for _, field := range definition.strct.Fields {
		if resolveFieldName(field) == "-" {
			continue
		}

		if field.Embedded && strings.Split(field.JSONTag, ",")[0] == "" {
			if embedded, ok := g.definitions[resolveStructFileKey(definition.file, strings.TrimPrefix(field.Type, "*"))]; ok {
				for _, inherited := range g.fields(embedded, visited) {
					if !directNames[resolveFieldName(inherited.field)] {
						fields = append(fields, inherited)
					}
				}
			}

			continue
		}

		fields = append(fields, schemaField{file: definition.file, field: field})
	}

	return fields
}

func (g *schemaGenerator) generate(file *parser.File) string {
	var body strings.Builder
	imports := make(map[string]bool)
	localNames := fileTypeNames(file)

	for _, strct := range file.Structs {
		key := file.Entity + "/" + strct.Name
		if !g.required[key] {
			continue
		}

		fmt.Fprintf(&body, "export const %sSchema = z.object({\n", strct.Name)
		for _, field := range g.fields(g.definitions[key], make(map[string]bool)) {
			initial := func(rawType string, rules []bindingRule) string {
				targetKey := resolveStructFileKey(field.file, rawType)
				target, ok := g.definitions[targetKey]
				if !ok || !g.required[targetKey] {
					return initialZodType(rawType, rules)
				}

				name := target.strct.Name + "Schema"
				if target.file != file {
					path, err := filepath.Rel(filepath.Dir(schemaOutputPathForFile(file, "")), schemaOutputPathForFile(target.file, ""))
					if err != nil {
						return initialZodType(rawType, rules)
					}

					path = strings.TrimSuffix(filepath.ToSlash(path), ".ts")
					if !strings.HasPrefix(path, ".") {
						path = "./" + path
					}

					local := name
					if localNames[target.strct.Name] {
						local = importAliasName(rawType, target.strct.Name) + "Schema"
					}

					imports[fmt.Sprintf("import { %s as %s } from '%s';", name, local, path)] = true
					name = local
				}

				// Аннотация разрывает цикл вывода TypeScript у рекурсивных структур.
				if g.references(targetKey, key, make(map[string]bool)) {
					return "z.lazy((): z.ZodType => " + name + ")"
				}

				return "z.lazy(() => " + name + ")"
			}

			fmt.Fprintf(&body, "  %q: %s,\n", resolveFieldName(field.field), zodSchemaForFieldWithType(field.field, initial))
		}

		body.WriteString("});\n\n")
	}

	var result strings.Builder
	result.WriteString(generatedHeader)
	result.WriteString("import { z } from 'zod';\n")
	lines := make([]string, 0, len(imports))
	for line := range imports {
		lines = append(lines, line)
	}

	sort.Strings(lines)
	for _, line := range lines {
		result.WriteString(line + "\n")
	}

	result.WriteString("\n")
	result.WriteString(body.String())
	return result.String()
}

func (g *schemaGenerator) references(from, target string, visited map[string]bool) bool {
	if from == target {
		return true
	}

	if visited[from] {
		return false
	}

	visited[from] = true
	definition, ok := g.definitions[from]
	if !ok {
		return false
	}

	for _, field := range definition.strct.Fields {
		if resolveFieldName(field) != "-" && g.references(resolveStructFileKey(definition.file, schemaElementType(field.Type)), target, visited) {
			return true
		}
	}

	return false
}
