package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/buarki/viztruct/structi"
	"github.com/buarki/viztruct/svg"
)

var (
	binVersion = ""
)

type OutputFormat string

const (
	FormatText OutputFormat = "txt"
	FormatJSON OutputFormat = "json"

	svgFile = "struct-layout.svg"
)

func analyzeStructs(input string, format OutputFormat, generateSVG bool) {
	structs, err := structi.AnalyseStructs(input)
	if err != nil {
		if errI, ok := err.(*structi.Error); ok {
			fmt.Fprintf(os.Stderr, "%v\n", errI.Error())
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}

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
	fmt.Fprintf(os.Stderr, "  --version          Show version information\n")
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
	structDef := flag.String("struct", "", "Struct definition to visualize")
	fileFlag := flag.String("file", "", "Path to file containing struct definitions")
	helpFlag := flag.Bool("help", false, "Show help message")
	svgFlag := flag.Bool("svg", false, "Generate SVG visualization")
	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *version {
		fmt.Printf("viztruct version %s\n", binVersion)
		os.Exit(0)
	}

	if *helpFlag || len(os.Args) == 1 {
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
	} else if *structDef != "" {
		input = *structDef
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
