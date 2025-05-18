package structi

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestTotalSize(t *testing.T) {
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
			info := Info{Fields: tt.fields}
			got := info.TotalSize()
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestWastedSpace(t *testing.T) {
	tests := []struct {
		name        string
		fields      []Field
		wantBytes   int64
		wantPercent float64
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
			info := Info{Fields: tt.fields}
			gotBytes, gotPercent := info.WastedSpace()

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
	info := Info{}
	fields := info.calculateLayout(structType, &sizes)

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

			results, err := AnalyseStructsWithStrategies(tt.src, nil)
			if err != nil {
				t.Fatalf("error analyzing nested structs: %v", err)
			}

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

func TestAnalyseStructs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, infos []Info)
	}{
		{
			name: "basic struct",
			input: `type Basic struct {
				A int64
				B int32
				C bool
			}`,
			validate: func(t *testing.T, infos []Info) {
				if len(infos) != 1 {
					t.Fatalf("expected 1 struct, got %d", len(infos))
				}
				if infos[0].Name != "Basic" {
					t.Errorf("expected struct name 'Basic', got %q", infos[0].Name)
				}
				var nonPaddingFields int
				for _, f := range infos[0].Fields {
					if !f.IsPadding {
						nonPaddingFields++
					}
				}
				if nonPaddingFields != 3 {
					t.Errorf("expected 3 non-padding fields, got %d", nonPaddingFields)
				}
			},
		},
		{
			name: "multiple structs",
			input: `
			type First struct {
				X int32
			}
			type Second struct {
				Y int64
			}`,
			validate: func(t *testing.T, infos []Info) {
				if len(infos) != 2 {
					t.Fatalf("expected 2 structs, got %d", len(infos))
				}
				names := []string{infos[0].Name, infos[1].Name}
				sort.Strings(names)
				if !reflect.DeepEqual(names, []string{"First", "Second"}) {
					t.Errorf("expected struct names [First Second], got %v", names)
				}
			},
		},
		{
			name: "struct with undefined type",
			input: `type HasUndefined struct {
				Field types.Struct
			}`,
			wantErr:     true,
			errContains: "type from [types] package is undefined",
		},
		{
			name: "invalid syntax",
			input: `type Invalid struct {
				Field: string // using colon instead of space
			}`,
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name: "struct with padding",
			input: `type NeedsPadding struct {
				A bool    // 1 byte
				B int64   // 8 bytes, needs 7 bytes padding
				C int32   // 4 bytes
			}`,
			validate: func(t *testing.T, infos []Info) {
				if len(infos) != 1 {
					t.Fatalf("expected 1 struct, got %d", len(infos))
				}
				info := infos[0]

				// should have 5 fields including padding
				if len(info.Fields) != 5 {
					t.Errorf("expected 5 fields (including padding), got %d", len(info.Fields))
				}

				// check for padding fields
				var paddingCount int
				for _, f := range info.Fields {
					if f.IsPadding {
						paddingCount++
					}
				}
				if paddingCount != 2 {
					t.Errorf("expected 2 padding fields, got %d", paddingCount)
				}

				// original size should be 24 bytes (1 + 7pad + 8 + 4 + 4pad)
				if info.OriginalSize != 24 {
					t.Errorf("expected original size of 24 bytes, got %d", info.OriginalSize)
				}
			},
		},
		{
			name: "optimized struct layout",
			input: `type NeedsOptimization struct {
				A bool    // 1 byte
				B int64   // 8 bytes
				C int32   // 4 bytes
				D bool    // 1 byte
			}`,
			validate: func(t *testing.T, infos []Info) {
				if len(infos) != 1 {
					t.Fatalf("expected 1 struct, got %d", len(infos))
				}
				info := infos[0]

				// Original layout should be larger than optimized
				if info.OriginalSize <= info.OptimizedSize {
					t.Errorf("expected original size (%d) to be larger than optimized size (%d)",
						info.OriginalSize, info.OptimizedSize)
				}

				// Check field order in optimized layout
				var lastField string
				for _, f := range info.OptimizedFields {
					if !f.IsPadding {
						lastField = f.Name
					}
				}
				if lastField != "A" && lastField != "D" { // bools should be last in optimized layout
					t.Errorf("expected bool fields to be last in optimized layout, got %q", lastField)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := AnalyseStructsWithStrategies(tt.input, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, results)
			}
		})
	}
}

