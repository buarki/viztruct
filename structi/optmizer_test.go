package structi

import (
	"go/types"
	"testing"
)

func TestNewAlignmentThenSizeOptimizer(t *testing.T) {
	tests := []struct {
		name          string
		fields        []fieldWithMeta
		expectedOrder []string
	}{
		{
			name:          "empty fields",
			fields:        []fieldWithMeta{},
			expectedOrder: []string{},
		},
		{
			name: "single field",
			fields: []fieldWithMeta{
				{name: "field1", size: 8, align: 8, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field1"},
		},
		{
			name: "same size different alignments",
			fields: []fieldWithMeta{
				{name: "field1", size: 8, align: 4, typ: &MockStruct{}},
				{name: "field2", size: 8, align: 8, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field2", "field1"},
		},
		{
			name: "different sizes same alignment",
			fields: []fieldWithMeta{
				{name: "field1", size: 4, align: 4, typ: &MockStruct{}},
				{name: "field2", size: 8, align: 4, typ: &MockStruct{}},
				{name: "field3", size: 2, align: 4, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field2", "field1", "field3"},
		},
		{
			name: "mixed size and alignment",
			fields: []fieldWithMeta{
				{name: "bool", size: 1, align: 1, typ: &MockStruct{}},
				{name: "int64", size: 8, align: 8, typ: &MockStruct{}},
				{name: "int32", size: 4, align: 4, typ: &MockStruct{}},
				{name: "int16", size: 2, align: 2, typ: &MockStruct{}},
				{name: "byte", size: 1, align: 1, typ: &MockStruct{}},
				{name: "float64", size: 8, align: 4, typ: &MockStruct{}},
			},
			expectedOrder: []string{"int64", "float64", "int32", "int16", "bool", "byte"},
		},
	}

	// Create the optimizer to test
	optimizer := NewAlignmentThenSizeOptimizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the optimizer
			result := optimizer.Optimize(tt.fields)

			// Extract only non-padding fields
			var actualOrder []string
			for _, field := range result {
				if !field.IsPadding {
					actualOrder = append(actualOrder, field.Name)
				}
			}

			// Compare the expected and actual ordering
			if len(actualOrder) != len(tt.expectedOrder) {
				t.Fatalf("expected %d fields, got %d", len(tt.expectedOrder), len(actualOrder))
			}

			for i, expected := range tt.expectedOrder {
				if i >= len(actualOrder) {
					t.Fatalf("missing expected field at position %d: %s", i, expected)
				}
				if actualOrder[i] != expected {
					t.Errorf("field at position %d: expected %q, got %q", i, expected, actualOrder[i])
				}
			}
		})
	}
}

func TestNewSizeThenAlignmentOptimizer(t *testing.T) {
	tests := []struct {
		name          string
		fields        []fieldWithMeta
		expectedOrder []string
	}{
		{
			name:          "empty fields",
			fields:        []fieldWithMeta{},
			expectedOrder: []string{},
		},
		{
			name: "single field",
			fields: []fieldWithMeta{
				{name: "field1", size: 8, align: 8, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field1"},
		},
		{
			name: "same size different alignments",
			fields: []fieldWithMeta{
				{name: "field1", size: 8, align: 4, typ: &MockStruct{}},
				{name: "field2", size: 8, align: 8, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field2", "field1"},
		},
		{
			name: "different sizes same alignment",
			fields: []fieldWithMeta{
				{name: "field1", size: 4, align: 4, typ: &MockStruct{}},
				{name: "field2", size: 8, align: 4, typ: &MockStruct{}},
				{name: "field3", size: 2, align: 4, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field2", "field1", "field3"},
		},
		{
			name: "mixed size and alignment",
			fields: []fieldWithMeta{
				{name: "bool", size: 1, align: 1, typ: &MockStruct{}},
				{name: "int64", size: 8, align: 8, typ: &MockStruct{}},
				{name: "int32", size: 4, align: 4, typ: &MockStruct{}},
				{name: "int16", size: 2, align: 2, typ: &MockStruct{}},
				{name: "byte", size: 1, align: 1, typ: &MockStruct{}},
				{name: "float64", size: 8, align: 4, typ: &MockStruct{}},
			},
			expectedOrder: []string{"int64", "float64", "int32", "int16", "bool", "byte"},
		},
	}

	// Create the optimizer to test
	optimizer := NewSizeThenAlignmentOptimizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the optimizer
			result := optimizer.Optimize(tt.fields)

			// Extract only non-padding fields
			var actualOrder []string
			for _, field := range result {
				if !field.IsPadding {
					actualOrder = append(actualOrder, field.Name)
				}
			}

			// Compare the expected and actual ordering
			if len(actualOrder) != len(tt.expectedOrder) {
				t.Fatalf("expected %d fields, got %d", len(tt.expectedOrder), len(actualOrder))
			}

			for i, expected := range tt.expectedOrder {
				if i >= len(actualOrder) {
					t.Fatalf("missing expected field at position %d: %s", i, expected)
				}
				if actualOrder[i] != expected {
					t.Errorf("field at position %d: expected %q, got %q", i, expected, actualOrder[i])
				}
			}
		})
	}
}

func TestNewGroupByAlignmentOptimizer(t *testing.T) {
	tests := []struct {
		name          string
		fields        []fieldWithMeta
		expectedOrder []string
	}{
		{
			name:          "empty fields",
			fields:        []fieldWithMeta{},
			expectedOrder: []string{},
		},
		{
			name: "single field",
			fields: []fieldWithMeta{
				{name: "field1", size: 8, align: 8, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field1"},
		},
		{
			name: "same alignment different sizes",
			fields: []fieldWithMeta{
				{name: "field1", size: 4, align: 8, typ: &MockStruct{}},
				{name: "field2", size: 8, align: 8, typ: &MockStruct{}},
			},
			expectedOrder: []string{"field2", "field1"},
		},
		{
			name: "mixed alignments",
			fields: []fieldWithMeta{
				{name: "align4_small", size: 2, align: 4, typ: &MockStruct{}},
				{name: "align8_medium", size: 4, align: 8, typ: &MockStruct{}},
				{name: "align4_large", size: 8, align: 4, typ: &MockStruct{}},
				{name: "align8_small", size: 2, align: 8, typ: &MockStruct{}},
				{name: "align1_medium", size: 4, align: 1, typ: &MockStruct{}},
			},
			expectedOrder: []string{
				"align8_medium", "align8_small",
				"align4_large", "align4_small",
				"align1_medium",
			},
		},
		{
			name: "multiple fields same alignment",
			fields: []fieldWithMeta{
				{name: "small1", size: 1, align: 1, typ: &MockStruct{}},
				{name: "small2", size: 2, align: 1, typ: &MockStruct{}},
				{name: "medium1", size: 4, align: 4, typ: &MockStruct{}},
				{name: "medium2", size: 8, align: 4, typ: &MockStruct{}},
				{name: "large1", size: 16, align: 8, typ: &MockStruct{}},
			},
			expectedOrder: []string{
				"large1",
				"medium2", "medium1",
				"small2", "small1",
			},
		},
	}

	// Create the optimizer to test
	optimizer := NewGroupByAlignmentOptimizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the optimizer
			result := optimizer.Optimize(tt.fields)

			// Extract only non-padding fields
			var actualOrder []string
			for _, field := range result {
				if !field.IsPadding {
					actualOrder = append(actualOrder, field.Name)
				}
			}

			// Compare the expected and actual ordering
			if len(actualOrder) != len(tt.expectedOrder) {
				t.Fatalf("expected %d fields, got %d", len(tt.expectedOrder), len(actualOrder))
			}

			for i, expected := range tt.expectedOrder {
				if i >= len(actualOrder) {
					t.Fatalf("missing expected field at position %d: %s", i, expected)
				}
				if actualOrder[i] != expected {
					t.Errorf("field at position %d: expected %q, got %q", i, expected, actualOrder[i])
				}
			}
		})
	}
}

func TestNewGreedyPackingOptimizer(t *testing.T) {
	tests := []struct {
		name   string
		fields []fieldWithMeta
	}{
		{
			name:   "empty fields",
			fields: []fieldWithMeta{},
		},
		{
			name: "single field",
			fields: []fieldWithMeta{
				{name: "field1", size: 8, align: 8, typ: &MockStruct{}},
			},
		},
		{
			name: "typical struct fields",
			fields: []fieldWithMeta{
				{name: "bool", size: 1, align: 1, typ: &MockStruct{}},
				{name: "int64", size: 8, align: 8, typ: &MockStruct{}},
				{name: "int32", size: 4, align: 4, typ: &MockStruct{}},
				{name: "int16", size: 2, align: 2, typ: &MockStruct{}},
			},
		},
		{
			name: "mix of small and large fields",
			fields: []fieldWithMeta{
				{name: "small1", size: 1, align: 1, typ: &MockStruct{}},
				{name: "large1", size: 8, align: 8, typ: &MockStruct{}},
				{name: "small2", size: 1, align: 1, typ: &MockStruct{}},
				{name: "large2", size: 8, align: 8, typ: &MockStruct{}},
				{name: "medium", size: 4, align: 4, typ: &MockStruct{}},
			},
		},
	}

	// Create the optimizer to test
	optimizer := NewGreedyPackingOptimizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Make a copy of the original field names for verification
			expectedFieldNames := make([]string, len(tt.fields))
			for i, f := range tt.fields {
				expectedFieldNames[i] = f.name
			}

			// Run the optimizer
			result := optimizer.Optimize(tt.fields)

			// Check that all fields are present (though possibly reordered)
			var actualFieldNames []string
			for _, field := range result {
				if !field.IsPadding {
					actualFieldNames = append(actualFieldNames, field.Name)
				}
			}

			if len(actualFieldNames) != len(expectedFieldNames) {
				t.Fatalf("expected %d fields, got %d", len(expectedFieldNames), len(actualFieldNames))
			}

			// Verify all original field names exist in the result
			fieldNameExists := make(map[string]bool)
			for _, name := range actualFieldNames {
				fieldNameExists[name] = true
			}

			for _, expected := range expectedFieldNames {
				if !fieldNameExists[expected] {
					t.Errorf("expected field %q missing from result", expected)
				}
			}

			// Verify proper alignment of fields
			for i, f := range result {
				if f.IsPadding {
					continue
				}
				if f.Offset%f.Align != 0 {
					t.Errorf("field %q at position %d is not properly aligned: offset %d with alignment %d",
						f.Name, i, f.Offset, f.Align)
				}
			}
		})
	}
}

// MockStruct implements types.Type for testing
type MockStruct struct{}

func (m *MockStruct) Underlying() types.Type { return m }
func (m *MockStruct) String() string         { return "MockType" }
