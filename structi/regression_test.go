package structi

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// analyzeFileForTest is a simplified version of AnalyseFromFileWithStrategies that
// doesn't depend on running the Go toolchain. It's for regression tests only.
func analyzeFileForTest(filePath string, strategyNames []string) ([]Info, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return AnalyseStructsAsStringWithStrategies(string(content), strategyNames)
}

const sampleFile = `package sample

// The test doesn't actually need the time.Time type, we can define a mock replacement
type Time struct {
	sec  int64
	nsec int32
	loc  int
}

type SampleStruct struct {
	UserID          uint32 // 4 bytes
	Activated       bool   // 1 byte
	Username        string // 16 bytes
	PersonalDetails struct {
		FirstName     string // 16 bytes
		MiddleInitial byte   // 1 byte
		LastName      string // 16 bytes
		Age           int8   // 1 byte
	}
	SessionToken       [16]byte          // 16 bytes fixed array
	LastLoginTime      Time              // 24 bytes
	PreferenceFlags    uint8             // 1 byte
	AccountBalance     float64           // 8 bytes
	Friends            map[string]uint64 // 8 bytes (pointer to map)
	RecentSearches     []string          // 24 bytes (slice)
	PremiumMember      bool              // 1 byte
	MemberSince        Time              // 24 bytes
	NotificationPrefs  byte              // 1 byte
	ProfilePictureData []byte            // 24 bytes (slice)
	DeviceID           uint64            // 8 bytes
	EmailVerified      bool              // 1 byte
	AddressInfo        struct {
		Street    string // 16 bytes
		City      string // 16 bytes
		ZipCode   uint16 // 2 bytes
		Country   string // 16 bytes
		IsPrimary bool   // 1 byte
	}
	LoginAttempts     uint16            // 2 bytes
	SecurityQuestions [3]string         // 48 bytes (array of 3 strings)
	AccountType       byte              // 1 byte
	Preferences       map[string]string // 8 bytes (pointer to map)
}
`

func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func countNonPaddingFields(fields []Field) int {
	count := 0
	for _, field := range fields {
		if !field.IsPadding {
			count++
		}
	}
	return count
}

