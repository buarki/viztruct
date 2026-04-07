package structi

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

var (
	maxPackages = 1500  // Maximum number of packages to analyze
	verboseMode = false // Whether to print verbose output
	skipErrors  = true  // Whether to skip packages with errors
	timeout     = 300   // Timeout in seconds for package loading
)

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
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered from panic in calculateLayout: %v\n", r)
		}
	}()

	var fields []Field
	offset := int64(0)

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)

		if field.Type() == nil {
			fmt.Printf("warning: nil type found for field %s\n", field.Name())
			continue
		}

		size := int64(0)
		align := int64(1) // Default minimum alignment

		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("warning: failed to calculate size/alignment for field %s: %v\n",
						field.Name(), r)
				}
			}()
			size = sizes.Sizeof(field.Type())
			align = sizes.Alignof(field.Type())
		}()

		// skip fields with negative size (indicates error in size calculation)
		// zero-size fields (like struct{}) are valid and must be kept in the layout
		if size < 0 {
			fmt.Printf("warning: invalid size (%d) for field %s, skipping\n", size, field.Name())
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
					// let's just use default alignment if calculation fails
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

	// Check if the last field has zero size (e.g. struct{}).
	// Go adds padding equal to the struct's alignment in this case to prevent
	// pointers to the zero-size field from pointing beyond the allocation.
	lastFieldIsZeroSize := false
	if structType.NumFields() > 0 {
		lastField := structType.Field(structType.NumFields() - 1)
		if lastField.Type() != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// use default if size calculation fails
					}
				}()
				if sizes.Sizeof(lastField.Type()) == 0 {
					lastFieldIsZeroSize = true
				}
			}()
		}
	}

	if lastFieldIsZeroSize && offset > 0 {
		fields = append(fields, Field{
			Name:      "tail padding",
			TypeName:  "",
			Offset:    offset,
			Size:      structAlign,
			Align:     1,
			IsPadding: true,
		})
	} else if rem := offset % structAlign; rem != 0 {
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

		// set Fields before calling WastedSpace so it can iterate over them
		tempInfo.Fields = fields
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

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered from panic in collectStructs: %v\n", r)
		}
	}()

	ast.Inspect(node, func(n ast.Node) bool {
		// recover from panics in the inspection function
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("recovered from panic in struct inspection: %v\n", r)
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

		// process each struct with error handling
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("recovered from panic processing struct %s: %v\n",
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
	// set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recovered from panic in processStructWithStrategies for %s: %v\n", name, r)
		}
	}()

	tempInfo := Info{
		Name: name,
	}

	// calculate layout
	fields := tempInfo.calculateLayout(structType, &customSizes)

	// ff we got no fields (possibly due to error recovery), return early
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
		// just use original layout if optimization fails
		optimizedFields = fields
	}

	originalSize := calculateTotalSize(fields)
	optimizedSize := calculateTotalSize(optimizedFields)

	// set Fields before calling WastedSpace so it can iterate over them
	tempInfo.Fields = fields
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
		logf("loading packages with %d second timeout...\n", timeout)
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
		logf("loading packages (no timeout)...\n")
		return packages.Load(config, "./...")
	}
}

// AnalyseStructsAtDirectoryPath analyzes structs from a directory path with specified optimizers
func AnalyseStructsAtDirectoryPath(directoryPath string, strategyNames []string) ([][]Info, error) {
	s, err := os.Stat(directoryPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("error: directory %s does not exist", directoryPath)
	}

	if !s.IsDir() {
		return nil, fmt.Errorf("error: %s is not a directory", directoryPath)
	}

	defer func() {
		if r := recover(); r != nil {
			logf("recovered from panic in AnalyseStructsAtDirectoryPath: %v\n", r)
		}
	}()

	logf("loading packages from %s...\n", directoryPath)
	pkgs, err := loadPackagesFromDirectory(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	if maxPackages > 0 && len(pkgs) > maxPackages {
		fmt.Printf("Warning: Limiting analysis to %d packages out of %d total\n", maxPackages, len(pkgs))
		pkgs = pkgs[:maxPackages]
	}

	return processPackagesSequentially(pkgs, strategyNames)
}

// process packages one by one (sequential mode)
func processPackagesSequentially(pkgs []*packages.Package, strategyNames []string) ([][]Info, error) {
	var allInfos [][]Info

	for i, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			if skipErrors {
				logf("skipping package %s of path %s due to errors: %v\n", pkg.Name, pkg.PkgPath, pkg.Errors)
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
					logf("error processing file in package %s: %v\n", pkg.PkgPath, err)
					continue
				} else {
					return nil, fmt.Errorf("error processing package %s: %w", pkg.PkgPath, err)
				}
			}

			packageInfos = append(packageInfos, fileInfos...)
		}

		if len(packageInfos) > 0 {
			allInfos = append(allInfos, packageInfos)

			if verboseMode && i%10 == 0 {
				logf("processed %d/%d packages...\n", i+1, len(pkgs))
			}
		}
	}

	logf("analysis complete. Found structs in %d packages.\n", len(allInfos))
	return allInfos, nil
}
