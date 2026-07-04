package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sladkoelllka/struct-codogen-ts/internal/permissions"
)

func main() {
	input := flag.String("input", "", "Input Go file or directory")
	output := flag.String("output", "", "Output TypeScript file")
	typeName := flag.String("name", "Permission", "Generated TypeScript const/type name")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Println("Usage: permission-codegen-ts -input <file-or-dir> -output <file.ts> [-name Permission]")
		os.Exit(1)
	}

	matches, err := filepath.Glob(*input)
	if err != nil {
		fmt.Printf("Error: invalid input pattern: %v\n", err)
		os.Exit(1)
	}
	if len(matches) == 0 {
		fmt.Printf("Error: no files or directories match the input pattern: %s\n", *input)
		os.Exit(1)
	}

	var allPermissions []permissions.Permission
	seen := map[string]bool{}
	for _, match := range matches {
		parsedPermissions, err := permissions.ParsePath(match)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", match, err)
			continue
		}
		for _, permission := range parsedPermissions {
			if seen[permission.Name] {
				continue
			}
			seen[permission.Name] = true
			allPermissions = append(allPermissions, permission)
		}
	}

	err = os.MkdirAll(filepath.Dir(*output), 0755)
	if err != nil {
		fmt.Printf("Error: failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(*output, []byte(permissions.Generate(*typeName, allPermissions)), 0644)
	if err != nil {
		fmt.Printf("Error: failed to write %s: %v\n", *output, err)
		os.Exit(1)
	}

	fmt.Printf("Generated 1 file(s)\n")
	fmt.Printf("Processed %d permission(s)\n", len(allPermissions))
}
