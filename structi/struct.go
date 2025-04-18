package structi

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

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

func (i Info) OptimazedTotalSize() int64 {
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

func (i Info) OptimazedWastedSpace() (int64, float64) {
	var wastedBytes int64
	for _, f := range i.OptimizedFields {
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

func (i Info) calculateLayout(structType *types.Struct, sizes types.Sizes) []Field {
	var fields []Field
	offset := int64(0)

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		size := sizes.Sizeof(field.Type())
		align := sizes.Alignof(field.Type())

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
		fieldAlign := sizes.Alignof(structType.Field(i).Type())
		if fieldAlign > structAlign {
			structAlign = fieldAlign
		}
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

func (i Info) optimizeStructLayout(structType *types.Struct, sizes types.Sizes) []Field {
	type fieldWithMeta struct {
		name  string
		typ   types.Type
		size  int64
		align int64
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

	// sort fields by alignment (descending) and then by size (descending)
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].align != fields[j].align {
			return fields[i].align > fields[j].align
		}
		return fields[i].size > fields[j].size
	})

	// calculate offsets for optimized layout
	var optimizedFields []Field
	var offset int64 = 0

	for _, f := range fields {
		// align field
		if rem := offset % f.align; rem != 0 {
			paddingSize := f.align - rem
			optimizedFields = append(optimizedFields, Field{
				Name:      "padding",
				TypeName:  "",
				Offset:    offset,
				Size:      paddingSize,
				Align:     1,
				IsPadding: true,
			})
			offset += paddingSize
		}

		optimizedFields = append(optimizedFields, Field{
			Name:      f.name,
			TypeName:  typeName(f.typ),
			Offset:    offset,
			Size:      f.size,
			Align:     f.align,
			IsPadding: false,
		})

		offset += f.size
	}

	// add final padding for struct alignment
	var structAlign int64 = 1
	for _, f := range fields {
		if f.align > structAlign {
			structAlign = f.align
		}
	}

	if rem := offset % structAlign; rem != 0 {
		paddingSize := structAlign - rem
		optimizedFields = append(optimizedFields, Field{
			Name:      "tail padding",
			TypeName:  "",
			Offset:    offset,
			Size:      paddingSize,
			Align:     1,
			IsPadding: true,
		})
	}

	return optimizedFields
}

func AnalyseStructs(structsSource string) ([]Info, error) {
	// just prepend package declaration if needed
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

	return analyzeNestedStructs(node, &customSizes, info, fset)
}

func analyzeNestedStructs(node *ast.File, sizes types.Sizes, info *types.Info, fset *token.FileSet) ([]Info, error) {
	var structInfos []Info

	// find all struct declarations including nested ones
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true // continue traversing
		}

		_, ok = typeSpec.Type.(*ast.StructType)
		if !ok {
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
		optimizedFields := tempInfo.optimizeStructLayout(underlyingType, sizes)

		// calculate sizes using the fields directly
		originalSize := int64(0)
		if len(fields) > 0 {
			last := fields[len(fields)-1]
			originalSize = last.Offset + last.Size
		}

		optimizedSize := int64(0)
		if len(optimizedFields) > 0 {
			last := optimizedFields[len(optimizedFields)-1]
			optimizedSize = last.Offset + last.Size
		}

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

	return structInfos, nil
}