func TestRegression_AlignmentThenSize(t *testing.T) {
	// Expected values for this specific struct layout and optimizer
	const (
		expectedOriginalSize     int64   = 368
		expectedOptimizedSize    int64   = 336
		expectedOrigWasted       int64   = 50
		expectedOptWasted        int64   = 18
		expectedOrigWastedPct    float64 = 13.59
		expectedOptWastedPct     float64 = 5.36
		expectedSizeReductionPct float64 = 8.7 // (368-336)/368 * 100
	)

	// Expected field alignments in the optimized layout - first fields of each alignment group
	// For alignment-then-size, larger alignments should come first
	expectedAlignmentGroups := []int64{8, 4, 2, 1}

	// Expected fields at beginning of each alignment group (first field for each alignment)
	// These are fields we expect to see at the start of their respective alignment groups
	// Note: The actual first fields might be different depending on how the optimizer is implemented
	expectedGroupStarters := map[int64]string{
		8: "AddressInfo",   // An 8-byte aligned field at the start of 8-byte group
		4: "UserID",        // A 4-byte aligned field at the start of 4-byte group
		2: "LoginAttempts", // A 2-byte aligned field at the start of 2-byte group
		1: "SessionToken",  // A 1-byte aligned field at the start of 1-byte group
	}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "sample_struct.go")

	err := os.WriteFile(tempFile, []byte(sampleFile), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	results, err := analyzeFileForTest(tempFile, []string{"alignment-then-size"})
	if err != nil {
		t.Fatalf("failed to analyze struct: %v", err)
	}

	var sampleResult Info
	found := false
	for _, r := range results {
		if r.Name == "SampleStruct" {
			sampleResult = r
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("SampleStruct not found in results")
	}
	if sampleResult.Name != "SampleStruct" {
		t.Fatalf("expected struct name 'SampleStruct', got '%s'", sampleResult.Name)
	}

	if sampleResult.OriginalSize != expectedOriginalSize {
		t.Errorf("original size: got %d, expected %d", sampleResult.OriginalSize, expectedOriginalSize)
	}

	if sampleResult.OptimizedSize != expectedOptimizedSize {
		t.Errorf("optimized size: got %d, expected %d", sampleResult.OptimizedSize, expectedOptimizedSize)
	}

	origWasted, origWastedPercent := sampleResult.WastedSpace()
	if origWasted != expectedOrigWasted {
		t.Errorf("original wasted bytes: got %d, expected %d", origWasted, expectedOrigWasted)
	}

	if !floatEquals(origWastedPercent, expectedOrigWastedPct, 0.01) {
		t.Errorf("original wasted percentage: got %.2f%%, expected %.2f%%", origWastedPercent, expectedOrigWastedPct)
	}

	optWasted, optWastedPercent := sampleResult.OptimizedWastedSpace()
	if optWasted != expectedOptWasted {
		t.Errorf("optimized wasted bytes: got %d, expected %d", optWasted, expectedOptWasted)
	}

	if !floatEquals(optWastedPercent, expectedOptWastedPct, 0.01) {
		t.Errorf("optimized wasted percentage: got %.2f%%, expected %.2f%%", optWastedPercent, expectedOptWastedPct)
	}

	sizeReductionPct := float64(sampleResult.OriginalSize-sampleResult.OptimizedSize) / float64(sampleResult.OriginalSize) * 100
	if !floatEquals(sizeReductionPct, expectedSizeReductionPct, 0.1) {
		t.Errorf("size reduction percentage: got %.2f%%, expected %.2f%%", sizeReductionPct, expectedSizeReductionPct)
	}

	// check field alignment groups and order
	var currentAlignGroup int64 = -1
	var alignmentGroups []int64
	var firstFieldsInGroup = make(map[int64]string)
	var seenFields = make(map[string]bool)

	for _, field := range sampleResult.OptimizedFields {
		if field.IsPadding {
			continue
		}

		// record the first field we see for each alignment
		if field.Align != currentAlignGroup {
			currentAlignGroup = field.Align
			alignmentGroups = append(alignmentGroups, currentAlignGroup)

			// record the first field name for this alignment group
			if _, exists := firstFieldsInGroup[currentAlignGroup]; !exists {
				firstFieldsInGroup[currentAlignGroup] = field.Name
			}
		}

		// check field alignment
		if field.Offset%field.Align != 0 {
			t.Errorf("field %s is not properly aligned. Offset %d is not divisible by alignment %d",
				field.Name, field.Offset, field.Align)
		}

		seenFields[field.Name] = true
	}

	// check alignment groups are in the expected order
	if len(alignmentGroups) != len(expectedAlignmentGroups) {
		t.Errorf("expected %d alignment groups, got %d", len(expectedAlignmentGroups), len(alignmentGroups))
	} else {
		for i, align := range expectedAlignmentGroups {
			if i >= len(alignmentGroups) {
				t.Errorf("missing alignment group %d", align)
				continue
			}
			if alignmentGroups[i] != align {
				t.Errorf("alignment group at position %d: expected %d, got %d",
					i, align, alignmentGroups[i])
			}
		}
	}

	// check that expected fields are at the start of their respective groups
	for align, expectedField := range expectedGroupStarters {
		if actualField, exists := firstFieldsInGroup[align]; !exists {
			t.Errorf("no field found for alignment group %d", align)
		} else if actualField != expectedField {
			t.Errorf("expected field %s at start of alignment group %d, got %s",
				expectedField, align, actualField)
		}
	}

	if len(seenFields) != countNonPaddingFields(sampleResult.Fields) {
		t.Errorf("expected %d fields in optimized layout, got %d",
			countNonPaddingFields(sampleResult.Fields), len(seenFields))
	}

	t.Logf("Original: %d bytes total, %d bytes wasted (%.2f%%)",
		sampleResult.OriginalSize, origWasted, origWastedPercent)
	t.Logf("Optimized: %d bytes total, %d bytes wasted (%.2f%%)",
		sampleResult.OptimizedSize, optWasted, optWastedPercent)
	t.Logf("Size reduction: %.2f%%", sizeReductionPct)

	t.Logf("Field alignment groups:")
	for align, field := range firstFieldsInGroup {
		t.Logf("  Alignment %d: first field is %s", align, field)
	}
}

func TestRegression_GreedyPacking(t *testing.T) {
	const (
		expectedOriginalSize     int64   = 368
		expectedOptimizedSize    int64   = 336
		expectedOrigWasted       int64   = 50
		expectedOptWasted        int64   = 18
		expectedOrigWastedPct    float64 = 13.59
		expectedOptWastedPct     float64 = 5.36
		expectedSizeReductionPct float64 = 8.7
	)

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "sample_struct.go")

	err := os.WriteFile(tempFile, []byte(sampleFile), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	results, err := analyzeFileForTest(tempFile, []string{"greedy-packing"})
	if err != nil {
		t.Fatalf("failed to analyze struct: %v", err)
	}

	var sampleResult Info
	found := false
	for _, r := range results {
		if r.Name == "SampleStruct" {
			sampleResult = r
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("SampleStruct not found in results")
	}
	if sampleResult.Name != "SampleStruct" {
		t.Fatalf("expected struct name 'SampleStruct', got '%s'", sampleResult.Name)
	}

	if sampleResult.OriginalSize != expectedOriginalSize {
		t.Errorf("original size: got %d, expected %d", sampleResult.OriginalSize, expectedOriginalSize)
	}

	if sampleResult.OptimizedSize != expectedOptimizedSize {
		t.Errorf("optimized size: got %d, expected %d", sampleResult.OptimizedSize, expectedOptimizedSize)
	}

	origWasted, origWastedPercent := sampleResult.WastedSpace()
	if origWasted != expectedOrigWasted {
		t.Errorf("original wasted bytes: got %d, expected %d", origWasted, expectedOrigWasted)
	}

	if !floatEquals(origWastedPercent, expectedOrigWastedPct, 0.01) {
		t.Errorf("original wasted percentage: got %.2f%%, expected %.2f%%", origWastedPercent, expectedOrigWastedPct)
	}

	optWasted, optWastedPercent := sampleResult.OptimizedWastedSpace()
	if optWasted != expectedOptWasted {
		t.Errorf("optimized wasted bytes: got %d, expected %d", optWasted, expectedOptWasted)
	}

	if !floatEquals(optWastedPercent, expectedOptWastedPct, 0.01) {
		t.Errorf("optimized wasted percentage: got %.2f%%, expected %.2f%%", optWastedPercent, expectedOptWastedPct)
	}

	sizeReductionPct := float64(sampleResult.OriginalSize-sampleResult.OptimizedSize) / float64(sampleResult.OriginalSize) * 100
	if !floatEquals(sizeReductionPct, expectedSizeReductionPct, 0.1) {
		t.Errorf("size reduction percentage: got %.2f%%, expected %.2f%%", sizeReductionPct, expectedSizeReductionPct)
	}

	// for greedy packing, check that all fields are present and properly aligned

	// track the non-padding fields we've seen and their positions
	seenFields := make(map[string]bool)
	fieldPositions := make(map[string]int)
	var lastOffset int64 = 0

	fieldsByAlign := make(map[int64][]Field)

	var fieldIndex int
	for i, field := range sampleResult.OptimizedFields {
		if field.IsPadding {
			continue
		}

		if field.Offset%field.Align != 0 {
			t.Errorf("field %s is not properly aligned: offset %d is not divisible by alignment %d",
				field.Name, field.Offset, field.Align)
		}

		// ensure offsets are increasing (no overlapping fields)
		if field.Offset < lastOffset {
			t.Errorf("field %s at offset %d comes before previous field ending at offset %d",
				field.Name, field.Offset, lastOffset)
		}

		lastOffset = field.Offset + field.Size
		seenFields[field.Name] = true
		fieldPositions[field.Name] = i
		fieldIndex++

		fieldsByAlign[field.Align] = append(fieldsByAlign[field.Align], field)
	}

	originalFieldCount := countNonPaddingFields(sampleResult.Fields)
	if len(seenFields) != originalFieldCount {
		t.Errorf("optimized layout has %d fields, expected %d (original count)",
			len(seenFields), originalFieldCount)
	}

	for _, field := range sampleResult.OptimizedFields {
		if field.IsPadding {
			continue
		}

		// 8-byte aligned fields should be at 8-byte boundaries
		if field.Align == 8 && field.Offset%8 != 0 {
			t.Errorf("8-byte aligned field %s is at offset %d, which is not an 8-byte boundary",
				field.Name, field.Offset)
		}

		// 4-byte aligned fields should be at 4-byte boundaries
		if field.Align == 4 && field.Offset%4 != 0 {
			t.Errorf("4-byte aligned field %s is at offset %d, which is not a 4-byte boundary",
				field.Name, field.Offset)
		}

		// 2-byte aligned fields should be at 2-byte boundaries
		if field.Align == 2 && field.Offset%2 != 0 {
			t.Errorf("2-byte aligned field %s is at offset %d, which is not a 2-byte boundary",
				field.Name, field.Offset)
		}
	}

	// Check specific field type groupings. For greedy packing, similar-sized fields should generally be near each other
	reasonableDistance := 10
	timeFieldsPresent := false
	if pos1, ok1 := fieldPositions["LastLoginTime"]; ok1 {
		if pos2, ok2 := fieldPositions["MemberSince"]; ok2 {
			timeFieldsPresent = true
			posDiff := abs(pos1 - pos2)
			// note:  the exact distance can vary with the specific algorithm
			if posDiff > reasonableDistance {
				t.Logf("warning: Time fields are far apart (%d positions between LastLoginTime and MemberSince)",
					posDiff)
			}
		}
	}

	if timeFieldsPresent {
		t.Logf("time fields present and checked for proximity")
	}

	// Boolean fields (1 byte) should generally be grouped together
	// Collect positions of boolean fields
	boolFieldPositions := []int{}
	for _, fieldName := range []string{"Activated", "PremiumMember", "EmailVerified"} {
		if pos, ok := fieldPositions[fieldName]; ok {
			boolFieldPositions = append(boolFieldPositions, pos)
		}
	}

	// calculate the spread of boolean fields
	if len(boolFieldPositions) >= 2 {
		sort.Ints(boolFieldPositions)
		boolFieldSpread := boolFieldPositions[len(boolFieldPositions)-1] - boolFieldPositions[0]
		t.Logf("boolean fields spread: %d positions between first and last", boolFieldSpread)

		// soft check because the optimal layout might sometimes separate bool fields
		if boolFieldSpread > 15 && boolFieldSpread > originalFieldCount/2 {
			t.Logf("warning: Boolean fields are unusually spread out")
		}
	}

	// verify that the algorithm efficiently groups similarly sized fields: for each alignment group,
	// check if there's a continuous block of fields with minimal padding
	for align, fields := range fieldsByAlign {
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Offset < fields[j].Offset
		})

		// check for gaps between fields of the same alignment
		var paddingBetweenSameAlignCount int
		for i := 0; i < len(fields)-1; i++ {
			current := fields[i]
			next := fields[i+1]

			gap := next.Offset - (current.Offset + current.Size)

			// note: if there's a gap and the next field could have been placed immediately after
			// (considering alignment requirements), that's inefficient packing
			if gap > 0 && (current.Offset+current.Size)%next.Align == 0 {
				paddingBetweenSameAlignCount++
				t.Logf("potential inefficiency: %d bytes gap between %s and %s (same alignment group %d)",
					gap, current.Name, next.Name, align)
			}
		}

		// too many gaps between fields of the same alignment is a sign of inefficient packing
		if paddingBetweenSameAlignCount > len(fields)/2 {
			t.Logf("warning: Found %d gaps between fields in alignment group %d (total fields: %d)",
				paddingBetweenSameAlignCount, align, len(fields))
		}
	}

	// checking the total amount of padding compared to original
	var originalPaddingCount, optimizedPaddingCount int
	var originalPaddingSize, optimizedPaddingSize int64

	for _, field := range sampleResult.Fields {
		if field.IsPadding {
			originalPaddingCount++
			originalPaddingSize += field.Size
		}
	}

	for _, field := range sampleResult.OptimizedFields {
		if field.IsPadding {
			optimizedPaddingCount++
			optimizedPaddingSize += field.Size
		}
	}

	if optimizedPaddingSize >= originalPaddingSize {
		t.Errorf("optimized layout has %d bytes of padding, original had %d bytes - optimization failed",
			optimizedPaddingSize, originalPaddingSize)
	}

	minExpectedReductionPercentage := 5.0
	if sizeReductionPct < minExpectedReductionPercentage {
		t.Errorf("size reduction of %.2f%% is less than the minimum expected %.2f%%",
			sizeReductionPct, minExpectedReductionPercentage)
	}

	t.Logf("original: %d bytes total, %d bytes wasted (%.2f%%), %d padding fields, %d padding bytes",
		sampleResult.OriginalSize, origWasted, origWastedPercent, originalPaddingCount, originalPaddingSize)
	t.Logf("optimized: %d bytes total, %d bytes wasted (%.2f%%), %d padding fields, %d padding bytes",
		sampleResult.OptimizedSize, optWasted, optWastedPercent, optimizedPaddingCount, optimizedPaddingSize)
	t.Logf("size reduction: %.2f%%", sizeReductionPct)
}

