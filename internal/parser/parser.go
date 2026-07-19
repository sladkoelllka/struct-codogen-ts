package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var structTagPatterns = map[string]*regexp.Regexp{
	"json":    regexp.MustCompile(`json:"([^"]*)"`),
	"uri":     regexp.MustCompile(`uri:"([^"]*)"`),
	"form":    regexp.MustCompile(`form:"([^"]*)"`),
	"binding": regexp.MustCompile(`binding:"([^"]*)"`),
}

// ParseFile парсит Go файл и извлекает из него определения структур
func ParseFile(path string) (*File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	file := &File{
		Path:     path,
		Package:  f.Name.Name,
		Dir:      filepath.Dir(path),
		Entity:   extractEntityFromDir(filepath.Dir(path)),
		Type:     extractTypeFromPath(path),
		Imports:  extractImports(f),
		Structs:  []Struct{},
		Generate: true,
	}
	stringConstValues := collectStringConstValues(f)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if typeSpec.Assign.IsValid() {
				file.Structs = append(file.Structs, Struct{
					Name:      typeSpec.Name.Name,
					Package:   file.Package,
					Fields:    []Field{},
					Comment:   extractTypeComment(genDecl, typeSpec),
					AliasType: typeToString(typeSpec.Type),
				})
				continue
			}

			if isStringType(typeSpec.Type) {
				file.Structs = append(file.Structs, Struct{
					Name:      typeSpec.Name.Name,
					Package:   file.Package,
					Fields:    []Field{},
					Comment:   extractTypeComment(genDecl, typeSpec),
					AliasType: "string",
					Values:    stringConstValues[typeSpec.Name.Name],
				})
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			strct := Struct{
				Name:    typeSpec.Name.Name,
				Package: file.Package,
				Fields:  []Field{},
			}

			strct.Comment = extractTypeComment(genDecl, typeSpec)

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					strct.Fields = append(strct.Fields, newEmbeddedField(field))
					continue
				}

				for _, name := range field.Names {
					strct.Fields = append(strct.Fields, newNamedField(name.Name, field))
				}
			}

			if len(strct.Fields) > 0 {
				file.Structs = append(file.Structs, strct)
			}
		}
	}

	return file, nil
}

func collectStringConstValues(f *ast.File) map[string][]string {
	result := map[string][]string{}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		var currentType string
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if valueSpec.Type != nil {
				currentType = namedConstType(valueSpec.Type)
			}
			if currentType == "" {
				continue
			}

			for i := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				value, ok := stringLiteralValue(valueSpec.Values[i])
				if !ok {
					continue
				}
				result[currentType] = append(result[currentType], value)
			}
		}
	}

	return result
}

func namedConstType(expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	if ident.Name == "string" {
		return ""
	}
	return ident.Name
}

func isStringType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "string"
}

func stringLiteralValue(expr ast.Expr) (string, bool) {
	basicLit, ok := expr.(*ast.BasicLit)
	if !ok || basicLit.Kind != token.STRING {
		return "", false
	}

	value, err := strconv.Unquote(basicLit.Value)
	if err != nil {
		return "", false
	}

	return value, true
}

func extractImports(f *ast.File) map[string]string {
	imports := make(map[string]string, len(f.Imports))

	for _, spec := range f.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		if importPath == "" {
			continue
		}

		localName := path.Base(importPath)
		if spec.Name != nil {
			if spec.Name.Name == "_" || spec.Name.Name == "." {
				continue
			}
			localName = spec.Name.Name
		}

		imports[localName] = importPath
	}

	return imports
}

func extractTypeComment(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) string {
	if typeSpec.Doc != nil {
		return typeSpec.Doc.Text()
	}
	if genDecl.Doc != nil {
		return genDecl.Doc.Text()
	}
	return ""
}

func newEmbeddedField(field *ast.Field) Field {
	fieldType := typeToString(field.Type)

	return Field{
		Name:     fieldType,
		Type:     fieldType,
		JSONTag:  extractStructTag(field, "json"),
		UriTag:   extractStructTag(field, "uri"),
		FormTag:  extractStructTag(field, "form"),
		Binding:  extractStructTag(field, "binding"),
		Embedded: true,
	}
}

func newNamedField(name string, field *ast.Field) Field {
	fieldType := typeToString(field.Type)

	return Field{
		Name:    name,
		Type:    fieldType,
		JSONTag: extractStructTag(field, "json"),
		UriTag:  extractStructTag(field, "uri"),
		FormTag: extractStructTag(field, "form"),
		Binding: extractStructTag(field, "binding"),
	}
}

// typeToString преобразует тип из AST в строковое представление
func typeToString(t ast.Expr) string {
	switch v := t.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + typeToString(v.X)
	case *ast.ArrayType:
		return "[]" + typeToString(v.Elt)
	case *ast.MapType:
		return "map[" + typeToString(v.Key) + "]" + typeToString(v.Value)
	case *ast.SelectorExpr:
		return typeToString(v.X) + "." + v.Sel.Name
	case *ast.IndexExpr:
		return typeToString(v.X) + "[" + typeToString(v.Index) + "]"
	case *ast.IndexListExpr:
		var indexes []string
		for _, index := range v.Indices {
			indexes = append(indexes, typeToString(index))
		}
		return typeToString(v.X) + "[" + strings.Join(indexes, ", ") + "]"
	default:
		return "interface{}"
	}
}

func extractStructTag(field *ast.Field, tagName string) string {
	if field.Tag == nil {
		return ""
	}

	pattern, ok := structTagPatterns[tagName]
	if !ok {
		return ""
	}

	tag := strings.Trim(field.Tag.Value, "`")
	matches := pattern.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// extractTypeFromPath извлекает тип файла из пути (например, request из request.go)
func extractTypeFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".go")
}

// extractEntityFromDir извлекает entity path из директории (например, user из .../handler/user/dto
// или report/whitebox из .../handler/report/whitebox/dto)
func extractEntityFromDir(dir string) string {
	parts := strings.Split(filepath.Clean(dir), string(filepath.Separator))
	entityParts := extractEntityParts(parts)
	if len(entityParts) > 0 {
		return filepath.ToSlash(filepath.Join(entityParts...))
	}

	return "unknown"
}

func extractEntityParts(parts []string) []string {
	for i, part := range parts {
		if part == "handler" && i+1 < len(parts) {
			end := i + 1
			for end < len(parts) && parts[end] != "dto" {
				end++
			}
			if end > i+1 {
				return parts[i+1 : end]
			}
			return parts[i+1 : i+2]
		}
	}

	for i, part := range parts {
		if part == "internal" && i+1 < len(parts) {
			return parts[i+1:]
		}
		if part == "pkg" && i+1 < len(parts) {
			return parts[i:]
		}
	}

	return nil
}
