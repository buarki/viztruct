package structi

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
)

type Info struct {
	Name            string
	Type            *types.Struct
	OriginalSize    int64
	OptimizedSize   int64
	WastedBytes     int64
	WastedPercent   float64
	Fields          []Field
	OptimizedFields []Field
}

type Field struct {
	Name      string
	TypeName  string
	Offset    int64
	Size      int64
	Align     int64
	IsPadding bool
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

func AnalyzeNestedStructs(node *ast.File, sizes types.Sizes, info *types.Info, fset *token.FileSet) []Info {
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

		// Calculate sizes using the fields directly
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

	return structInfos
}