func TestRegression_SizeThenAlignment(t *testing.T) {
	const (
		expectedOriginalSize     int64   = 368
		expectedOptimizedSize    int64   = 336
		expectedOrigWasted       int64   = 50
		expectedOptWasted        int64   = 18
		expectedOrigWastedPct    float64 = 13.59
		expectedOptWastedPct     float64 = 5.36
		expectedSizeReductionPct float64 = 8.7
	)

	// for size-then-alignment strategy, the expected order is by field size first
	// this optimizer prioritizes larger fields, so we expect fields like time.Time (24 bytes)
	// and strings (16 bytes) to come before smaller fields

	// actual size groups - may include composite struct sizes as well
	expectedSizeGroups := []int64{57, 48, 41, 24, 16, 8, 4, 2, 1}

	// expected fields at the beginning of each size group
	expectedGroupStarters := map[int64]string{
		57: "AddressInfo",       // struct with multiple fields
		48: "SecurityQuestions", // array of strings
		41: "PersonalDetails",   // nested struct
		24: "LastLoginTime",     // time fields (24 bytes)
		16: "Username",          // string fields (16 bytes)
		8:  "Preferences",       // 8-byte fields like pointers/maps
		4:  "UserID",            // 4-byte fields like uint32
		2:  "LoginAttempts",     // 2-byte fields like uint16
		1:  "PremiumMember",     // 1-byte fields like bool, byte, uint8
	}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "sample_struct.go")

	err := os.WriteFile(tempFile, []byte(sampleFile), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	results, err := analyzeFileForTest(tempFile, []string{"size-then-alignment"})
	if err != nil {
		t.Fatalf("failed to analyze struct: %v", err)
	}

	var sampleResult Info
	found := false
	for _, r := range results {
		if r.Name == "SampleStruct" {
			sampleResult = r
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("SampleStruct not found in results")
	}
	if sampleResult.Name != "SampleStruct" {
		t.Fatalf("expected struct name 'SampleStruct', got '%s'", sampleResult.Name)
	}

	if sampleResult.OriginalSize != expectedOriginalSize {
		t.Errorf("original size: got %d, expected %d", sampleResult.OriginalSize, expectedOriginalSize)
	}

	if sampleResult.OptimizedSize != expectedOptimizedSize {
		t.Errorf("optimized size: got %d, expected %d", sampleResult.OptimizedSize, expectedOptimizedSize)
	}

	origWasted, origWastedPercent := sampleResult.WastedSpace()
	if origWasted != expectedOrigWasted {
		t.Errorf("original wasted bytes: got %d, expected %d", origWasted, expectedOrigWasted)
	}

	if !floatEquals(origWastedPercent, expectedOrigWastedPct, 0.01) {
		t.Errorf("original wasted percentage: got %.2f%%, expected %.2f%%", origWastedPercent, expectedOrigWastedPct)
	}

	optWasted, optWastedPercent := sampleResult.OptimizedWastedSpace()
	if optWasted != expectedOptWasted {
		t.Errorf("optimized wasted bytes: got %d, expected %d", optWasted, expectedOptWasted)
	}

	if !floatEquals(optWastedPercent, expectedOptWastedPct, 0.01) {
		t.Errorf("optimized wasted percentage: got %.2f%%, expected %.2f%%", optWastedPercent, expectedOptWastedPct)
	}

	sizeReductionPct := float64(sampleResult.OriginalSize-sampleResult.OptimizedSize) / float64(sampleResult.OriginalSize) * 100
	if !floatEquals(sizeReductionPct, expectedSizeReductionPct, 0.1) {
		t.Errorf("size reduction percentage: got %.2f%%, expected %.2f%%", sizeReductionPct, expectedSizeReductionPct)
	}

	// check field size groups and order
	var currentSizeGroup int64 = -1
	var sizeGroups []int64
	var firstFieldsInGroup = make(map[int64]string)
	var seenFields = make(map[string]bool)

	// track non-padding fields and their sizes
	fieldsBySize := make(map[int64][]Field)

	for _, field := range sampleResult.OptimizedFields {
		if field.IsPadding {
			continue
		}

		// record the first field we see for each size group
		if field.Size != currentSizeGroup {
			currentSizeGroup = field.Size
			sizeGroups = append(sizeGroups, currentSizeGroup)

			// record the first field name for this size group
			if _, exists := firstFieldsInGroup[currentSizeGroup]; !exists {
				firstFieldsInGroup[currentSizeGroup] = field.Name
			}
		}

		// check field alignment
		if field.Offset%field.Align != 0 {
			t.Errorf("field %s is not properly aligned. offset %d is not divisible by alignment %d",
				field.Name, field.Offset, field.Align)
		}

		seenFields[field.Name] = true
		fieldsBySize[field.Size] = append(fieldsBySize[field.Size], field)
	}

	// check size groups are in the expected order
	if len(sizeGroups) != len(expectedSizeGroups) {
		t.Errorf("expected %d size groups, got %d", len(expectedSizeGroups), len(sizeGroups))
	} else {
		for i, size := range expectedSizeGroups {
			if i >= len(sizeGroups) {
				t.Errorf("missing size group %d", size)
				continue
			}
			if sizeGroups[i] != size {
				t.Errorf("size group at position %d: expected %d, got %d",
					i, size, sizeGroups[i])
			}
		}
	}

	// check that expected fields are at the start of their respective groups
	for size, expectedField := range expectedGroupStarters {
		if actualField, exists := firstFieldsInGroup[size]; !exists {
			t.Errorf("no field found for size group %d", size)
		} else if actualField != expectedField {
			t.Errorf("expected field %s at start of size group %d, got %s",
				expectedField, size, actualField)
		}
	}

	// for fields of the same size, check they're ordered by alignment (largest first)
	for size, fields := range fieldsBySize {
		if len(fields) <= 1 {
			continue // nothing to check if there's only one field
		}

		// sort fields by position in the layout
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Offset < fields[j].Offset
		})

		// check alignment ordering within size group
		for i := 0; i < len(fields)-1; i++ {
			current := fields[i]
			next := fields[i+1]

			// within a size group, higher alignments should come first
			if current.Align < next.Align {
				t.Logf("warning: in size group %d, field %s (align %d) comes before %s (align %d) - suboptimal alignment ordering",
					size, current.Name, current.Align, next.Name, next.Align)
			}
		}
	}

	if len(seenFields) != countNonPaddingFields(sampleResult.Fields) {
		t.Errorf("expected %d fields in optimized layout, got %d",
			countNonPaddingFields(sampleResult.Fields), len(seenFields))
	}

	// check that the struct size reduction is significant enough
	minExpectedReductionPercentage := 5.0
	if sizeReductionPct < minExpectedReductionPercentage {
		t.Errorf("size reduction of %.2f%% is less than the minimum expected %.2f%%",
			sizeReductionPct, minExpectedReductionPercentage)
	}

	t.Logf("original: %d bytes total, %d bytes wasted (%.2f%%)",
		sampleResult.OriginalSize, origWasted, origWastedPercent)
	t.Logf("optimized: %d bytes total, %d bytes wasted (%.2f%%)",
		sampleResult.OptimizedSize, optWasted, optWastedPercent)
	t.Logf("size reduction: %.2f%%", sizeReductionPct)

	t.Logf("field size groups:")
	for size, field := range firstFieldsInGroup {
		t.Logf("  size %d: first field is %s", size, field)
	}
}

