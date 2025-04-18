package main

import (
	"bufio"
	"encoding/json"
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

type OutputFormat string

const (
	FormatText OutputFormat = "txt"
	FormatJSON OutputFormat = "json"

	svgFile = "struct-layout.svg"
)

func analyzeStructs(input string, format OutputFormat, generateSVG bool) {
	src := fmt.Sprintf("package main\n\n%s", input)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "input.go", src, parser.AllErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing input: %v\n", err)
		os.Exit(1)
	}

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	_, err = conf.Check("main", fset, []*ast.File{file}, info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error type checking: %v\n", err)
		os.Exit(1)
	}

	sizes := types.StdSizes{WordSize: 8, MaxAlign: 8}
	structs := structi.AnalyzeNestedStructs(file, &sizes, info, fset)

	if generateSVG {
		svgOutput, err := svg.BuildVisualization(structs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building SVG: %v\n", err)
			os.Exit(1)
		}
		err = os.WriteFile(svgFile, []byte(svgOutput), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing svg file: %v\n", err)
			os.Exit(1)
		}
	}

	if format == FormatJSON {
		jsonOutput, err := json.MarshalIndent(structs, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error encoding json: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	} else {
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
}

func readStructFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	var builder strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	return builder.String(), nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  --format string    Output format (json or txt) (default \"txt\")\n")
	fmt.Fprintf(os.Stderr, "  --struct string    Inline struct definition\n")
	fmt.Fprintf(os.Stderr, "  --file string      Path to file containing struct definitions\n")
	fmt.Fprintf(os.Stderr, "  --svg              Generate SVG visualization (default false)\n")
	fmt.Fprintf(os.Stderr, "  --help             Show help message\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s --struct 'type MyStruct struct { a int; b string }'\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --file structs.go\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --format json --struct 'type MyStruct struct { a int; b string }'\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --svg --struct 'type MyStruct struct { a int; b string }'\n", os.Args[0])
	os.Exit(1)
}

func main() {
	formatFlag := flag.String("format", "txt", "Output format (json or txt)")
	structFlag := flag.String("struct", "", "Inline struct definition")
	fileFlag := flag.String("file", "", "Path to file containing struct definitions")
	helpFlag := flag.Bool("help", false, "Show help message")
	svgFlag := flag.Bool("svg", false, "Generate SVG visualization")

	flag.Parse()

	if *helpFlag || (len(os.Args) == 1) {
		printUsage()
	}

	format := OutputFormat(*formatFlag)
	if format != FormatJSON && format != FormatText {
		fmt.Fprintf(os.Stderr, "invalid format: %s. use 'json' or 'txt'\n", format)
		os.Exit(1)
	}

	var input string
	var err error

	if *fileFlag != "" {
		input, err = readStructFromFile(*fileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading struct from file: %v\n", err)
			os.Exit(1)
		}
	} else if *structFlag != "" {
		input = *structFlag
	} else {
		fmt.Fprintf(os.Stderr, "error: no struct definition provided\n")
		printUsage()
	}

	if input == "" {
		fmt.Fprintf(os.Stderr, "error: empty struct definition\n")
		printUsage()
	}

	analyzeStructs(input, format, *svgFlag)
}
