package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"
)

func processFile(filePath string, allMappings map[string]string) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Failed to parse file: %v\n", err)
		os.Exit(1)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		for _, field := range structType.Fields.List {
			if field.Tag == nil {
				continue
			}

			tag := strings.Trim(field.Tag.Value, "`")
			jsonTag := reflect.StructTag(tag).Get("json")
			if jsonTag == "" {
				continue
			}

			for _, fieldName := range field.Names {
				// Change the first letter to lowercase
				convertedKey := string(unicode.ToLower(rune(fieldName.Name[0]))) + fieldName.Name[1:]
				if _, exists := allMappings[convertedKey]; !exists {
					allMappings[convertedKey] = jsonTag
				}
			}
		}

		return false
	})
}

func main() {
	var filePattern string
	var individualFilePaths string
	var output string

	flag.StringVar(&filePattern, "pattern", "", "The file pattern to match (e.g. ./graph/model/*.go)")
	flag.StringVar(&individualFilePaths, "files", "", "Comma-separated list of individual file paths (e.g. ./db/sqlc/models.go,./graph/model/auth.go)")
	flag.StringVar(&output, "output", "./generate/mapping/field_and_json_mapping.go", "Output file path")

	flag.Parse()

	if filePattern == "" && individualFilePaths == "" {
		fmt.Println("You must provide either a file pattern or a list of individual file paths.")
		os.Exit(1)
	}

	allMappings := make(map[string]string)

	if filePattern != "" {
		matches, err := filepath.Glob(filePattern)
		if err != nil {
			fmt.Printf("Failed to match file pattern: %v\n", err)
			os.Exit(1)
		}

		for _, filePath := range matches {
			processFile(filePath, allMappings)
		}
	}

	if individualFilePaths != "" {
		files := strings.Split(individualFilePaths, ",")
		for _, filePath := range files {
			processFile(filePath, allMappings)
		}
	}

	tmpl := `package mapping

var AllMappings = map[string]string{
	{{- range $key, $value := .}}
		"{{$key}}": "{{$value}}",
	{{- end}}
}
`

	t := template.Must(template.New("mappings").Parse(tmpl))
	f, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = t.Execute(f, allMappings)
	if err != nil {
		panic(err)
	}
}
