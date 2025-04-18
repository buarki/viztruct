package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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
	// just prepend package declaration if needed
	if !strings.Contains(structCode, "package") {
		structCode = "package temp\n\n" + structCode
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "input.go", structCode, parser.AllErrors)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse: %v", err)
	}

	conf := types.Config{Importer: nil, Sizes: &customSizes}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	_, err = conf.Check("temp", fset, []*ast.File{node}, info)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to type-check: %v", err)
	}

	structInfos := structi.AnalyzeNestedStructs(node, &customSizes, info, fset)

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
