package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sladkoelllka/struct-codogen-ts/internal/generator"
	"github.com/sladkoelllka/struct-codogen-ts/internal/parser"
)

const supportedMode = "per-file"

type cliConfig struct {
	input      string
	output     string
	exportType string
	mode       string
}

func main() {
	config := parseFlags()
	validateConfig(config)

	parsedFiles, totalStructs := collectParsedFiles(config.input)
	if totalStructs == 0 {
		fmt.Println("Warning: no structs found")
		return
	}

	for _, file := range parsedFiles {
		log.Printf("Processing file %s", file.Path)
	}

	gen := generator.NewGenerator(generator.GeneratorConfig{
		ExportType: config.exportType,
	})

	outputs := gen.GeneratePerFile(parsedFiles, config.output)

	for _, output := range outputs {
		log.Printf("Writing output %s", output.Path)
	}

	writeOutputs(config.output, outputs)

	fmt.Printf("Generated %d file(s)\n", len(outputs))
	fmt.Printf("Processed %d structs\n", totalStructs)
}

func parseFlags() cliConfig {
	// Флаги CLI
	// input  — путь к Go файлу или директории с Go файлами
	// output — путь к результирующему TypeScript файлу или директории
	// type   — тип экспорта в TypeScript (interface или type)
	// mode   — legacy-флаг, поддерживается только значение per-file
	input := flag.String("input", "", "Input Go file or directory")
	output := flag.String("output", "", "Output TypeScript file or directory")
	exportType := flag.String("type", "interface", "Export type: interface or type")
	mode := flag.String("mode", supportedMode, "Generation mode: per-file")
	flag.Parse()

	return cliConfig{
		input:      *input,
		output:     *output,
		exportType: *exportType,
		mode:       *mode,
	}
}

func validateConfig(config cliConfig) {
	if config.input == "" {
		fmt.Println("Usage: struct-codegen -input <file-or-dir> -output <output> [-type interface|type] [-mode per-file]")
		fmt.Println("\nGeneration always runs in per-file mode.")
		os.Exit(1)
	}

	if config.output == "" {
		fmt.Println("Error: -output flag is required")
		os.Exit(1)
	}

	if config.mode != supportedMode {
		fmt.Printf("Error: invalid mode '%s'. Only per-file is supported\n", config.mode)
		os.Exit(1)
	}
}

func collectParsedFiles(input string) ([]*parser.File, int) {
	matches, err := filepath.Glob(input)
	if err != nil {
		fmt.Printf("Error: invalid input pattern: %v\n", err)
		os.Exit(1)
	}
	if len(matches) == 0 {
		fmt.Printf("Error: no files or directories match the input pattern: %s\n", input)
		os.Exit(1)
	}

	var parsedFiles []*parser.File
	var totalStructs int

	for _, match := range matches {
		files, structs := collectMatchFiles(match)
		if len(files) == 0 {
			continue
		}

		parsedFiles = append(parsedFiles, files...)
		totalStructs += structs
	}

	parsedFiles = append(parsedFiles, collectImportedStructFiles(parsedFiles)...)

	return parsedFiles, totalStructs
}

func collectMatchFiles(match string) ([]*parser.File, int) {
	fileInfo, err := os.Stat(match)
	if err != nil {
		fmt.Printf("Warning: cannot stat %s: %v\n", match, err)
		return nil, 0
	}

	if fileInfo.IsDir() {
		return collectDirectoryFiles(match)
	}

	file, ok := parseStructFile(match)
	if !ok {
		return nil, 0
	}

	return []*parser.File{file}, len(file.Structs)
}

