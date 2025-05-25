package structi

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"
)

// ExecutionMode defines how packages are processed
type ExecutionMode int

const (
	// Sequential processes packages one by one (single thread)
	Sequential ExecutionMode = iota

	// Concurrent processes packages in parallel (multiple threads)
	Concurrent

	// DependencyAware processes packages in parallel, but respects dependencies
	DependencyAware
)

// Configuration variables
var (
	maxPackages = 1500             // Maximum number of packages to analyze
	verboseMode = false            // Whether to print verbose output
	skipErrors  = true             // Whether to skip packages with errors
	timeout     = 300              // Timeout in seconds for package loading
	concurrency = runtime.NumCPU() // Number of concurrent workers
	execMode    = Concurrent       // Default execution mode
)

// SetExecutionMode sets the execution mode (Sequential or Concurrent)
func SetExecutionMode(mode ExecutionMode) {
	execMode = mode
	if mode == Sequential {
		concurrency = 1 // Force single worker for sequential mode
	} else {
		concurrency = 2 // runtime.NumCPU() // Reset to default for concurrent mode
	}
}

// GetExecutionMode returns the current execution mode
func GetExecutionMode() ExecutionMode {
	return execMode
}

// SetMaxPackages sets the maximum number of packages to analyze
func SetMaxPackages(max int) {
	if max <= 0 {
		maxPackages = 0 // No limit
	} else {
		maxPackages = max
	}
}

// SetVerboseMode enables or disables verbose output
func SetVerboseMode(verbose bool) {
	verboseMode = verbose
}

// SetSkipErrors sets whether to skip packages with errors
func SetSkipErrors(skip bool) {
	skipErrors = skip
}

// SetTimeout sets the timeout in seconds for package loading
func SetTimeout(seconds int) {
	if seconds <= 0 {
		timeout = 0 // No timeout
	} else {
		timeout = seconds
	}
}

// SetConcurrency sets the number of concurrent workers
func SetConcurrency(workers int) {
	if workers <= 0 {
		concurrency = runtime.NumCPU()
	} else {
		concurrency = workers
	}
}

// logf prints a message if verbose mode is enabled
func logf(format string, args ...interface{}) {
	if verboseMode {
		fmt.Printf(format, args...)
	}
}

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("type from [%s] package is undefined. Check your imports or provide the type definition as well.", e.Message)
}

var (
	customSizes = types.StdSizes{
		WordSize: 8,
		MaxAlign: 8,
	}

	ErrNoOptimizers = errors.New("at least one optimizer is required")
)

type Info struct {
	Name            string        `json:"name"`
	Type            *types.Struct `json:"type,omitempty,omitzero"`
	OriginalSize    int64         `json:"original_size"`
	OptimizedSize   int64         `json:"optimized_size"`
	WastedBytes     int64         `json:"wasted_bytes"`
	WastedPercent   float64       `json:"wasted_percent"`
	Fields          []Field       `json:"fields"`
	OptimizedFields []Field       `json:"optimized_fields"`
}

type Field struct {
	Name      string `json:"name"`
	TypeName  string `json:"type,omitempty,omitzero"`
	Offset    int64  `json:"offset"`
	Size      int64  `json:"size"`
	Align     int64  `json:"align"`
	IsPadding bool   `json:"is_padding"`
}

type fieldWithMeta struct {
	name  string
	typ   types.Type
	size  int64
	align int64
}

func typeName(t types.Type) string {
	return t.String()
}

func (i Info) TotalSize() int64 {
	if len(i.Fields) == 0 {
		return 0
	}
	last := i.Fields[len(i.Fields)-1]
	return last.Offset + last.Size
}

func (i Info) OptimizedTotalSize() int64 {
	if len(i.OptimizedFields) == 0 {
		return 0
	}
	last := i.OptimizedFields[len(i.OptimizedFields)-1]
	return last.Offset + last.Size
}

func (i Info) WastedSpace() (int64, float64) {
	var wastedBytes int64
	for _, f := range i.Fields {
		if f.IsPadding {
			wastedBytes += f.Size
		}
	}

	totalSize := i.TotalSize()
	if totalSize == 0 {
		return 0, 0
	}

	wastedPercent := float64(wastedBytes) / float64(totalSize) * 100
	return wastedBytes, wastedPercent
}

