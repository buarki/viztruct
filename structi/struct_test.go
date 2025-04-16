package structi

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"math"
	"testing"
)

func TestGetTotalSize(t *testing.T) {
	tests := []struct {
		name   string
		fields []Field
		want   int64
	}{
		{
			name:   "empty slice",
			fields: []Field{},
			want:   0,
		},
		{
			name: "one field",
			fields: []Field{
				{Offset: 0, Size: 10},
			},
			want: 10,
		},
		{
			name: "multiple fields",
			fields: []Field{
				{Offset: 0, Size: 5},
				{Offset: 5, Size: 10},
				{Offset: 15, Size: 20},
			},
			want: 35, // 15 + 20
		},
		{
			name: "non-contiguous fields",
			fields: []Field{
				{Offset: 0, Size: 5},
				{Offset: 10, Size: 3},
				{Offset: 20, Size: 2},
			},
			want: 22, // 20 + 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTotalSize(tt.fields)
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestCalcWastedSpace(t *testing.T) {
	tests := []struct {
		name         string
		fields       []Field
		wantBytes    int64
		wantPercent  float64
	}{
		{
			name:        "no fields",
			fields:      []Field{},
			wantBytes:   0,
			wantPercent: 0,
		},
		{
			name: "no padding",
			fields: []Field{
				{Offset: 0, Size: 4, IsPadding: false},
				{Offset: 4, Size: 4, IsPadding: false},
			},
			wantBytes:   0,
			wantPercent: 0,
		},
		{
			name: "all padding",
			fields: []Field{
				{Offset: 0, Size: 4, IsPadding: true},
				{Offset: 4, Size: 4, IsPadding: true},
			},
			wantBytes:   8,
			wantPercent: 100,
		},
		{
			name: "mixed fields",
			fields: []Field{
				{Offset: 0, Size: 4, IsPadding: false},
				{Offset: 4, Size: 2, IsPadding: true},
				{Offset: 6, Size: 2, IsPadding: false},
				{Offset: 8, Size: 4, IsPadding: true},
			},
			wantBytes:   6,
			wantPercent: 50, // total size is 12, 6/12 = 0.5 = 50%
		},
	}

	const floatTolerance = 1e-6

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBytes, gotPercent := calcWastedSpace(tt.fields)

			if gotBytes != tt.wantBytes {
				t.Errorf("wastedBytes = %d, want %d", gotBytes, tt.wantBytes)
			}

			if math.Abs(gotPercent-tt.wantPercent) > floatTolerance {
				t.Errorf("wastedPercent = %f, want %f", gotPercent, tt.wantPercent)
			}
		})
	}
}