func collectDirectoryFiles(dir string) ([]*parser.File, int) {
	var parsedFiles []*parser.File
	var totalStructs int

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !isGoSourceFile(path) {
			return nil
		}

		file, ok := parseStructFile(path)
		if !ok {
			return nil
		}

		parsedFiles = append(parsedFiles, file)
		totalStructs += len(file.Structs)
		return nil
	})
	if err != nil {
		fmt.Printf("Error: failed to walk directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	return parsedFiles, totalStructs
}

func parseStructFile(path string) (*parser.File, bool) {
	file, err := parser.ParseFile(path)
	if err != nil {
		fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
		return nil, false
	}
	if len(file.Structs) == 0 {
		return nil, false
	}

	return file, true
}

func collectImportedStructFiles(primaryFiles []*parser.File) []*parser.File {
	var result []*parser.File
	seenPaths := make(map[string]bool, len(primaryFiles))
	seenDirs := make(map[string]bool, len(primaryFiles))
	moduleCache := make(map[string]moduleInfo)

	queue := append([]*parser.File(nil), primaryFiles...)
	for _, file := range primaryFiles {
		seenPaths[file.Path] = true
		seenDirs[file.Dir] = true
	}

	for i := 0; i < len(queue); i++ {
		file := queue[i]
		module := cachedModuleInfo(file.Dir, moduleCache)
		if module.Root == "" || module.Path == "" {
			continue
		}

		for importName, importPath := range file.Imports {
			if !usesImportedPackage(file, importName) {
				continue
			}

			depDir := resolveImportDir(module, importPath)
			if depDir == "" || seenDirs[depDir] {
				continue
			}

			if _, err := os.Stat(depDir); err != nil {
				continue
			}

			depFiles, _ := collectDirectoryFiles(depDir)
			seenDirs[depDir] = true
			for _, depFile := range depFiles {
				if seenPaths[depFile.Path] {
					continue
				}
				if !fileHasJSONTaggedStruct(depFile) {
					continue
				}
				seenPaths[depFile.Path] = true
				result = append(result, depFile)
				queue = append(queue, depFile)
			}
		}
	}

	return result
}

func usesImportedPackage(file *parser.File, importName string) bool {
	prefix := importName + "."
	for _, strct := range file.Structs {
		if strings.Contains(strct.AliasType, prefix) {
			return true
		}
		for _, field := range strct.Fields {
			if strings.Contains(field.Type, prefix) {
				return true
			}
		}
	}

	return false
}

func fileHasJSONTaggedStruct(file *parser.File) bool {
	for _, strct := range file.Structs {
		for _, field := range strct.Fields {
			if field.JSONTag != "" {
				return true
			}
		}
	}

	return false
}

type moduleInfo struct {
	Root string
	Path string
}

func cachedModuleInfo(startDir string, cache map[string]moduleInfo) moduleInfo {
	if info, ok := cache[startDir]; ok {
		return info
	}

	info := findModuleInfo(startDir)
	cache[startDir] = info
	return info
}

func findModuleInfo(startDir string) moduleInfo {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if stat, err := os.Stat(goModPath); err == nil && !stat.IsDir() {
			return moduleInfo{
				Root: dir,
				Path: readModulePath(goModPath),
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return moduleInfo{}
		}
		dir = parent
	}
}

func readModulePath(goModPath string) string {
	file, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}

	return ""
}

func resolveImportDir(module moduleInfo, importPath string) string {
	prefix := module.Path + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return ""
	}

	relPath := strings.TrimPrefix(importPath, prefix)
	return filepath.Join(module.Root, filepath.FromSlash(relPath))
}

func isGoSourceFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

func writeOutputs(outputDir string, outputs []generator.FileOutput) {
	if len(outputs) > 1 {
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			fmt.Printf("Error: failed to create output directory: %v\n", err)
			os.Exit(1)
		}
	}

	for _, output := range outputs {
		writeOutput(output)
	}
}

func writeOutput(output generator.FileOutput) {
	err := os.MkdirAll(filepath.Dir(output.Path), 0755)
	if err != nil {
		fmt.Printf("Error: failed to create directory: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(output.Path, []byte(output.Content), 0644)
	if err != nil {
		fmt.Printf("Error: failed to write %s: %v\n", output.Path, err)
		os.Exit(1)
	}
}
