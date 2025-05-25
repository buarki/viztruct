package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buarki/viztruct/structi"
)

// StructSummary stores summary information about analyzed structs
type StructSummary struct {
	Name             string
	FilePath         string
	OriginalSize     int64
	WastedBytes      int64
	WastedPercent    float64
	PotentialSavings int64
	SavingsPercent   float64
}

func main() {
	repoPath := flag.String("repo", ".", "Path to the Git repository")
	fromCommit := flag.String("from", "", "Starting commit hash (defaults to HEAD~1)")
	toCommit := flag.String("to", "", "Ending commit hash (defaults to HEAD)")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	analyzeStructs := flag.Bool("analyze", true, "Analyze struct changes in Go files")
	summaryOnly := flag.Bool("summary", false, "Show only the summary of all struct changes")

	flag.Parse()

	structi.SetVerboseMode(*verbose)
	structi.SetSkipErrors(true)

	changedFiles, err := getChangedFilesBetweenCommits(*repoPath, *fromCommit, *toCommit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	var goFiles []string
	for _, file := range changedFiles {
		if filepath.Ext(file) == ".go" {
			goFiles = append(goFiles, file)
		}
	}

	fmt.Println(">>>", goFiles)

	if len(goFiles) == 0 {
		fmt.Println("No Go files changed in this commit range")
		return
	}

	var allStructs []StructSummary

	analyzedDirs := make(map[string]bool)

	fmt.Printf("Found %d changed Go files:\n", len(goFiles))

	for _, file := range goFiles {
		if !*summaryOnly {
			fmt.Printf("\n- %s\n", file)

			if *verbose {
				absPath, err := filepath.Abs(filepath.Join(*repoPath, file))
				if err == nil {
					fmt.Printf("  Absolute path: %s\n", absPath)
				}
			}
		}

		if *analyzeStructs {
			filePath := filepath.Join(*repoPath, file)

			dirPath := filepath.Dir(filePath)

			directoryAlreadyAnalyzed := analyzedDirs[dirPath]
			if directoryAlreadyAnalyzed {
				if *verbose && !*summaryOnly {
					fmt.Printf("  Skipping already analyzed directory: %s\n", dirPath)
				}
				continue
			}

			analyzedDirs[dirPath] = true

			_, structSummaries := analyzeFile(filePath, *verbose && !*summaryOnly, *summaryOnly)
			allStructs = append(allStructs, structSummaries...)
		}
	}

	if len(allStructs) > 0 {
		printStructSummary(allStructs)
	} else if !*summaryOnly {
		fmt.Println("\nNo structs found in the analyzed files.")
	}
}

func analyzeFile(filePath string, verbose, quietMode bool) ([]structi.Info, []StructSummary) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if !quietMode {
			fmt.Printf("  Error: file does not exist: %s\n", filePath)
		}
		return nil, nil
	}

	dirPath := filepath.Dir(filePath)

	structGroups, err := structi.AnalyseStructsAtDirectoryPath(dirPath, nil)
	if err != nil {
		if verbose {
			fmt.Printf("  Error analyzing directory: %v\n", err)
		}
		return nil, nil
	}

	if len(structGroups) == 0 {
		if verbose {
			fmt.Printf("  No structs found in directory\n")
		}
		return nil, nil
	}

	var allStructs []structi.Info
	for _, group := range structGroups {
		allStructs = append(allStructs, group...)
	}

	if len(allStructs) == 0 {
		if verbose {
			fmt.Printf("  No structs found\n")
		}
		return nil, nil
	}

	var filteredStructs []structi.Info
	var summaries []StructSummary

	targetFile := filepath.Base(filePath)

	if !quietMode {
		fmt.Printf("  Found %d structs in directory, filtering for file %s\n", len(allStructs), targetFile)
	}

	for _, s := range allStructs {
		// Calculate waste percentage
		wastePercentage := 0.0
		if s.OriginalSize > 0 {
			wastePercentage = float64(s.WastedBytes) / float64(s.OriginalSize) * 100
		}

		// Calculate potential savings
		potentialSavings := s.OriginalSize - s.OptimizedSize
		savingsPercentage := 0.0
		if s.OriginalSize > 0 {
			savingsPercentage = float64(potentialSavings) / float64(s.OriginalSize) * 100
		}

		// For now, we'll include all structs from the directory
		// In a real CI plugin, you'd want to filter only structs from the changed file
		filteredStructs = append(filteredStructs, s)

		// Add to summary data
		summaries = append(summaries, StructSummary{
			Name:             s.Name,
			FilePath:         filePath, // We don't know the actual file for each struct
			OriginalSize:     s.OriginalSize,
			WastedBytes:      s.WastedBytes,
			WastedPercent:    wastePercentage,
			PotentialSavings: potentialSavings,
			SavingsPercent:   savingsPercentage,
		})

		if !quietMode {
			fmt.Printf("    • %s\n", s.Name)
			fmt.Printf("      - Size: %d bytes\n", s.OriginalSize)
			fmt.Printf("      - Wasted: %d bytes (%.2f%%)\n", s.WastedBytes, wastePercentage)

			if potentialSavings > 0 {
				fmt.Printf("      - Potential savings: %d bytes (%.2f%%)\n", potentialSavings, savingsPercentage)
			}

			if verbose {
				fmt.Printf("      - Fields: %d\n", len(s.Fields))
				for _, f := range s.Fields {
					if !f.IsPadding {
						fmt.Printf("        • %s: %s (%d bytes, align %d)\n",
							f.Name, f.TypeName, f.Size, f.Align)
					}
				}
			}
		}
	}

	return filteredStructs, summaries
}