func (i Info) OptimizedWastedSpace() (int64, float64) {
	var wastedBytes int64
	for _, f := range i.OptimizedFields {
		if f.IsPadding {
			wastedBytes += f.Size
		}
	}

	totalSize := i.OptimizedTotalSize()
	if totalSize == 0 {
		return 0, 0
	}

	wastedPercent := float64(wastedBytes) / float64(totalSize) * 100
	return wastedBytes, wastedPercent
}

func (i Info) calculateLayout(structType *types.Struct, sizes types.Sizes) []Field {
	// Set up panic recovery to handle type system errors
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in calculateLayout: %v\n", r)
			// We'll return an empty layout rather than crashing
		}
	}()

	var fields []Field
	offset := int64(0)

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)

		// Check for nil type to avoid panic
		if field.Type() == nil {
			fmt.Printf("Warning: Nil type found for field %s\n", field.Name())
			continue
		}

		// Safely calculate size and alignment with panic protection
		size := int64(0)
		align := int64(1) // Default minimum alignment

		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Warning: Failed to calculate size/alignment for field %s: %v\n",
						field.Name(), r)
				}
			}()
			size = sizes.Sizeof(field.Type())
			align = sizes.Alignof(field.Type())
		}()

		// Skip fields with zero or negative size (indicates error in size calculation)
		if size <= 0 {
			fmt.Printf("Warning: Invalid size (%d) for field %s, skipping\n", size, field.Name())
			continue
		}

		// add padding if needed
		if rem := offset % align; rem != 0 {
			paddingSize := align - rem
			fields = append(fields, Field{
				Name:      "padding",
				TypeName:  "",
				Offset:    offset,
				Size:      paddingSize,
				Align:     1,
				IsPadding: true,
			})
			offset += paddingSize
		}

		fields = append(fields, Field{
			Name:      field.Name(),
			TypeName:  typeName(field.Type()),
			Offset:    offset,
			Size:      size,
			Align:     align,
			IsPadding: false,
		})

		offset += size
	}

	// adding final padding for struct alignment
	structAlign := int64(1)
	for i := 0; i < structType.NumFields(); i++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Just use default alignment if calculation fails
				}
			}()

			field := structType.Field(i)
			if field.Type() != nil {
				fieldAlign := sizes.Alignof(field.Type())
				if fieldAlign > structAlign {
					structAlign = fieldAlign
				}
			}
		}()
	}

	if rem := offset % structAlign; rem != 0 {
		paddingSize := structAlign - rem
		fields = append(fields, Field{
			Name:      "tail padding",
			TypeName:  "",
			Offset:    offset,
			Size:      paddingSize,
			Align:     1,
			IsPadding: true,
		})
	}

	return fields
}

func (i Info) optimizeStructLayoutWithStrategies(structType *types.Struct, sizes types.Sizes, optimizers []FieldOptimizer) ([]Field, error) {
	if len(optimizers) == 0 {
		return nil, ErrNoOptimizers
	}

	var fields []fieldWithMeta
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		size := sizes.Sizeof(field.Type())
		align := sizes.Alignof(field.Type())
		fields = append(fields, fieldWithMeta{
			name:  field.Name(),
			typ:   field.Type(),
			size:  size,
			align: align,
		})
	}

	// try optimizing with the first optimizer
	bestLayout := optimizers[0].Optimize(fields)
	bestTotalSize := calculateTotalSize(bestLayout)

	// try other optimizers and keep the best result
	for i := 1; i < len(optimizers); i++ {
		layout := optimizers[i].Optimize(fields)
		totalSize := calculateTotalSize(layout)

		if totalSize < bestTotalSize {
			bestLayout = layout
			bestTotalSize = totalSize
		}
	}

	return bestLayout, nil
}

// calculate the total size of a struct by looking at the last field's offset and size
func calculateTotalSize(layout []Field) int64 {
	if len(layout) == 0 {
		return 0
	}
	last := layout[len(layout)-1]
	return last.Offset + last.Size
}

