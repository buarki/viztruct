package structi

import (
	"fmt"
	"sort"
	"strings"
)

// FieldOptimizer defines an interface for struct field layout optimization strategies
type FieldOptimizer interface {
	// Name returns the unique identifier for this strategy
	Name() string
	// Optimize is the main method that takes struct field metadata and returns the optimized layout
	Optimize(fields []fieldWithMeta) []Field
}

// ArrangeStrategy is a function type that defines how fields should be arranged
type ArrangeStrategy func([]fieldWithMeta) []fieldWithMeta

// Optimizer implements the field layout optimization logic
type Optimizer struct {
	name            string
	arrangeStrategy ArrangeStrategy
}

// Name returns the optimizer's name
func (o Optimizer) Name() string {
	return o.name
}

// Optimize implements the main optimization flow
func (o Optimizer) Optimize(fields []fieldWithMeta) []Field {
	fieldsCopy := make([]fieldWithMeta, len(fields))
	copy(fieldsCopy, fields)

	arrangedFields := o.arrangeStrategy(fieldsCopy)

	var result []Field
	var offset int64 = 0

	for _, f := range arrangedFields {
		// align field
		if rem := offset % f.align; rem != 0 {
			paddingSize := f.align - rem
			result = append(result, Field{
				Name:      "padding",
				TypeName:  "",
				Offset:    offset,
				Size:      paddingSize,
				Align:     1,
				IsPadding: true,
			})
			offset += paddingSize
		}

		result = append(result, Field{
			Name:      f.name,
			TypeName:  typeName(f.typ),
			Offset:    offset,
			Size:      f.size,
			Align:     f.align,
			IsPadding: false,
		})

		offset += f.size
	}

	result = addTailPadding(result, arrangedFields, offset)

	return result
}

// addTailPadding appends the correct tail padding to a layout result.
// If the last real field has zero size (e.g. struct{}), Go adds padding equal to structAlign.
// Otherwise, standard alignment-based tail padding is added.
func addTailPadding(result []Field, arrangedFields []fieldWithMeta, offset int64) []Field {
	var structAlign int64 = 1
	for _, f := range arrangedFields {
		if f.align > structAlign {
			structAlign = f.align
		}
	}

	lastFieldIsZeroSize := false
	if len(arrangedFields) > 0 {
		if arrangedFields[len(arrangedFields)-1].size == 0 {
			lastFieldIsZeroSize = true
		}
	}

	if lastFieldIsZeroSize && offset > 0 {
		result = append(result, Field{
			Name:      "tail padding",
			TypeName:  "",
			Offset:    offset,
			Size:      structAlign,
			Align:     1,
			IsPadding: true,
		})
	} else if rem := offset % structAlign; rem != 0 {
		paddingSize := structAlign - rem
		result = append(result, Field{
			Name:      "tail padding",
			TypeName:  "",
			Offset:    offset,
			Size:      paddingSize,
			Align:     1,
			IsPadding: true,
		})
	}

	return result
}

// NewAlignmentThenSizeOptimizer creates an optimizer that sorts fields by alignment then size
func NewAlignmentThenSizeOptimizer() FieldOptimizer {
	return Optimizer{
		name: "alignment-then-size",
		arrangeStrategy: func(fields []fieldWithMeta) []fieldWithMeta {
			result := make([]fieldWithMeta, len(fields))
			copy(result, fields)

			sort.Slice(result, func(i, j int) bool {
				if result[i].align != result[j].align {
					return result[i].align > result[j].align
				}
				return result[i].size > result[j].size
			})

			return result
		},
	}
}

// NewSizeThenAlignmentOptimizer creates an optimizer that sorts fields by size then alignment
func NewSizeThenAlignmentOptimizer() FieldOptimizer {
	return Optimizer{
		name: "size-then-alignment",
		arrangeStrategy: func(fields []fieldWithMeta) []fieldWithMeta {
			result := make([]fieldWithMeta, len(fields))
			copy(result, fields)

			sort.Slice(result, func(i, j int) bool {
				if result[i].size != result[j].size {
					return result[i].size > result[j].size
				}
				return result[i].align > result[j].align
			})

			return result
		},
	}
}