func TestAnalyseFromFile(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_struct.go")

	testFileContent := `package test

import (
	"time"
)

type SimpleStruct struct {
	A int64
	B int32
	C bool
}

type ComplexStruct struct {
	ID        string
	Data      SimpleStruct
	Values    []int
	Mapping   map[string]int
	Timestamp time.Time
}
`

	err := os.WriteFile(testFilePath, []byte(testFileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	structs, err := AnalyseFromFileWithStrategies(testFilePath, nil)
	if err != nil {
		t.Fatalf("AnalyseFromFile failed: %v", err)
	}

	if len(structs) != 2 {
		t.Errorf("Expected 2 structs, got %d", len(structs))
	}

	var simpleStruct *Info
	var complexStruct *Info

	for i := range structs {
		if structs[i].Name == "SimpleStruct" {
			simpleStruct = &structs[i]
		} else if structs[i].Name == "ComplexStruct" {
			complexStruct = &structs[i]
		}
	}

	if simpleStruct == nil {
		t.Fatalf("SimpleStruct not found in results")
	}

	nonPaddingFields := 0
	for _, field := range simpleStruct.Fields {
		if !field.IsPadding {
			nonPaddingFields++
		}
	}

	if nonPaddingFields != 3 {
		t.Errorf("Expected 3 non-padding fields in SimpleStruct, got %d", nonPaddingFields)
	}

	if complexStruct == nil {
		t.Fatalf("ComplexStruct not found in results")
	}

	nonPaddingFields = 0
	for _, field := range complexStruct.Fields {
		if !field.IsPadding {
			nonPaddingFields++
		}
	}

	if nonPaddingFields != 5 {
		t.Errorf("Expected 5 non-padding fields in ComplexStruct, got %d", nonPaddingFields)
	}

	timeFieldFound := false
	for _, field := range complexStruct.Fields {
		if !field.IsPadding && field.Name == "Timestamp" && field.TypeName == "time.Time" {
			timeFieldFound = true
			break
		}
	}

	if !timeFieldFound {
		t.Errorf("time.Time field not properly resolved in ComplexStruct")
	}
}

func TestCollectFieldOptmizers(t *testing.T) {
	// Create a mock type for testing
	mockType := &MockType{}

	tests := []struct {
		name           string
		strategyNames  []string
		expectedCount  int
		expectedNames  []string
		expectError    bool
		errorSubstring string
	}{
		{
			name:          "empty strategy names returns all optimizers",
			strategyNames: []string{},
			expectedCount: 4,
			expectedNames: []string{"alignment-then-size", "size-then-alignment", "group-by-alignment", "greedy-packing"},
		},
		{
			name:          "single strategy by CLI name",
			strategyNames: []string{"alignment"},
			expectedCount: 1,
			expectedNames: []string{"alignment-then-size"},
		},
		{
			name:          "multiple strategies by CLI names",
			strategyNames: []string{"alignment", "size", "group"},
			expectedCount: 3,
			expectedNames: []string{"alignment-then-size", "size-then-alignment", "group-by-alignment"},
		},
		{
			name:          "multiple strategies by full names",
			strategyNames: []string{"alignment-then-size", "size-then-alignment"},
			expectedCount: 2,
			expectedNames: []string{"alignment-then-size", "size-then-alignment"},
		},
		{
			name:          "mixed CLI and full names",
			strategyNames: []string{"alignment", "size-then-alignment", "greedy"},
			expectedCount: 3,
			expectedNames: []string{"alignment-then-size", "size-then-alignment", "greedy-packing"},
		},
		{
			name:           "invalid strategy name",
			strategyNames:  []string{"invalid"},
			expectError:    true,
			errorSubstring: "unknown optimizer: invalid",
		},
		{
			name:           "mix of valid and invalid",
			strategyNames:  []string{"alignment", "invalid", "size"},
			expectError:    true,
			errorSubstring: "invalid optimizers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizers, err := collectFieldOptmizers(tt.strategyNames)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if tt.errorSubstring != "" && !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("error %q does not contain expected substring %q", err.Error(), tt.errorSubstring)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(optimizers) != tt.expectedCount {
				t.Errorf("expected %d optimizers, got %d", tt.expectedCount, len(optimizers))
			}

			if tt.expectedNames != nil {
				var names []string
				for _, opt := range optimizers {
					names = append(names, opt.Name())
				}

				sort.Strings(names)
				expectedSorted := make([]string, len(tt.expectedNames))
				copy(expectedSorted, tt.expectedNames)
				sort.Strings(expectedSorted)

				if !reflect.DeepEqual(names, expectedSorted) {
					t.Errorf("expected optimizer names %v, got %v", expectedSorted, names)
				}
			}

			if len(optimizers) > 0 {
				testFields := []fieldWithMeta{
					{name: "test1", typ: mockType, size: 4, align: 4},
					{name: "test2", typ: mockType, size: 8, align: 8},
				}

				originalFields := make([]fieldWithMeta, len(testFields))
				copy(originalFields, testFields)

				for _, opt := range optimizers {
					// Call Optimize instead of ArrangeFields
					result := opt.Optimize(testFields)

					if !reflect.DeepEqual(testFields, originalFields) {
						t.Errorf("optimizer %s modified the input slice", opt.Name())
					}

					// Count non-padding fields in the result
					nonPaddingCount := 0
					for _, f := range result {
						if !f.IsPadding {
							nonPaddingCount++
						}
					}

					if nonPaddingCount != len(testFields) {
						t.Errorf("optimizer %s returned incorrect number of non-padding fields: got %d, expected %d",
							opt.Name(), nonPaddingCount, len(testFields))
					}
				}
			}
		})
	}
}

// MockType implements types.Type for testing
type MockType struct{}

func (m *MockType) Underlying() types.Type { return m }
func (m *MockType) String() string         { return "MockType" }
