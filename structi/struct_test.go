package structi

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"math"
	"os"
	"os/exec"
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
		expectedInfo []string // expected struct names (including nested ones)
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

			results, err := AnalyseStructsAsStringWithStrategies(tt.src, nil)
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
		{
			name: "struct with trailing empty struct",
			input: `type WithTrailingEmpty struct {
				A int64
				B int64
				C struct{}
			}`,
			validate: func(t *testing.T, infos []Info) {
				if len(infos) != 1 {
					t.Fatalf("expected 1 struct, got %d", len(infos))
				}
				info := infos[0]
				// Go adds 8 bytes tail padding when last field is zero-size
				if info.OriginalSize != 24 {
					t.Errorf("expected original size of 24 bytes, got %d", info.OriginalSize)
				}
				foundC := false
				for _, f := range info.Fields {
					if f.Name == "C" {
						foundC = true
						if f.Size != 0 {
							t.Errorf("expected field C to have size 0, got %d", f.Size)
						}
					}
				}
				if !foundC {
					t.Error("field C (struct{}) not found in layout")
				}
			},
		},
		{
			name: "struct with non-trailing empty struct",
			input: `type WithMiddleEmpty struct {
				A int64
				B struct{}
				C int64
			}`,
			validate: func(t *testing.T, infos []Info) {
				if len(infos) != 1 {
					t.Fatalf("expected 1 struct, got %d", len(infos))
				}
				info := infos[0]
				// No special tail padding when empty struct is not last
				if info.OriginalSize != 16 {
					t.Errorf("expected original size of 16 bytes, got %d", info.OriginalSize)
				}
			},
		},
		{
			name: "struct with only empty struct field",
			input: `type OnlyEmpty struct {
				A struct{}
			}`,
			validate: func(t *testing.T, infos []Info) {
				if len(infos) != 1 {
					t.Fatalf("expected 1 struct, got %d", len(infos))
				}
				info := infos[0]
				// struct{ A struct{} } has unsafe.Sizeof = 0
				if info.OriginalSize != 0 {
					t.Errorf("expected original size of 0 bytes, got %d", info.OriginalSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := AnalyseStructsAsStringWithStrategies(tt.input, nil)

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

func TestCollectFieldOptmizers(t *testing.T) {
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
					result := opt.Optimize(testFields)

					if !reflect.DeepEqual(testFields, originalFields) {
						t.Errorf("optimizer %s modified the input slice", opt.Name())
					}

					// count non-padding fields in the result
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

func TestAnalyseStructsAtDirectoryPath(t *testing.T) {
	tempDir := t.TempDir()

	goModContent := `module testmodule

go 1.20
`
	err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("failed to create go.mod file: %v", err)
	}

	structFileContent := `
package testmodule

// Struct with various field types and alignments
type TestStruct struct {
	A int64   // 8 bytes, 8-byte alignment
	B bool    // 1 byte, 1-byte alignment
	C int32   // 4 bytes, 4-byte alignment
	D string  // 16 bytes, 8-byte alignment
}

// Another struct to test multiple struct detection
type AnotherStruct struct {
	X float64 // 8 bytes, 8-byte alignment
	Y []byte  // 24 bytes, 8-byte alignment
}
`
	structFilePath := filepath.Join(tempDir, "structs.go")
	err = os.WriteFile(structFilePath, []byte(structFileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create struct file: %v", err)
	}

	t.Run("Test directory with Go module", func(t *testing.T) {
		_, err := exec.LookPath("go")
		if err != nil {
			t.Skip("go executable not found in PATH, skipping test")
		}

		results, err := AnalyseStructsAtDirectoryPath(tempDir, nil)
		if err != nil && strings.Contains(err.Error(), "fork/exec") {
			t.Skipf("skipping test due to toolchain error: %v", err)
			return
		}

		if err != nil {
			t.Fatalf("AnalyseStructsAtDirectoryPath failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("expected at least one package, got none")
		}

		var foundTestStruct, foundAnotherStruct bool
		for _, pkg := range results {
			for _, info := range pkg {
				if info.Name == "TestStruct" {
					foundTestStruct = true
					validateStructInfo(t, info, "TestStruct", 4) // 4 fields expected
				} else if info.Name == "AnotherStruct" {
					foundAnotherStruct = true
					validateStructInfo(t, info, "AnotherStruct", 2) // 2 fields expected
				}
			}
		}

		if !foundTestStruct {
			t.Error("TestStruct not found in analysis results")
		}
		if !foundAnotherStruct {
			t.Error("AnotherStruct not found in analysis results")
		}
	})
}

func validateStructInfo(t *testing.T, info Info, structName string, expectedFieldCount int) {
	t.Helper()

	if info.OriginalSize == 0 {
		t.Errorf("%s: original size is zero", structName)
	}
	if info.OptimizedSize == 0 {
		t.Errorf("%s: optimized size is zero", structName)
	}

	var fieldCount int
	for _, field := range info.Fields {
		if !field.IsPadding {
			fieldCount++
		}
	}

	if fieldCount != expectedFieldCount {
		t.Errorf("%s: expected %d non-padding fields, got %d",
			structName, expectedFieldCount, fieldCount)
	}

	if len(info.OptimizedFields) == 0 {
		t.Errorf("%s: optimized fields not populated", structName)
	}
}

type MockType struct{}

func (m *MockType) Underlying() types.Type { return m }
func (m *MockType) String() string         { return "MockType" }
