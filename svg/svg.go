package svg

const (
	blockHeight   = 40
	paddingX      = 10
	paddingY      = 10
	structSpacing = 30
	textOffsetY   = 20
	compareMode   = true // Set to true to enable side-by-side comparison
	optimizeMode  = true // Set to true to suggest optimized struct layouts
)

var typeColors = map[string]string{
	"uint64":  "#4285F4", // blue
	"uint32":  "#34A853", // green
	"uint16":  "#FBBC05", // yellow
	"uint8":   "#EA4335", // red
	"int64":   "#4285F4", // blue
	"int32":   "#34A853", // green
	"int16":   "#FBBC05", // yellow
	"int8":    "#EA4335", // red
	"bool":    "#9C27B0", // purple
	"string":  "#FF9800", // orange
	"byte":    "#607D8B", // blue gray
	"rune":    "#795548", // brown
	"float64": "#0097A7", // cyan
	"float32": "#00BCD4", // light cyan
	"":        "#CCCCCC", // gray (for padding)
}