func TestRegression_GroupByAlignment(t *testing.T) {
	const (
		expectedOriginalSize     int64   = 368
		expectedOptimizedSize    int64   = 336
		expectedOrigWasted       int64   = 50
		expectedOptWasted        int64   = 18
		expectedOrigWastedPct    float64 = 13.59
		expectedOptWastedPct     float64 = 5.36
		expectedSizeReductionPct float64 = 8.7
	)

	// with the group-by-alignment strategy, we expect fields to be grouped by their
	// alignment requirements, with higher alignments first
	expectedAlignmentGroups := []int64{8, 4, 2, 1}

	// expected fields at the beginning of each alignment group
	// based on actual implementation results
	expectedGroupStarters := map[int64]string{
		8: "AddressInfo",   // an 8-byte aligned field at the start of 8-byte group
		4: "UserID",        // a 4-byte aligned field at the start of 4-byte group
		2: "LoginAttempts", // a 2-byte aligned field at the start of 2-byte group
		1: "SessionToken",  // a 1-byte aligned field at the start of 1-byte group
	}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "XX_sample_struct.go")

	err := os.WriteFile(tempFile, []byte(sampleFile), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	results, err := analyzeFileForTest(tempFile, []string{"group-by-alignment"})
	if err != nil {
		t.Fatalf("failed to analyze struct: %v", err)
	}

	var sampleResult Info
	found := false
	for _, r := range results {
		if r.Name == "SampleStruct" {
			sampleResult = r
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("SampleStruct not found in results")
	}
	if sampleResult.Name != "SampleStruct" {
		t.Fatalf("expected struct name 'SampleStruct', got '%s'", sampleResult.Name)
	}

	if sampleResult.OriginalSize != expectedOriginalSize {
		t.Errorf("original size: got %d, expected %d", sampleResult.OriginalSize, expectedOriginalSize)
	}

	if sampleResult.OptimizedSize != expectedOptimizedSize {
		t.Errorf("optimized size: got %d, expected %d", sampleResult.OptimizedSize, expectedOptimizedSize)
	}

	origWasted, origWastedPercent := sampleResult.WastedSpace()
	if origWasted != expectedOrigWasted {
		t.Errorf("original wasted bytes: got %d, expected %d", origWasted, expectedOrigWasted)
	}

	if !floatEquals(origWastedPercent, expectedOrigWastedPct, 0.01) {
		t.Errorf("original wasted percentage: got %.2f%%, expected %.2f%%", origWastedPercent, expectedOrigWastedPct)
	}

	optWasted, optWastedPercent := sampleResult.OptimizedWastedSpace()
	if optWasted != expectedOptWasted {
		t.Errorf("optimized wasted bytes: got %d, expected %d", optWasted, expectedOptWasted)
	}

	if !floatEquals(optWastedPercent, expectedOptWastedPct, 0.01) {
		t.Errorf("optimized wasted percentage: got %.2f%%, expected %.2f%%", optWastedPercent, expectedOptWastedPct)
	}

	sizeReductionPct := float64(sampleResult.OriginalSize-sampleResult.OptimizedSize) / float64(sampleResult.OriginalSize) * 100
	if !floatEquals(sizeReductionPct, expectedSizeReductionPct, 0.1) {
		t.Errorf("size reduction percentage: got %.2f%%, expected %.2f%%", sizeReductionPct, expectedSizeReductionPct)
	}

	// check field alignment groups and order
	var currentAlignGroup int64 = -1
	var alignmentGroups []int64
	var firstFieldsInGroup = make(map[int64]string)
	var seenFields = make(map[string]bool)
	var lastFieldAlign int64 = -1

	// track non-padding fields by alignment for later analysis
	fieldsByAlign := make(map[int64][]Field)

	for _, field := range sampleResult.OptimizedFields {
		if field.IsPadding {
			continue
		}

		// check that fields with the same alignment are grouped together
		if field.Align != lastFieldAlign && lastFieldAlign != -1 {
			// we've moved to a new alignment group
			if field.Align > lastFieldAlign {
				t.Errorf("field %s with alignment %d comes after smaller alignment %d - should be ordered by decreasing alignment",
					field.Name, field.Align, lastFieldAlign)
			}
		}

		// record the first field we see for each alignment group
		if field.Align != currentAlignGroup {
			currentAlignGroup = field.Align
			alignmentGroups = append(alignmentGroups, currentAlignGroup)

			// record the first field name for this alignment group
			if _, exists := firstFieldsInGroup[currentAlignGroup]; !exists {
				firstFieldsInGroup[currentAlignGroup] = field.Name
			}
		}

		// check field alignment
		if field.Offset%field.Align != 0 {
			t.Errorf("field %s is not properly aligned. offset %d is not divisible by alignment %d",
				field.Name, field.Offset, field.Align)
		}

		lastFieldAlign = field.Align
		seenFields[field.Name] = true
		fieldsByAlign[field.Align] = append(fieldsByAlign[field.Align], field)
	}

	// check alignment groups are in the expected order
	if len(alignmentGroups) != len(expectedAlignmentGroups) {
		t.Errorf("expected %d alignment groups, got %d", len(expectedAlignmentGroups), len(alignmentGroups))
	} else {
		for i, align := range expectedAlignmentGroups {
			if i >= len(alignmentGroups) {
				t.Errorf("missing alignment group %d", align)
				continue
			}
			if alignmentGroups[i] != align {
				t.Errorf("alignment group at position %d: expected %d, got %d",
					i, align, alignmentGroups[i])
			}
		}
	}

	// check that expected fields are at the start of their respective groups
	for align, expectedField := range expectedGroupStarters {
		if actualField, exists := firstFieldsInGroup[align]; !exists {
			t.Errorf("no field found for alignment group %d", align)
		} else if actualField != expectedField {
			t.Errorf("expected field %s at start of alignment group %d, got %s",
				expectedField, align, actualField)
		}
	}

	// for fields of the same alignment, check they're ordered by size (largest first)
	for align, fields := range fieldsByAlign {
		if len(fields) <= 1 {
			continue // nothing to check if there's only one field
		}

		// sort fields by position in the layout
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Offset < fields[j].Offset
		})

		// check size ordering within alignment group
		for i := 0; i < len(fields)-1; i++ {
			current := fields[i]
			next := fields[i+1]

			// for the group-by-alignment strategy, size ordering within alignment groups may vary
			// depending on the implementation, so this is just a warning, not an error
			if current.Size < next.Size {
				t.Logf("note: in alignment group %d, field %s (size %d) comes before %s (size %d) - suboptimal size ordering",
					align, current.Name, current.Size, next.Name, next.Size)
			}
		}
	}

	if len(seenFields) != countNonPaddingFields(sampleResult.Fields) {
		t.Errorf("expected %d fields in optimized layout, got %d",
			countNonPaddingFields(sampleResult.Fields), len(seenFields))
	}

	// check for inefficient packing
	var totalGapSize int64
	for align, fields := range fieldsByAlign {
		if len(fields) <= 1 {
			continue
		}

		// fields should already be sorted by position
		for i := 0; i < len(fields)-1; i++ {
			current := fields[i]
			next := fields[i+1]

			// calculate gap between adjacent fields in same alignment group
			gap := next.Offset - (current.Offset + current.Size)
			if gap > 0 {
				totalGapSize += gap
				t.Logf("potential inefficiency: %d bytes gap between %s and %s (same alignment group %d)",
					gap, current.Name, next.Name, align)
			}
		}
	}

	// don't fail the test for gaps, but log if they're significant
	if totalGapSize > 0 {
		t.Logf("total gaps between fields of same alignment: %d bytes", totalGapSize)
	}

	// check that the struct size reduction is significant enough
	minExpectedReductionPercentage := 5.0
	if sizeReductionPct < minExpectedReductionPercentage {
		t.Errorf("size reduction of %.2f%% is less than the minimum expected %.2f%%",
			sizeReductionPct, minExpectedReductionPercentage)
	}

	t.Logf("original: %d bytes total, %d bytes wasted (%.2f%%)",
		sampleResult.OriginalSize, origWasted, origWastedPercent)
	t.Logf("optimized: %d bytes total, %d bytes wasted (%.2f%%)",
		sampleResult.OptimizedSize, optWasted, optWastedPercent)
	t.Logf("size reduction: %.2f%%", sizeReductionPct)

	t.Logf("field alignment groups:")
	for align, field := range firstFieldsInGroup {
		t.Logf("  alignment %d: first field is %s", align, field)
	}
}

func TestRegression_EmptyStructTailPadding(t *testing.T) {
	src := `package test

type TrailingEmpty struct {
	A int64
	B int64
	C struct{}
}

type MiddleEmpty struct {
	A int64
	B struct{}
	C int64
}

type MultipleTrailingEmpty struct {
	A int64
	B struct{}
	C struct{}
}
`
	results, err := AnalyseStructsAsStringWithStrategies(src, nil)
	if err != nil {
		t.Fatalf("failed to analyze: %v", err)
	}

	expected := map[string]int64{
		"TrailingEmpty":         24, // 8 + 8 + 0 + 8 tail padding
		"MiddleEmpty":           16, // 8 + 0 + 8, no special tail padding
		"MultipleTrailingEmpty": 16, // 8 + 0 + 0 + 8 tail padding (last field is zero-size)
	}

	for _, r := range results {
		if want, ok := expected[r.Name]; ok {
			if r.OriginalSize != want {
				t.Errorf("%s: expected original size %d, got %d", r.Name, want, r.OriginalSize)
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