// NewGroupByAlignmentOptimizer creates an optimizer that groups fields by alignment
func NewGroupByAlignmentOptimizer() FieldOptimizer {
	return Optimizer{
		name: "group-by-alignment",
		arrangeStrategy: func(fields []fieldWithMeta) []fieldWithMeta {
			fieldsGroupedAlignmentWise := make(map[int64][]fieldWithMeta)
			for _, f := range fields {
				fieldsGroupedAlignmentWise[f.align] = append(fieldsGroupedAlignmentWise[f.align], f)
			}

			var alignments []int64
			for align := range fieldsGroupedAlignmentWise {
				alignments = append(alignments, align)
			}
			sort.Slice(alignments, func(i, j int) bool {
				return alignments[i] > alignments[j]
			})

			result := make([]fieldWithMeta, 0, len(fields))

			// add fields back in order of alignment groups
			for _, align := range alignments {
				group := fieldsGroupedAlignmentWise[align]
				// sort fields within each group by size (descending)
				sort.Slice(group, func(i, j int) bool {
					return group[i].size > group[j].size
				})
				result = append(result, group...)
			}

			return result
		},
	}
}

// GreedyOptimizer is a specialized optimizer that handles the greedy packing algorithm
type GreedyOptimizer struct {
	name string
}

func (g GreedyOptimizer) Name() string {
	return g.name
}

// Optimize implements the greedy packing algorithm with special padding optimization
func (g GreedyOptimizer) Optimize(fields []fieldWithMeta) []Field {
	fieldsCopy := make([]fieldWithMeta, len(fields))
	copy(fieldsCopy, fields)

	// step 1: Arrange fields using greedy algorithm
	// sort by alignment (descending)
	sort.Slice(fieldsCopy, func(i, j int) bool {
		return fieldsCopy[i].align > fieldsCopy[j].align
	})

	// split into large and small fields
	var largeFields, smallFields []fieldWithMeta
	largeAlignThreshold := int64(4)

	for _, f := range fieldsCopy {
		if f.align >= largeAlignThreshold {
			largeFields = append(largeFields, f)
		} else {
			smallFields = append(smallFields, f)
		}
	}

	// sort small fields by size (descending)
	sort.Slice(smallFields, func(i, j int) bool {
		return smallFields[i].size > smallFields[j].size
	})

	// initialize result with large fields
	arrangedFields := make([]fieldWithMeta, len(largeFields))
	copy(arrangedFields, largeFields)

	// place small fields in optimal positions
	for _, small := range smallFields {
		bestPos := len(arrangedFields)
		for i, existing := range arrangedFields {
			if i == len(arrangedFields)-1 {
				continue
			}
			nextField := arrangedFields[i+1]
			gapSize := (nextField.align - (existing.size % nextField.align)) % nextField.align
			if gapSize >= small.size && small.align <= gapSize {
				bestPos = i + 1
				break
			}
		}

		if bestPos < len(arrangedFields) {
			// insert small field at best position
			newResult := make([]fieldWithMeta, 0, len(arrangedFields)+1)
			newResult = append(newResult, arrangedFields[:bestPos]...)
			newResult = append(newResult, small)
			newResult = append(newResult, arrangedFields[bestPos:]...)
			arrangedFields = newResult
		} else {
			arrangedFields = append(arrangedFields, small)
		}
	}

	// step 2: Calculate field offsets and add padding
	var result []Field
	var offset int64 = 0

	for _, f := range arrangedFields {
		// align field
		if rem := offset % f.align; rem != 0 {
			paddingSize := f.align - rem
			result = append(result, Field{
				Name:      "padding",
				TypeName:  "",
				Offset:    offset,
				Size:      paddingSize,
				Align:     1,
				IsPadding: true,
			})
			offset += paddingSize
		}

		result = append(result, Field{
			Name:      f.name,
			TypeName:  typeName(f.typ),
			Offset:    offset,
			Size:      f.size,
			Align:     f.align,
			IsPadding: false,
		})

		offset += f.size
	}

	result = addTailPadding(result, arrangedFields, offset)

	// step 3: Greedy-specific post-processing
	// sort by offset for proper display
	sort.Slice(result, func(i, j int) bool {
		return result[i].Offset < result[j].Offset
	})

	return recalculatePadding(result, arrangedFields)
}

