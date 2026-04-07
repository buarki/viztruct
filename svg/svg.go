package svg

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	svgTemplate "github.com/buarki/viztruct/internal/viz/template"
	"github.com/buarki/viztruct/structi"
)

const (
	blockHeight = 40
	paddingX    = 10
	barWidth    = 350
)

var typeColors = map[string]string{
	"uint64":       "#6366f1", // indigo
	"uint32":       "#10b981", // emerald
	"uint16":       "#f59e0b", // amber
	"uint8":        "#ef4444", // red
	"int64":        "#6366f1", // indigo
	"int32":        "#10b981", // emerald
	"int16":        "#f59e0b", // amber
	"int8":         "#ef4444", // red
	"bool":         "#a855f7", // purple
	"string":       "#f97316", // orange
	"byte":         "#64748b", // slate
	"rune":         "#78716c", // stone
	"float64":      "#06b6d4", // cyan
	"float32":      "#22d3ee", // light cyan
	"padding":      "#d1d5db", // gray-300
	"tail_padding": "#e5e7eb", // gray-200
	"unknown":      "#9ca3af", // gray-400
}

type FieldData struct {
	Name        string
	LabelX      float64
	X           float64
	Width       float64
	Color       string
	Offset      int64
	Size        int64
	IsPadding   bool
	BlockHeight float64
}

type FieldBreakdownData struct {
	Text      string
	IsPadding bool
}

type TemplateData struct {
	Name                  string
	TotalSize             int64
	WastedBytes           int64
	WastedPercent         float64
	OptimizedSize         int64
	SavedBytes            int64
	SavedPercent          float64
	OptimizedWastePercent float64
	Fields                []FieldData
	OptimizedFields       []FieldData
	FieldBreakdown        []FieldBreakdownData
	OptimizedFieldsCode   []string
	LastOffsetX           float64
	OptimizedLastX        float64
	BlockHeight           float64
}

func getTypeColor(typeName string) string {
	if color, ok := typeColors[typeName]; ok {
		return color
	}
	return typeColors["unknown"]
}