func analyzeNestedStructs(node *ast.File, sizes types.Sizes, info *types.Info, strategyNames []string) ([]Info, error) {
	var structInfos []Info

	var processingError error

	// find all struct declarations including nested ones
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true // continue traversing
		}

		if _, ok = typeSpec.Type.(*ast.StructType); !ok {
			return true // not a struct, continue
		}

		// get the type info
		typeObj := info.Defs[typeSpec.Name]
		if typeObj == nil {
			return true // no type info available
		}

		// get the underlying struct type
		underlyingType, ok := typeObj.Type().Underlying().(*types.Struct)
		if !ok {
			return true // not a struct type
		}

		tempInfo := Info{}
		fields := tempInfo.calculateLayout(underlyingType, sizes)
		optimazers, err := collectFieldOptmizers(strategyNames)
		if err != nil {
			processingError = err
			return false
		}
		optimizedFields, err := tempInfo.optimizeStructLayoutWithStrategies(underlyingType, sizes, optimazers)
		if err != nil {
			processingError = err
			return false
		}

		originalSize := calculateTotalSize(fields)
		optimizedSize := calculateTotalSize(optimizedFields)

		wastedBytes, wastedPercent := tempInfo.WastedSpace()

		structInfo := Info{
			Name:            typeSpec.Name.Name,
			Type:            underlyingType,
			OriginalSize:    originalSize,
			OptimizedSize:   optimizedSize,
			WastedBytes:     wastedBytes,
			WastedPercent:   wastedPercent,
			Fields:          fields,
			OptimizedFields: optimizedFields,
		}

		structInfos = append(structInfos, structInfo)
		return true
	})

	if processingError != nil {
		return nil, processingError
	}

	return structInfos, nil
}

func collectStructs(node *ast.File, info *types.Info, strategyNames []string) ([]Info, error) {
	var result []Info
	var processingErrors error

	// Set up panic recovery for the whole function
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in collectStructs: %v\n", r)
		}
	}()

	ast.Inspect(node, func(n ast.Node) bool {
		// Recover from panics in the inspection function
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic in struct inspection: %v\n", r)
			}
		}()

		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true // continue traversing
		}

		_, isStruct := typeSpec.Type.(*ast.StructType)
		if !isStruct {
			return true
		}

		// get the type info
		typeObj := info.Defs[typeSpec.Name]
		if typeObj == nil {
			return true // no type info available
		}

		// get the underlying struct type
		underlyingType, ok := typeObj.Type().Underlying().(*types.Struct)
		if !ok {
			return true // not a struct type
		}

		// Process each struct with error handling
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Recovered from panic processing struct %s: %v\n",
						typeSpec.Name.Name, r)
				}
			}()

			structInfo, err := processStructWithStrategies(typeSpec.Name.Name, underlyingType, strategyNames)
			if err != nil {
				processingErrors = err
				return
			}
			result = append(result, structInfo)
		}()

		return true
	})

	if processingErrors != nil {
		return nil, processingErrors
	}

	return result, nil
}

func collectFieldOptmizers(strategyNames []string) ([]FieldOptimizer, error) {
	var optimizers []FieldOptimizer
	if len(strategyNames) == 0 {
		return GetAllOptimizers(), nil
	}
	optimizers, err := GetOptimizersByNames(strategyNames)
	if err != nil {
		return nil, err
	}
	return optimizers, nil
}

// process a struct and calculate its layout with specified optimizers
func processStructWithStrategies(name string, structType *types.Struct, strategyNames []string) (Info, error) {
	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in processStructWithStrategies for %s: %v\n", name, r)
		}
	}()

	tempInfo := Info{
		Name: name,
	}

	// calculate layout
	fields := tempInfo.calculateLayout(structType, &customSizes)

	// If we got no fields (possibly due to error recovery), return early
	if len(fields) == 0 {
		return Info{
			Name: name,
			Type: structType,
		}, nil
	}

	optimizers, err := collectFieldOptmizers(strategyNames)
	if err != nil {
		return Info{}, err
	}

	optimizedFields, err := tempInfo.optimizeStructLayoutWithStrategies(structType, &customSizes, optimizers)
	if err != nil {
		// Just use original layout if optimization fails
		optimizedFields = fields
	}

	originalSize := calculateTotalSize(fields)
	optimizedSize := calculateTotalSize(optimizedFields)

	wastedBytes, wastedPercent := tempInfo.WastedSpace()

	return Info{
		Name:            name,
		Type:            structType,
		OriginalSize:    originalSize,
		OptimizedSize:   optimizedSize,
		WastedBytes:     wastedBytes,
		WastedPercent:   wastedPercent,
		Fields:          fields,
		OptimizedFields: optimizedFields,
	}, nil
}