// recalculate padding between fields - specifically for greedy packing
func recalculatePadding(fields []Field, arrangedFields []fieldWithMeta) []Field {
	// remove all padding fields
	var nonPaddingFields []Field
	for _, f := range fields {
		if !f.IsPadding {
			nonPaddingFields = append(nonPaddingFields, f)
		}
	}

	// sort by offset
	sort.Slice(nonPaddingFields, func(i, j int) bool {
		return nonPaddingFields[i].Offset < nonPaddingFields[j].Offset
	})

	// recalculate padding
	var result []Field
	for i := 0; i < len(nonPaddingFields); i++ {
		current := nonPaddingFields[i]

		// add the current field
		result = append(result, current)

		// if there's another field, check if padding is needed
		if i < len(nonPaddingFields)-1 {
			next := nonPaddingFields[i+1]
			gap := next.Offset - (current.Offset + current.Size)

			if gap > 0 {
				// add padding
				result = append(result, Field{
					Name:      "padding",
					TypeName:  "",
					Offset:    current.Offset + current.Size,
					Size:      gap,
					Align:     1,
					IsPadding: true,
				})
			}
		}
	}

	// calculate struct alignment
	var structAlign int64 = 1
	for _, f := range nonPaddingFields {
		if f.Align > structAlign {
			structAlign = f.Align
		}
	}

	// check if we need to add final padding
	if len(result) > 0 {
		lastField := result[len(result)-1]
		maxOffset := lastField.Offset + lastField.Size

		// check if the last arranged field has zero size (e.g. struct{})
		lastArrangedIsZeroSize := false
		if len(arrangedFields) > 0 {
			if arrangedFields[len(arrangedFields)-1].size == 0 {
				lastArrangedIsZeroSize = true
			}
		}

		if lastArrangedIsZeroSize && maxOffset > 0 {
			result = append(result, Field{
				Name:      "tail padding",
				TypeName:  "",
				Offset:    maxOffset,
				Size:      structAlign,
				Align:     1,
				IsPadding: true,
			})
		} else if rem := maxOffset % structAlign; rem != 0 {
			paddingSize := structAlign - rem
			result = append(result, Field{
				Name:      "tail padding",
				TypeName:  "",
				Offset:    maxOffset,
				Size:      paddingSize,
				Align:     1,
				IsPadding: true,
			})
		}
	}

	return result
}

// NewGreedyPackingOptimizer creates an optimizer that tries to fit fields into gaps
func NewGreedyPackingOptimizer() FieldOptimizer {
	return GreedyOptimizer{name: "greedy-packing"}
}

func GetAllOptimizers() []FieldOptimizer {
	return []FieldOptimizer{
		NewAlignmentThenSizeOptimizer(),
		NewSizeThenAlignmentOptimizer(),
		NewGroupByAlignmentOptimizer(),
		NewGreedyPackingOptimizer(),
	}
}

// GetOptimizerByName returns a field optimizer by its name or CLI shortname
func GetOptimizerByName(name string) (FieldOptimizer, error) {
	name = strings.ToLower(strings.TrimSpace(name))

	if fullName, ok := cliToOptimizerNames[name]; ok {
		name = fullName
	}

	for _, optimizer := range GetAllOptimizers() {
		if strings.ToLower(optimizer.Name()) == name {
			return optimizer, nil
		}
	}

	return nil, fmt.Errorf("unknown optimizer: %s", name)
}

// GetOptimizersByNames returns a list of field optimizers by their names or CLI shortnames
func GetOptimizersByNames(names []string) ([]FieldOptimizer, error) {
	if len(names) == 0 {
		return []FieldOptimizer{NewAlignmentThenSizeOptimizer()}, nil
	}

	var optimizers []FieldOptimizer
	var errors []string

	for _, name := range names {
		optimizer, err := GetOptimizerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
		} else {
			optimizers = append(optimizers, optimizer)
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("invalid optimizers: %s", strings.Join(errors, ", "))
	}

	return optimizers, nil
}

// GetAllOptimizerNames returns a list of all available optimizer names
func GetAllOptimizerNames() []string {
	return []string{
		"alignment",
		"size",
		"group",
		"greedy",
	}
}

// Reverse mapping from CLI names to full names
var cliToOptimizerNames = map[string]string{
	"alignment": "alignment-then-size",
	"size":      "size-then-alignment",
	"group":     "group-by-alignment",
	"greedy":    "greedy-packing",
}
