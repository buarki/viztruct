package main

import (
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

func analyzeInlineStructs(input string, format OutputFormat, generateSVG bool, strategyNames []string) {
	structs, err := structi.AnalyseStructsAsStringWithStrategies(input, strategyNames)
	if err != nil {
		if errI, ok := err.(*structi.Error); ok {
			fmt.Fprintf(os.Stderr, "%v\n", errI.Error())
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}

	processResults([][]structi.Info{structs}, format, generateSVG)
}

// analyzeFromPath handles analysis of structs from a file or directory path
func analyzeFromPath(filePath string, format OutputFormat, generateSVG bool, strategyNames []string) {
	structGroups, err := structi.AnalyseStructsAtDirectoryPath(filePath, strategyNames)
	if err != nil {
		if errI, ok := err.(*structi.Error); ok {
			fmt.Fprintf(os.Stderr, "%v\n", errI.Error())
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}

	processResults(structGroups, format, generateSVG)
}

func processResults(structGroups [][]structi.Info, format OutputFormat, generateSVG bool) {
	var allStructs []structi.Info
	for _, group := range structGroups {
		allStructs = append(allStructs, group...)
	}

	if generateSVG {
		svgOutputMap, err := svg.BuildVisualization(allStructs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building SVG: %v\n", err)
			os.Exit(1)
		}

		for structName, svgContent := range svgOutputMap {
			fileName := fmt.Sprintf("%s.svg", structName)
			err = os.WriteFile(fileName, []byte(svgContent), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing svg file %s: %v\n", fileName, err)
				os.Exit(1)
			}
		}

		combinedSvgOutput, err := svg.BuildSingleVisualization(allStructs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building combined SVG: %v\n", err)
			os.Exit(1)
		}
		err = os.WriteFile(svgFile, []byte(combinedSvgOutput), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing combined svg file: %v\n", err)
			os.Exit(1)
		}
	}

	if format == FormatJSON {
		jsonOutput, err := json.MarshalIndent(structGroups, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error encoding json: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	} else {
		totalStructCount := 0
		for _, group := range structGroups {
			totalStructCount += len(group)
		}

		fmt.Printf("Found %d struct(s) in %d package(s)\n\n", totalStructCount, len(structGroups))

		for groupIndex, group := range structGroups {
			if len(structGroups) > 1 {
				fmt.Printf("=== Package/File Group %d (%d structs) ===\n\n", groupIndex+1, len(group))
			}

			for i, s := range group {
				headerText := fmt.Sprintf(" STRUCT: %s ", s.Name)
				headerWidth := 74 // match box width
				leftPadding := (headerWidth - len(headerText)) / 2
				rightPadding := headerWidth - len(headerText) - leftPadding

				fmt.Println("┏" + strings.Repeat("━", headerWidth) + "┓")
				fmt.Println("┃" + strings.Repeat(" ", leftPadding) + headerText + strings.Repeat(" ", rightPadding) + "┃")
				fmt.Println("┗" + strings.Repeat("━", headerWidth) + "┛")

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

				fmt.Println("\n  " + strings.Repeat("▼", 35) + " OPTIMIZED " + strings.Repeat("▼", 35))

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
				if i < len(group)-1 {
					fmt.Println(strings.Repeat("■", 40))
					fmt.Println()
				}
			}

			if len(group) > 1 {
				outputSummary(group)
			}

			if len(structGroups) > 1 && groupIndex < len(structGroups)-1 {
				fmt.Println("\n" + strings.Repeat("=", 80) + "\n")
			}
		}

		if len(allStructs) > 1 {
			fmt.Println("\n" + strings.Repeat("=", 80))
			fmt.Println("OVERALL SUMMARY")
			fmt.Println(strings.Repeat("=", 80))
			outputOverallSummary(allStructs)
		}
	}
}

func outputSummary(structs []structi.Info) {
	fmt.Println("\n=== GROUP SUMMARY ===")
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

func outputOverallSummary(allStructs []structi.Info) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("=== OVERALL PROJECT SUMMARY ===")
	fmt.Printf("Total packages/files analyzed: %d\n", 1)
	fmt.Printf("Total structs analyzed: %d\n\n", len(allStructs))

	var highPotential, mediumPotential, lowPotential []structi.Info
	for _, s := range allStructs {
		savings := s.OriginalSize - s.OptimizedSize
		savingsPercent := float64(0)
		if s.OriginalSize > 0 {
			savingsPercent = float64(savings) / float64(s.OriginalSize) * 100
		}

		if savingsPercent >= 10 {
			highPotential = append(highPotential, s)
		} else if savingsPercent >= 5 {
			mediumPotential = append(mediumPotential, s)
		} else {
			lowPotential = append(lowPotential, s)
		}
	}

	var totalOriginalSize, totalOptimizedSize, totalWastedBytes int64
	for _, s := range allStructs {
		totalOriginalSize += s.OriginalSize
		totalOptimizedSize += s.OptimizedSize
		totalWastedBytes += s.WastedBytes
	}

	totalPotentialSavings := totalOriginalSize - totalOptimizedSize
	var totalWastedPercent, totalSavingsPercent float64
	if totalOriginalSize > 0 {
		totalWastedPercent = float64(totalWastedBytes) / float64(totalOriginalSize) * 100
		totalSavingsPercent = float64(totalPotentialSavings) / float64(totalOriginalSize) * 100
	}

	fmt.Println("Optimization potential:")
	fmt.Printf("  High potential (>= 10%% savings): %d structs\n", len(highPotential))
	fmt.Printf("  Medium potential (5-10%% savings): %d structs\n", len(mediumPotential))
	fmt.Printf("  Low potential (< 5%% savings): %d structs\n\n", len(lowPotential))

	fmt.Printf("Total original size: %d bytes\n", totalOriginalSize)
	fmt.Printf("Total optimized size: %d bytes\n", totalOptimizedSize)
	fmt.Printf("Total wasted bytes: %d (%.2f%%)\n", totalWastedBytes, totalWastedPercent)
	fmt.Printf("Total potential savings: %d bytes (%.2f%%)\n", totalPotentialSavings, totalSavingsPercent)

	if len(highPotential) > 0 {
		fmt.Println("\nTop structs with highest optimization potential:")
		fmt.Printf("%-20s %-15s %-15s %-15s\n",
			"Struct Name", "Original Size", "Potential Savings", "Savings %")
		fmt.Println(strings.Repeat("─", 70))

		sort.Slice(highPotential, func(i, j int) bool {
			iSavings := highPotential[i].OriginalSize - highPotential[i].OptimizedSize
			jSavings := highPotential[j].OriginalSize - highPotential[j].OptimizedSize
			iPercent := float64(iSavings) / float64(highPotential[i].OriginalSize) * 100
			jPercent := float64(jSavings) / float64(highPotential[j].OriginalSize) * 100
			return iPercent > jPercent
		})

		for i := 0; i < min(5, len(highPotential)); i++ {
			s := highPotential[i]
			savings := s.OriginalSize - s.OptimizedSize
			savingsPercent := float64(savings) / float64(s.OriginalSize) * 100
			fmt.Printf("%-20s %-15d %-15d %-15.2f\n",
				s.Name, s.OriginalSize, savings, savingsPercent)
		}
	}
}

func printUsage() {
	fmt.Println("Usage: ./viztruct [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --format string      Output format (json or txt) (default \"txt\")")
	fmt.Println("  --struct string      Inline struct definition")
	fmt.Println("  --file string        Path to file or directory containing struct definitions")
	fmt.Println("  --svg                Generate SVG visualization (default false)")
	fmt.Println("  --strategies string  Comma-separated list of optimization strategies to use")
	fmt.Println("                       Available strategies: alignment, size, group, greedy (default: all)")
	fmt.Println("  --concurrency int    Number of concurrent workers for analysis (0 for sequential)")
	fmt.Println("                       (default: use all available CPU cores)")
	fmt.Println("  --max-packages int   Maximum number of packages to analyze (0 for unlimited)")
	fmt.Println("                       (default: 500)")
	fmt.Println("  --skip-errors        Skip packages with errors instead of failing (default true)")
	fmt.Println("  --verbose            Enable verbose output with detailed warnings (default false)")
	fmt.Println("  --timeout int        Timeout in seconds for package loading (0 for no timeout)")
	fmt.Println("                       (default: 300)")
	fmt.Println("  --version            Show version information")
	fmt.Println("  --help               Show help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./viztruct --struct 'type MyStruct struct { a int; b string }'")
	fmt.Println("  ./viztruct --file structs.go")
	fmt.Println("  ./viztruct --file /path/to/project/dir")
	fmt.Println("  ./viztruct --strategies=\"greedy,group\" --file structs.go")
	fmt.Println("  ./viztruct --concurrency=4 --file /path/to/large/project")
	fmt.Println("  ./viztruct --format json --struct 'type MyStruct struct { a int; b string }'")
	fmt.Println("  ./viztruct --svg --struct 'type MyStruct struct { a int; b string }'")
	fmt.Println("  ./viztruct --max-packages=100 --timeout=60 --verbose --file /path/to/huge/project")
}

func main() {
	structArg := flag.String("struct", "", "Inline struct definition")
	fileArg := flag.String("file", "", "Path to file or directory containing struct definitions")
	formatArg := flag.String("format", string(FormatText), "Output format (json or txt)")
	svgArg := flag.Bool("svg", false, "Generate SVG visualization")
	versionArg := flag.Bool("version", false, "Show version information")
	helpArg := flag.Bool("help", false, "Show help message")
	strategiesArg := flag.String("strategies", "", "Comma-separated list of optimization strategies to use")
	concurrencyArg := flag.Int("concurrency", 0, "Number of concurrent workers for analysis (0 for sequential)")

	// Add new options for large codebases
	maxPackagesArg := flag.Int("max-packages", 1500, "Maximum number of packages to analyze (0 for unlimited)")
	skipErrorsArg := flag.Bool("skip-errors", true, "Skip packages with errors instead of failing")
	verboseArg := flag.Bool("verbose", false, "Enable verbose output with detailed warnings")
	timeoutArg := flag.Int("timeout", 300, "Timeout in seconds for package loading (0 for no timeout)")

	flag.Parse()

	if *helpArg {
		printUsage()
		return
	}

	if *versionArg {
		fmt.Printf("viztruct version %s\n", binVersion)
		return
	}

	format := OutputFormat(*formatArg)
	if format != FormatText && format != FormatJSON {
		fmt.Fprintf(os.Stderr, "error: invalid format: %s\n", *formatArg)
		printUsage()
		os.Exit(1)
	}

	// Handle strategy names
	var strategyNames []string
	if *strategiesArg != "" {
		strategyNames = strings.Split(*strategiesArg, ",")
		for i, name := range strategyNames {
			strategyNames[i] = strings.TrimSpace(name)
		}
	}

	// Configure global options based on flags
	if *maxPackagesArg > 0 {
		structi.SetMaxPackages(*maxPackagesArg)
	}

	structi.SetVerboseMode(*verboseArg)
	structi.SetSkipErrors(*skipErrorsArg)

	if *timeoutArg > 0 {
		structi.SetTimeout(*timeoutArg)
	}

	if *concurrencyArg > 0 {
		structi.SetConcurrency(*concurrencyArg)
	}

	if *structArg != "" {
		analyzeInlineStructs(*structArg, format, *svgArg, strategyNames)
	} else if *fileArg != "" {
		analyzeFromPath(*fileArg, format, *svgArg, strategyNames)
	} else {
		fmt.Fprintf(os.Stderr, "error: no struct definition provided\n")
		printUsage()
		os.Exit(1)
	}
}
