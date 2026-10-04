package main

import (
	"go/ast"
	goparser "go/parser"
	"sort"

	"github.com/sladkoelllka/struct-codogen-ts/internal/parser"
)

type typeLocation struct {
	file  *parser.File
	strct parser.Struct
}

// Собираем замыкание ссылок на типы, а не все публичные объявления импортированного пакета.
func collectReachableStructFiles(primaryFiles []*parser.File) []*parser.File {
	moduleCache := make(map[string]moduleInfo)
	directories := make(map[string][]*parser.File)
	primaryPaths := make(map[string]bool)
	included := make(map[string]map[string]bool)
	files := make(map[string]*parser.File)
	queue := make([]typeLocation, 0)

	include := func(file *parser.File, strct parser.Struct) {
		if included[file.Path] == nil {
			included[file.Path] = make(map[string]bool)
		}

		if included[file.Path][strct.Name] {
			return
		}

		included[file.Path][strct.Name] = true
		files[file.Path] = file
		queue = append(queue, typeLocation{file: file, strct: strct})
	}

	for _, file := range primaryFiles {
		primaryPaths[file.Path] = true
		for _, strct := range file.Structs {
			include(file, strct)
		}
	}

	load := func(dir string) []*parser.File {
		if dir == "" {
			return nil
		}

		if values, ok := directories[dir]; ok {
			return values
		}

		values, _ := collectDirectoryFiles(dir)
		directories[dir] = values

		return values
	}
	includeType := func(dir, name string) {
		for _, file := range load(dir) {
			for _, strct := range file.Structs {
				if strct.Name == name {
					include(file, strct)
					return
				}
			}
		}
	}

	for index := 0; index < len(queue); index++ {
		location := queue[index]
		for _, name := range localTypeRefs(location.strct) {
			includeType(location.file.Dir, name)
		}

		module := cachedModuleInfo(location.file.Dir, moduleCache)
		if module.Root == "" || module.Path == "" {
			continue
		}

		for _, ref := range importedTypeRefs(location.strct) {
			path := location.file.Imports[ref.packageName]
			includeType(resolveImportDir(module, path), ref.typeName)
		}
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		if !primaryPaths[path] {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	result := make([]*parser.File, 0, len(paths))
	for _, path := range paths {
		filtered := *files[path]
		filtered.Structs = nil
		for _, strct := range files[path].Structs {
			if included[path][strct.Name] {
				filtered.Structs = append(filtered.Structs, strct)
			}
		}

		result = append(result, &filtered)
	}

	return result
}

type importedTypeReference struct {
	packageName string
	typeName    string
}

func importedTypeRefs(strct parser.Struct) []importedTypeReference {
	types := []string{strct.AliasType}
	for _, field := range strct.Fields {
		types = append(types, field.Type)
	}

	var result []importedTypeReference
	for _, rawType := range types {
		expr, err := goparser.ParseExpr(rawType)
		if err != nil {
			continue
		}

		ast.Inspect(expr, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			identifier, ok := selector.X.(*ast.Ident)
			if ok {
				result = append(result, importedTypeReference{
					packageName: identifier.Name,
					typeName:    selector.Sel.Name,
				})
			}

			return true
		})
	}

	return result
}
