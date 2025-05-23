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

	svgBytes, optimizedCode, byteSavings, err := generateSVGAndCode(structCode)
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
		"byteSavings":   byteSavings,
	})
}

func generateSVGAndCode(structCode string) ([]byte, []byte, map[string]any, error) {
	structInfos, err := structi.AnalyseStructsAsStringWithStrategies(structCode, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	svgContent, err := svg.BuildSingleVisualization(structInfos)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build svg: %v", err)
	}

	var optimizedCode strings.Builder
	optimizedCode.WriteString("// Optimized struct definitions:\n\n")

	var totalOriginalSize int64
	var totalOptimizedSize int64
	byteSavings := make(map[string]any)

	for _, si := range structInfos {
		optimizedCode.WriteString(fmt.Sprintf("type %s struct {\n", si.Name+"Optimized"))
		for _, field := range si.OptimizedFields {
			if !field.IsPadding {
				optimizedCode.WriteString(fmt.Sprintf("\t%s %s\n", field.Name, field.TypeName))
			}
		}
		optimizedCode.WriteString("}\n\n")

		totalOriginalSize += si.OriginalSize
		totalOptimizedSize += si.OptimizedSize
	}

	bytesSavedPerStruct := totalOriginalSize - totalOptimizedSize
	byteSavingsAtScale := bytesSavedPerStruct * 1000000 // 1 million structs

	byteSavings["perStruct"] = bytesSavedPerStruct
	byteSavings["perMillion"] = byteSavingsAtScale
	byteSavings["originalSize"] = totalOriginalSize
	byteSavings["optimizedSize"] = totalOptimizedSize

	return []byte(svgContent), []byte(optimizedCode.String()), byteSavings, nil
}
