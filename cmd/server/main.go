package main

import (
	"fmt"
	"go/types"
	"strings"
	"syscall/js"

	"github.com/buarki/viztruct/structi"
	"github.com/buarki/viztruct/svg"
)

var customSizes = types.StdSizes{
	WordSize: 8,
	MaxAlign: 8,
}

func main() {
	js.Global().Set("generateStructLayoutSVG", js.FuncOf(generateStructLayoutSVG))

	<-make(chan bool)
}

func generateStructLayoutSVG(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf(map[string]any{
			"error": "Missing struct definition input",
		})
	}

	// reading passed struct code from JavaScript
	structCode := args[0].String()

	svgBytes, optimizedCode, err := generateSVGAndCode(structCode)
	if err != nil {
		return js.ValueOf(map[string]any{
			"error": err.Error(),
		})
	}

	svgArray := js.Global().Get("Uint8Array").New(len(svgBytes))
	js.CopyBytesToJS(svgArray, svgBytes)

	codeArray := js.Global().Get("Uint8Array").New(len(optimizedCode))
	js.CopyBytesToJS(codeArray, []byte(optimizedCode))

	return js.ValueOf(map[string]any{
		"svg":           svgArray,
		"optimizedCode": codeArray,
	})
}

func generateSVGAndCode(structCode string) ([]byte, []byte, error) {
	structInfos, err := structi.AnalyseStructs(structCode)
	if err != nil {
		return nil, nil, err
	}

	svgContent, err := svg.BuildVisualization(structInfos)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build svg: %v", err)
	}

	var optimizedCode strings.Builder
	optimizedCode.WriteString("// Optimized struct definitions:\n\n")
	for _, si := range structInfos {
		optimizedCode.WriteString(fmt.Sprintf("type %s struct {\n", si.Name+"Optimized"))
		for _, field := range si.OptimizedFields {
			if !field.IsPadding {
				optimizedCode.WriteString(fmt.Sprintf("\t%s %s\n", field.Name, field.TypeName))
			}
		}
		optimizedCode.WriteString("}\n\n")
	}

	return []byte(svgContent), []byte(optimizedCode.String()), nil
}