// AnalyseStructsAsStringWithStrategies analyzes structs from a string with specified optimizers
func AnalyseStructsAsStringWithStrategies(structsSource string, strategyNames []string) ([]Info, error) {
	if !strings.Contains(structsSource, "package") {
		structsSource = "package temp\n\n" + structsSource
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "input.go", structsSource, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %v", err)
	}

	conf := types.Config{Importer: nil, Sizes: &customSizes}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	if _, err = conf.Check("temp", fset, []*ast.File{node}, info); err != nil {
		if strings.Contains(err.Error(), "undefined:") {
			errParts := strings.Split(err.Error(), "undefined:")
			unknownPackage := errParts[len(errParts)-1]
			return nil, &Error{Message: strings.TrimSpace(unknownPackage)}
		}
		return nil, &Error{fmt.Sprintf("failed to type-check: %v", err)}
	}

	return analyzeNestedStructs(node, &customSizes, info, strategyNames)
}

// load all packages in a directory with auto-fetching dependencies
func loadPackagesFromDirectory(dirPath string) ([]*packages.Package, error) {
	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedSyntax,
		Dir:  dirPath,
	}

	if timeout > 0 {
		logf("Loading packages with %d second timeout...\n", timeout)
		done := make(chan struct{})
		var pkgs []*packages.Package
		var loadErr error

		go func() {
			pkgs, loadErr = packages.Load(config, "./...")
			close(done)
		}()

		select {
		case <-done:
			logf("Loaded %d packages\n", len(pkgs))
			return pkgs, loadErr
		case <-time.After(time.Duration(timeout) * time.Second):
			return nil, fmt.Errorf("package loading timed out after %d seconds", timeout)
		}
	} else {
		logf("Loading packages (no timeout)...\n")
		return packages.Load(config, "./...")
	}
}

// AnalyseStructsAtDirectoryPath analyzes structs from a directory path with specified optimizers
func AnalyseStructsAtDirectoryPath(directoryPath string, strategyNames []string) ([][]Info, error) {
	// Set up panic recovery for the main function
	defer func() {
		if r := recover(); r != nil {
			logf("Recovered from panic in AnalyseStructsAtDirectoryPath: %v\n", r)
		}
	}()

	logf("Loading packages from %s...\n", directoryPath)
	pkgs, err := loadPackagesFromDirectory(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}
	fmt.Printf(">>> packages %d\n", len(pkgs))

	// For very large codebases, limit the number of packages processed
	if maxPackages > 0 && len(pkgs) > maxPackages {
		fmt.Printf("Warning: Limiting analysis to %d packages out of %d total\n", maxPackages, len(pkgs))
		pkgs = pkgs[:maxPackages]
	}

	// Execute based on execution mode
	switch execMode {
	case Sequential:
		return processPackagesSequentially(pkgs, strategyNames)
	case Concurrent:
		return processPackagesConcurrently(pkgs, strategyNames)
	case DependencyAware:
		return processPackagesWithDependencies(pkgs, strategyNames)
	default:
		// Default to concurrent mode
		return processPackagesConcurrently(pkgs, strategyNames)
	}
}

