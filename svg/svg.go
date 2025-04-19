package svg

import (
	"bytes"
	"fmt"
	"html/template"

	svgTemplate "github.com/buarki/viztruct/internal/viz/template"
	"github.com/buarki/viztruct/structi"
)

const (
	blockHeight = 40
	paddingX    = 10
)

var typeColors = map[string]string{
	"uint64":       "#4285F4", // blue
	"uint32":       "#34A853", // green
	"uint16":       "#FBBC05", // yellow
	"uint8":        "#EA4335", // red
	"int64":        "#4285F4", // blue
	"int32":        "#34A853", // green
	"int16":        "#FBBC05", // yellow
	"int8":         "#EA4335", // red
	"bool":         "#9C27B0", // purple
	"string":       "#FF9800", // orange
	"byte":         "#607D8B", // blue gray
	"rune":         "#795548", // brown
	"float64":      "#0097A7", // cyan
	"float32":      "#00BCD4", // light cyan
	"padding":      "#E0E0E0", // light gray for regular padding
	"tail_padding": "#F5F5F5", // very light gray for tail padding
	"unknown":      "#AAAAAA", // default gray for unknown types
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

func BuildVisualization(structs []structi.Info) (string, error) {
	tmpl := template.New("svg_template").Funcs(template.FuncMap{
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
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
		"lt": func(a, b int64) bool { return a < b },
	})

	tmpl, err := tmpl.Parse(svgTemplate.StructLayoutTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	var result bytes.Buffer
	result.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?>\n")

	width := 1200.0 - (2 * paddingX)
	for _, structInfo := range structs {
		data := prepareTemplateData(structInfo, width)
		err = tmpl.ExecuteTemplate(&result, "struct_layout", data)
		if err != nil {
			return "", fmt.Errorf("error executing template: %v", err)
		}
	}

	return result.String(), nil
}

func prepareTemplateData(info structi.Info, width float64) TemplateData {
	wastedBytes, wastedPercent := info.WastedSpace()
	_, optimizedWastedPercent := info.OptimazedWastedSpace()
	structTotalSize := info.TotalSize()
	optimizedSize := info.OptimazedTotalSize()

	scale := width / float64(structTotalSize)
	if structTotalSize == 0 {
		scale = width // to avoid division by zero
	}

	var fields []FieldData
	for _, f := range info.Fields {
		blockX := paddingX + float64(f.Offset)*scale
		blockWidth := float64(f.Size) * scale

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
			LabelX:      blockX + blockWidth/2,
			X:           blockX,
			Width:       blockWidth,
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
		blockX := paddingX + float64(f.Offset)*scale
		blockWidth := float64(f.Size) * scale

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
			LabelX:      blockX + blockWidth/2,
			X:           blockX,
			Width:       blockWidth,
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
		OptimizedWastePercent: optimizedWastedPercent,
		Fields:                fields,
		OptimizedFields:       optimizedFields,
		FieldBreakdown:        fieldBreakdown,
		OptimizedFieldsCode:   optimizedFieldsCode,
		LastOffsetX:           paddingX + float64(structTotalSize)*scale,
		OptimizedLastX:        paddingX + float64(optimizedSize)*scale,
		BlockHeight:           float64(blockHeight),
	}
}