func TestCalculateLayout(t *testing.T) {
	const src = `
package test

type MyStruct struct {
	A int8
	B int32
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("test", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("type check error: %v", err)
	}

	obj := pkg.Scope().Lookup("MyStruct")
	if obj == nil {
		t.Fatal("type MyStruct not found")
	}

	structType, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		t.Fatal("not a struct type")
	}

	sizes := types.StdSizes{WordSize: 8, MaxAlign: 8}
	fields := calculateLayout(structType, &sizes)

	expected := []Field{
		{Name: "A", Offset: 0, Size: 1, Align: 1, IsPadding: false},
		{Name: "padding", Offset: 1, Size: 3, Align: 1, IsPadding: true},
		{Name: "B", Offset: 4, Size: 4, Align: 4, IsPadding: false},
	}

	if len(fields) != len(expected) {
		t.Fatalf("unexpected field count: got %d, want %d", len(fields), len(expected))
	}

	for i, f := range fields {
		exp := expected[i]
		if f.Name != exp.Name || f.Offset != exp.Offset || f.Size != exp.Size || f.IsPadding != exp.IsPadding {
			t.Errorf("field[%d] = %+v, want %+v", i, f, exp)
		}
	}
}


func TestOptimizeStructLayout(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		typeName string
		expected []Field
	}{
		{
			name: "int8 then int32",
			src: `
				package test
				type MyStruct struct {
					A int8
					B int32
				}
			`,
			typeName: "MyStruct",
			expected: []Field{
				{Name: "B", Offset: 0, Size: 4, Align: 4, IsPadding: false},
				{Name: "A", Offset: 4, Size: 1, Align: 1, IsPadding: false},
				{Name: "tail padding", Offset: 5, Size: 3, Align: 1, IsPadding: true},
			},
		},
		{
			name: "three mixed types",
			src: `
				package test
				type MyStruct struct {
					A int8
					B int64
					C int32
				}
			`,
			typeName: "MyStruct",
			expected: []Field{
				{Name: "B", Offset: 0, Size: 8, Align: 8, IsPadding: false},
				{Name: "C", Offset: 8, Size: 4, Align: 4, IsPadding: false},
				{Name: "A", Offset: 12, Size: 1, Align: 1, IsPadding: false},
				{Name: "tail padding", Offset: 13, Size: 3, Align: 1, IsPadding: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "src.go", tt.src, 0)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			conf := types.Config{Importer: importer.Default()}
			pkg, err := conf.Check("test", fset, []*ast.File{node}, nil)
			if err != nil {
				t.Fatalf("type check error: %v", err)
			}

			obj := pkg.Scope().Lookup(tt.typeName)
			if obj == nil {
				t.Fatalf("type %s not found", tt.typeName)
			}

			structType, ok := obj.Type().Underlying().(*types.Struct)
			if !ok {
				t.Fatalf("%s is not a struct", tt.typeName)
			}

			sizes := types.StdSizes{WordSize: 8, MaxAlign: 8}
			fields := optimizeStructLayout(structType, &sizes)

			if len(fields) != len(tt.expected) {
				t.Fatalf("unexpected field count: got %d, want %d", len(fields), len(tt.expected))
			}

			for i, f := range fields {
				exp := tt.expected[i]
				if f.Name != exp.Name || f.Offset != exp.Offset || f.Size != exp.Size || f.IsPadding != exp.IsPadding {
					t.Errorf("field[%d] = %+v, want %+v", i, f, exp)
				}
			}
		})
	}
}


func TestAnalyzeNestedStructs(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		expectedInfo []string // Expected struct names (including nested ones)
	}{
		{
			name: "single top-level struct",
			src: `
				package test
				type A struct {
					X int8
					Y int32
				}
			`,
			expectedInfo: []string{"A"},
		},
		{
			name: "struct with nested struct",
			src: `
				package test
				type B struct {
					A int8
					Inner struct {
						P int64
						Q int8
					}
				}
			`,
			expectedInfo: []string{"B"},
		},
		{
			name: "multiple structs",
			src: `
				package test
				type A struct { X int8 }
				type B struct { Y int64 }
				type C struct { Z float32 }
			`,
			expectedInfo: []string{"A", "B", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "src.go", tt.src, parser.AllErrors)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			conf := types.Config{Importer: importer.Default()}
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
			}

			_, err = conf.Check("test", fset, []*ast.File{node}, info)
			if err != nil {
				t.Fatalf("type check error: %v", err)
			}

			sizes := types.StdSizes{WordSize: 8, MaxAlign: 8}
			results := AnalyzeNestedStructs(node, &sizes, info, fset)

			if len(results) != len(tt.expectedInfo) {
				t.Fatalf("unexpected number of structs: got %d, want %d", len(results), len(tt.expectedInfo))
			}

			for i, expectedName := range tt.expectedInfo {
				if results[i].Name != expectedName {
					t.Errorf("struct[%d] name = %s, want %s", i, results[i].Name, expectedName)
				}

				if results[i].OriginalSize == 0 {
					t.Errorf("struct[%d] original size should be > 0", i)
				}
				if results[i].OptimizedSize == 0 {
					t.Errorf("struct[%d] optimized size should be > 0", i)
				}
			}
		})
	}
}