// sequential mode
func processPackagesSequentially(pkgs []*packages.Package, strategyNames []string) ([][]Info, error) {
	var allInfos [][]Info

	fmt.Printf("Processing packages sequentially...\n")

	for i, pkg := range pkgs {
		// skip packages with errors if configured to do so
		if len(pkg.Errors) > 0 {
			if skipErrors {
				logf("Skipping package %s due to errors\n", pkg.PkgPath)
				continue
			} else {
				return nil, fmt.Errorf("error in package %s: %v", pkg.PkgPath, pkg.Errors[0])
			}
		}

		// skip packages with no syntax
		if len(pkg.Syntax) == 0 {
			continue
		}

		var packageInfos []Info

		// process each file in the package
		for _, syntax := range pkg.Syntax {
			fileInfos, err := collectStructs(syntax, pkg.TypesInfo, strategyNames)
			if err != nil {
				if skipErrors {
					logf("Error processing file in package %s: %v\n", pkg.PkgPath, err)
					continue
				} else {
					return nil, fmt.Errorf("error processing package %s: %w", pkg.PkgPath, err)
				}
			}

			packageInfos = append(packageInfos, fileInfos...)
		}

		// if we found any structs, add them to the results
		if len(packageInfos) > 0 {
			allInfos = append(allInfos, packageInfos)

			if verboseMode && i%10 == 0 {
				logf("Processed %d/%d packages...\n", i+1, len(pkgs))
			}
		}
	}

	logf("Analysis complete. Found structs in %d packages.\n", len(allInfos))
	return allInfos, nil
}

// parallel mode
func processPackagesConcurrently(pkgs []*packages.Package, strategyNames []string) ([][]Info, error) {
	var allInfos [][]Info
	var allInfosMutex sync.Mutex // protect concurrent access to allInfos

	// semaphore to limit concurrent goroutines based on the concurrency setting
	semaphore := make(chan struct{}, concurrency)

	var wg sync.WaitGroup
	wg.Add(len(pkgs))

	errChan := make(chan error, len(pkgs))

	logf("Processing packages concurrently with %d workers...\n", concurrency)
	for i, pkg := range pkgs {
		go func(i int, pkg *packages.Package) {
			// cquire semaphore slot
			semaphore <- struct{}{}
			defer func() {
				// release semaphore slot
				<-semaphore
				wg.Done()

				if r := recover(); r != nil {
					logf("Recovered from panic in package processing goroutine (%s): %v\n",
						pkg.PkgPath, r)
				}
			}()

			// skip packages with errors if configured to do so
			if len(pkg.Errors) > 0 {
				if skipErrors {
					logf("Skipping package %s due to errors\n", pkg.PkgPath)
					return
				} else {
					errChan <- fmt.Errorf("error in package %s: %v", pkg.PkgPath, pkg.Errors[0])
					return
				}
			}

			// skip packages with no syntax
			if len(pkg.Syntax) == 0 {
				return
			}

			var packageInfos []Info

			// process each file in the package
			for _, syntax := range pkg.Syntax {
				// handle file processing errors gracefully
				func() {
					defer func() {
						if r := recover(); r != nil {
							logf("Recovered from panic processing file in package %s: %v\n",
								pkg.PkgPath, r)
						}
					}()

					fileInfos, err := collectStructs(syntax, pkg.TypesInfo, strategyNames)
					if err != nil {
						if skipErrors {
							logf("Error processing package %s: %v\n", pkg.PkgPath, err)
							return
						} else {
							errChan <- fmt.Errorf("error processing package %s: %w", pkg.PkgPath, err)
							return
						}
					}

					// only append if we found structs
					if len(fileInfos) > 0 {
						packageInfos = append(packageInfos, fileInfos...)
					}
				}()
			}

			// if we found any structs, add them to the results
			if len(packageInfos) > 0 {
				allInfosMutex.Lock()
				allInfos = append(allInfos, packageInfos)
				allInfosMutex.Unlock()

				if verboseMode && i%10 == 0 {
					logf("Processed %d/%d packages...\n", i+1, len(pkgs))
				}
			}
		}(i, pkg)
	}

	// wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// check for errors if we're not skipping them
	if !skipErrors {
		select {
		case err := <-errChan:
			return nil, err
		default:
			// no errors
		}
	} else {
		// log errors but don't fail
		var errList []string
		for err := range errChan {
			errList = append(errList, err.Error())
		}

		if len(errList) > 0 {
			logf("Warning: Encountered %d errors during processing\n", len(errList))
		}
	}

	logf("Analysis complete. Found structs in %d packages.\n", len(allInfos))
	return allInfos, nil
}
