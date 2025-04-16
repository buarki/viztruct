package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"

	"github.com/buarki/viztruct/structi"
	"github.com/buarki/viztruct/svg"
)

func analyzeStructs(input string) {
	src := fmt.Sprintf("package main\n\n%s", input)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "input.go", src, parser.AllErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing input: %v\n", err)
		os.Exit(1)
	}

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	_, err = conf.Check("main", fset, []*ast.File{file}, info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error type checking: %v\n", err)
		os.Exit(1)
	}

	sizes := types.StdSizes{WordSize: 8, MaxAlign: 8}
	structs := structi.AnalyzeNestedStructs(file, &sizes, info, fset)

	svgOutput := svg.BuildVisualization(structs)

	svgFile := "struct_layout.svg"
	err = os.WriteFile(svgFile, []byte(svgOutput), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing SVG file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("SVG visualization saved to %s\n", svgFile)

	for _, s := range structs {
		fmt.Printf("\nStruct: %s\n", s.Name)
		fmt.Printf("Original Size: %d bytes\n", s.OriginalSize)
		fmt.Printf("Optimized Size: %d bytes\n", s.OptimizedSize)
		fmt.Printf("Wasted Space: %d bytes (%.2f%%)\n", s.WastedBytes, s.WastedPercent)

		fmt.Println("\nOriginal Layout:")
		for _, f := range s.Fields {
			if f.IsPadding {
				fmt.Printf("  [padding] %d bytes at offset %d\n", f.Size, f.Offset)
			} else {
				fmt.Printf("  %s (%s) %d bytes at offset %d\n", f.Name, f.TypeName, f.Size, f.Offset)
			}
		}

		fmt.Println("\nOptimized Layout:")
		for _, f := range s.OptimizedFields {
			if f.IsPadding {
				fmt.Printf("  [padding] %d bytes at offset %d\n", f.Size, f.Offset)
			} else {
				fmt.Printf("  %s (%s) %d bytes at offset %d\n", f.Name, f.TypeName, f.Size, f.Offset)
			}
		}
	}
}

func main() {
	flag.Parse()
	args := flag.Args()

	var input string
	if len(args) > 0 {
		input = strings.Join(args, " ")
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		var builder strings.Builder

		for scanner.Scan() {
			builder.WriteString(scanner.Text())
			builder.WriteString("\n")
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		input = builder.String()
	}

	analyzeStructs(input)
}