// BuildVisualization generates SVG visualizations for each struct in the provided slice
// and returns a map of struct names to SVG content
func BuildVisualization(structs []structi.Info) (map[string]string, error) {
	tmpl := template.New("svg_template").Funcs(template.FuncMap{
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 { return a / b },
		"float64": func(i interface{}) float64 {
			switch v := i.(type) {
			case int:
				return float64(v)
			case int64:
				return float64(v)
			case float64:
				return v
			default:
				return 0
			}
		},
		"lt":      func(a, b int64) bool { return a < b },
		"printf":  func(format string, a interface{}) string { return fmt.Sprintf(format, a) },
	})

	tmpl, err := tmpl.Parse(svgTemplate.StructLayoutTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	svgMap := make(map[string]string)

	for _, structInfo := range structs {
		var result bytes.Buffer
		result.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n")

		data := prepareTemplateData(structInfo)
		err = tmpl.ExecuteTemplate(&result, "struct_layout", data)
		if err != nil {
			return nil, fmt.Errorf("error executing template for struct %s: %v", structInfo.Name, err)
		}

		fileName := sanitizeFileName(structInfo.Name)
		svgMap[fileName] = result.String()
	}

	return svgMap, nil
}

// BuildSingleVisualization generates a combined SVG visualization for all structs
func BuildSingleVisualization(structs []structi.Info) (string, error) {
	tmpl := template.New("svg_template").Funcs(template.FuncMap{
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 { return a / b },
		"float64": func(i interface{}) float64 {
			switch v := i.(type) {
			case int:
				return float64(v)
			case int64:
				return float64(v)
			case float64:
				return v
			default:
				return 0
			}
		},
		"lt":      func(a, b int64) bool { return a < b },
		"printf":  func(format string, a interface{}) string { return fmt.Sprintf(format, a) },
	})

	tmpl, err := tmpl.Parse(svgTemplate.StructLayoutTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	var result bytes.Buffer
	result.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n")

	for _, structInfo := range structs {
		data := prepareTemplateData(structInfo)
		err = tmpl.ExecuteTemplate(&result, "struct_layout", data)
		if err != nil {
			return "", fmt.Errorf("error executing template: %v", err)
		}
	}

	return result.String(), nil
}

// sanitizeFileName sanitizes a struct name to be used as a file name
func sanitizeFileName(name string) string {
	// replace invalid filename characters with underscores
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

func prepareTemplateData(info structi.Info) TemplateData {
	wastedBytes, wastedPercent := info.WastedSpace()
	_, optimizedWastedPercent := info.OptimizedWastedSpace()
	structTotalSize := info.TotalSize()
	optimizedSize := info.OptimizedTotalSize()

	var fields []FieldData
	for _, f := range info.Fields {
		color := getTypeColor(f.TypeName)
		if f.IsPadding {
			if f.Offset+f.Size == structTotalSize {
				color = getTypeColor("tail_padding")
			} else {
				color = getTypeColor("padding")
			}
		}

		field := FieldData{
			Name:        f.Name,
			LabelX:      float64(paddingX),
			X:           float64(paddingX),
			Width:       float64(barWidth),
			Color:       color,
			Offset:      f.Offset,
			Size:        f.Size,
			IsPadding:   f.IsPadding,
			BlockHeight: float64(blockHeight),
		}
		fields = append(fields, field)
	}

	var optimizedFields []FieldData
	for _, f := range info.OptimizedFields {
		color := getTypeColor(f.TypeName)
		if f.IsPadding {
			if f.Offset+f.Size == optimizedSize {
				color = getTypeColor("tail_padding")
			} else {
				color = getTypeColor("padding")
			}
		}

		field := FieldData{
			Name:        f.Name,
			LabelX:      float64(paddingX),
			X:           float64(paddingX),
			Width:       float64(barWidth),
			Color:       color,
			Offset:      f.Offset,
			Size:        f.Size,
			IsPadding:   f.IsPadding,
			BlockHeight: float64(blockHeight),
		}
		optimizedFields = append(optimizedFields, field)
	}

	var fieldBreakdown []FieldBreakdownData
	for _, f := range info.Fields {
		text := fmt.Sprintf("%s: Offset=%d, Size=%d", f.Name, f.Offset, f.Size)
		if !f.IsPadding {
			text += fmt.Sprintf(", Type=%s, Align=%d", f.TypeName, f.Align)
		}
		fieldBreakdown = append(fieldBreakdown, FieldBreakdownData{
			Text:      text,
			IsPadding: f.IsPadding,
		})
	}

	var optimizedFieldsCode []string
	for _, f := range info.OptimizedFields {
		if !f.IsPadding {
			optimizedFieldsCode = append(optimizedFieldsCode, fmt.Sprintf("%s %s", f.Name, f.TypeName))
		}
	}

	return TemplateData{
		Name:                  info.Name,
		TotalSize:             structTotalSize,
		WastedBytes:           wastedBytes,
		WastedPercent:         wastedPercent,
		OptimizedSize:         optimizedSize,
		SavedBytes:            structTotalSize - optimizedSize,
		SavedPercent:          func() float64 { if structTotalSize > 0 { return float64(structTotalSize-optimizedSize) / float64(structTotalSize) * 100 }; return 0 }(),
		OptimizedWastePercent: optimizedWastedPercent,
		Fields:                fields,
		OptimizedFields:       optimizedFields,
		FieldBreakdown:        fieldBreakdown,
		OptimizedFieldsCode:   optimizedFieldsCode,
		LastOffsetX:           float64(paddingX + barWidth),
		OptimizedLastX:        float64(paddingX + barWidth),
		BlockHeight:           float64(blockHeight),
	}
}