// printStructSummary prints a summary of all analyzed structs
func printStructSummary(structs []StructSummary) {
	fmt.Println("\n====== STRUCT OPTIMIZATION SUMMARY ======")
	fmt.Printf("Total structs analyzed: %d\n\n", len(structs))

	// Sort structs by waste percentage
	sort.Slice(structs, func(i, j int) bool {
		return structs[i].WastedPercent > structs[j].WastedPercent
	})

	// Print header
	fmt.Printf("%-30s %-40s %-10s %-15s %-15s\n",
		"Struct Name", "File", "Size", "Wasted %", "Savings %")
	fmt.Println(strings.Repeat("-", 110))

	// Print each struct
	for _, s := range structs {
		// Get just the base file name for display
		baseFileName := filepath.Base(s.FilePath)

		// Truncate filename if too long
		if len(baseFileName) > 38 {
			baseFileName = "..." + baseFileName[len(baseFileName)-35:]
		}

		fmt.Printf("%-30s %-40s %-10d %-15.2f %-15.2f\n",
			truncateString(s.Name, 28),
			baseFileName,
			s.OriginalSize,
			s.WastedPercent,
			s.SavingsPercent)
	}

	// Calculate totals
	var totalSize, totalWasted, totalSavings int64
	for _, s := range structs {
		totalSize += s.OriginalSize
		totalWasted += s.WastedBytes
		totalSavings += s.PotentialSavings
	}

	// Calculate percentages
	wastedPercent := 0.0
	savingsPercent := 0.0
	if totalSize > 0 {
		wastedPercent = float64(totalWasted) / float64(totalSize) * 100
		savingsPercent = float64(totalSavings) / float64(totalSize) * 100
	}

	// Print totals
	fmt.Println(strings.Repeat("-", 110))
	fmt.Printf("%-30s %-40s %-10d %-15.2f %-15.2f\n",
		"TOTAL", "", totalSize, wastedPercent, savingsPercent)

	// Categorize structs
	var highWaste, mediumWaste, lowWaste int
	for _, s := range structs {
		if s.WastedPercent >= 15 {
			highWaste++
		} else if s.WastedPercent >= 5 {
			mediumWaste++
		} else {
			lowWaste++
		}
	}

	fmt.Println("\nWaste categories:")
	fmt.Printf("  High (>= 15%%): %d structs\n", highWaste)
	fmt.Printf("  Medium (5-15%%): %d structs\n", mediumWaste)
	fmt.Printf("  Low (< 5%%): %d structs\n", lowWaste)

	if totalSavings > 0 {
		fmt.Printf("\nOptimizing all structs could save %d bytes (%.2f%% of total size)\n",
			totalSavings, savingsPercent)
	}
}

// truncateString truncates a string if it's longer than the specified length
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

func getChangedFilesBetweenCommits(repoPath, fromCommit, toCommit string) ([]string, error) {
	if repoPath != "." {
		originalDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		if err := os.Chdir(repoPath); err != nil {
			return nil, fmt.Errorf("failed to change to repository directory %s: %w", repoPath, err)
		}

		defer func() {
			_ = os.Chdir(originalDir)
		}()
	}

	if fromCommit == "" {
		fromCommitCmd := exec.Command("git", "rev-parse", "HEAD~1")
		output, err := fromCommitCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get previous commit: %w", err)
		}
		fromCommit = strings.TrimSpace(string(output))
	}

	if toCommit == "" {
		toCommitCmd := exec.Command("git", "rev-parse", "HEAD")
		output, err := toCommitCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get current commit: %w", err)
		}
		toCommit = strings.TrimSpace(string(output))
	}

	diffCmd := exec.Command("git", "diff", "--name-only", fromCommit, toCommit)
	diffOutput, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(diffOutput) == 0 {
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(diffOutput)), "\n")
	return files, nil
}
