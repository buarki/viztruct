package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/buarki/viztruct/structi"
	"github.com/buarki/viztruct/svg"
)

var (
	binVersion = "devel"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			binVersion = info.Main.Version
		} else {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					binVersion = "devel-" + setting.Value[:7]
					break
				}
			}
		}
	}
}

type OutputFormat string

const (
	FormatText OutputFormat = "txt"
	FormatJSON OutputFormat = "json"

	svgFile = "struct-layout.svg"
)

func analyzeStructs(input string, format OutputFormat, generateSVG bool, filePath string, strategyNames []string) {
	var structs []structi.Info
	var err error

	if filePath != "" {
		structs, err = structi.AnalyseFromFileWithStrategies(filePath, strategyNames)
	} else {
		structs, err = structi.AnalyseStructsWithStrategies(input, strategyNames)
	}

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
		fmt.Printf("Found %d struct(s)\n\n", len(structs))

		for _, s := range structs {
			fmt.Printf("Struct: %s\n", s.Name)
			fmt.Printf("  Original size: %d bytes\n", s.OriginalSize)
			fmt.Printf("  Optimized size: %d bytes\n", s.OptimizedSize)

			savings := s.OriginalSize - s.OptimizedSize
			if savings > 0 {
				fmt.Printf("  Potential savings: %d bytes (%.2f%%)\n",
					savings, float64(savings)/float64(s.OriginalSize)*100)
			}

			fmt.Printf("  Wasted bytes: %d (%.2f%%)\n", s.WastedBytes, s.WastedPercent)

			paddingCount := 0
			totalPaddingBytes := int64(0)
			for _, f := range s.Fields {
				if f.IsPadding {
					paddingCount++
					totalPaddingBytes += f.Size
				}
			}
			fmt.Printf("  Padding: %d bytes in %d locations\n", totalPaddingBytes, paddingCount)

			fmt.Println("\n  Original memory layout:")
			fmt.Println("  ┌" + strings.Repeat("─", 70) + "┐")

			for _, f := range s.Fields {
				prefix := "  │ "
				if f.IsPadding {
					fmt.Printf("%s%-12s offset: %3d  size: %3d bytes %s\n",
						prefix, "Padding", f.Offset, f.Size, strings.Repeat("▒", int(f.Size)))
				} else {
					fmt.Printf("%s%-12s offset: %3d  size: %3d bytes  align: %d\n",
						prefix, f.Name+":", f.Offset, f.Size, f.Align)
				}
			}
			fmt.Println("  └" + strings.Repeat("─", 70) + "┘")

			fmt.Println("\n  Optimized layout:")
			fmt.Println("  ┌" + strings.Repeat("─", 70) + "┐")

			for _, f := range s.OptimizedFields {
				prefix := "  │ "
				if f.IsPadding {
					fmt.Printf("%s%-12s offset: %3d  size: %3d bytes %s\n",
						prefix, "Padding", f.Offset, f.Size, strings.Repeat("▒", int(f.Size)))
				} else {
					fmt.Printf("%s%-12s offset: %3d  size: %3d bytes  align: %d\n",
						prefix, f.Name+":", f.Offset, f.Size, f.Align)
				}
			}
			fmt.Println("  └" + strings.Repeat("─", 70) + "┘")

			fmt.Println("\n  Field details:")
			fmt.Printf("  %-20s %-20s %-10s %-10s %-10s\n", "Name", "Type", "Size", "Align", "Offset")
			fmt.Println("  " + strings.Repeat("─", 70))
			for _, f := range s.Fields {
				if !f.IsPadding {
					fmt.Printf("  %-20s %-20s %-10d %-10d %-10d\n",
						f.Name, f.TypeName, f.Size, f.Align, f.Offset)
				}
			}

			fmt.Println()
		}

		if len(structs) > 1 {
			outputSummary(structs)
		}
	}
}

func outputSummary(structs []structi.Info) {
	fmt.Println("\n=== SUMMARY ===")
	fmt.Printf("Total structs analyzed: %d\n\n", len(structs))

	sort.Slice(structs, func(i, j int) bool {
		return structs[i].WastedPercent > structs[j].WastedPercent
	})

	fmt.Println("Structs ranked by wasted space:")
	fmt.Printf("%-20s %-15s %-15s %-15s %-15s\n",
		"Struct Name", "Original Size", "Wasted Bytes", "Wasted %", "Potential Savings")
	fmt.Println(strings.Repeat("─", 80))

	var totalOriginalSize, totalWastedBytes, totalPotentialSavings int64

	for _, s := range structs {
		potentialSavings := s.OriginalSize - s.OptimizedSize
		fmt.Printf("%-20s %-15d %-15d %-15.2f %-15d\n",
			s.Name, s.OriginalSize, s.WastedBytes, s.WastedPercent, potentialSavings)

		totalOriginalSize += s.OriginalSize
		totalWastedBytes += s.WastedBytes
		totalPotentialSavings += potentialSavings
	}

	fmt.Println(strings.Repeat("─", 80))

	var totalWastedPercent float64
	if totalOriginalSize > 0 {
		totalWastedPercent = float64(totalWastedBytes) / float64(totalOriginalSize) * 100
	}

	fmt.Printf("%-20s %-15d %-15d %-15.2f %-15d\n",
		"TOTAL", totalOriginalSize, totalWastedBytes, totalWastedPercent, totalPotentialSavings)

	if totalPotentialSavings > 0 {
		savingsPercent := float64(totalPotentialSavings) / float64(totalOriginalSize) * 100
		fmt.Printf("\nOptimizing all structs could save %d bytes (%.2f%% of total size)\n",
			totalPotentialSavings, savingsPercent)
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
	fmt.Fprintf(os.Stderr, "  --format string      Output format (json or txt) (default \"txt\")\n")
	fmt.Fprintf(os.Stderr, "  --struct string      Inline struct definition\n")
	fmt.Fprintf(os.Stderr, "  --file string        Path to file containing struct definitions\n")
	fmt.Fprintf(os.Stderr, "  --svg                Generate SVG visualization (default false)\n")
	fmt.Fprintf(os.Stderr, "  --strategies string  Comma-separated list of optimization strategies to use\n")
	fmt.Fprintf(os.Stderr, "                       Available strategies: %s (default: all)\n",
		strings.Join(structi.GetAllOptimizerNames(), ", "))
	fmt.Fprintf(os.Stderr, "  --version            Show version information\n")
	fmt.Fprintf(os.Stderr, "  --help               Show help message\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s --struct 'type MyStruct struct { a int; b string }'\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --file structs.go\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --strategies=\"greedy,group\" --file structs.go\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --format json --struct 'type MyStruct struct { a int; b string }'\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --svg --struct 'type MyStruct struct { a int; b string }'\n", os.Args[0])
}

func main() {
	formatFlag := flag.String("format", "txt", "Output format (json or txt)")
	structDef := flag.String("struct", "", "Struct definition to visualize")
	fileFlag := flag.String("file", "", "Path to file containing struct definitions")
	helpFlag := flag.Bool("help", false, "Show help message")
	svgFlag := flag.Bool("svg", false, "Generate SVG visualization")
	version := flag.Bool("version", false, "Show version information")
	strategiesFlag := flag.String("strategies", "", "Comma-separated list of optimization strategies to use")

	flag.Parse()

	if *version {
		fmt.Printf("viztruct version %s\n", binVersion)
		os.Exit(0)
	}

	if *helpFlag || len(os.Args) == 1 {
		printUsage()
		os.Exit(1)
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
		os.Exit(1)
	}

	if input == "" {
		fmt.Fprintf(os.Stderr, "error: empty struct definition\n")
		printUsage()
		os.Exit(1)
	}

	var strategyNames []string
	if *strategiesFlag != "" {
		strategyNames = strings.Split(*strategiesFlag, ",")
		for i := range strategyNames {
			strategyNames[i] = strings.TrimSpace(strategyNames[i])
		}

		allNames := structi.GetAllOptimizerNames()
		allNamesMap := make(map[string]bool)
		for _, name := range allNames {
			allNamesMap[name] = true
		}

		for _, name := range strategyNames {
			if !allNamesMap[name] {
				fmt.Fprintf(os.Stderr, "error: unknown strategy '%s'\nAvailable strategies: %s\n",
					name, strings.Join(allNames, ", "))
				os.Exit(1)
			}
		}
	}

	analyzeStructs(input, format, *svgFlag, *fileFlag, strategyNames)
}
