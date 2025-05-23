package template

// returning it as a string because functions like os.Getwd and os.Stat
// are unsupported in the WebAssembly runtime environment (e.g., browsers)
var (
	StructLayoutTemplate = `{{define "struct_layout"}}
{{$lastFieldY := 0.0}}

{{/* Calculate the needed height based on content */}}
{{$totalFields := len .Fields}}
{{$totalOptimizedFields := len .OptimizedFields}}
{{$totalBreakdownFields := len .FieldBreakdown}}
{{$totalCodeLines := len .OptimizedFieldsCode}}

{{/* Calculate estimated height */}}
{{$heightForFields := mul 40.0 (float64 $totalFields)}}
{{$heightForBreakdown := mul 15.0 (float64 $totalBreakdownFields)}}
{{$heightForOptimizedFields := mul 40.0 (float64 $totalOptimizedFields)}}
{{$heightForCode := mul 15.0 (float64 $totalCodeLines)}}

{{/* Base height plus calculated content height */}}
{{$baseHeight := 500.0}}
{{$calculatedHeight := add $baseHeight $heightForFields}}
{{$calculatedHeight := add $calculatedHeight $heightForBreakdown}}
{{$calculatedHeight := add $calculatedHeight $heightForOptimizedFields}}
{{$calculatedHeight := add $calculatedHeight $heightForCode}}

{{/* Minimum height of 1700, or calculated if larger */}}
{{$svgHeight := 1700.0}}
{{if gt $calculatedHeight $svgHeight}}
  {{$svgHeight = $calculatedHeight}}
{{end}}

{{/* Add extra padding to ensure everything fits */}}
{{$svgHeight := add $svgHeight 300.0}}

<svg width="1200" height="{{$svgHeight}}" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
		<style>
			.field-text { font-family: Arial, sans-serif; font-size: 14px; fill: #000000; }
			.field-label { font-family: Arial, sans-serif; font-size: 14px; fill: #000000; text-anchor: end; }
			.struct-name { font-family: Arial, sans-serif; font-size: 16px; font-weight: bold; fill: #000000; }
			.offset-text { font-family: Arial, sans-serif; font-size: 12px; fill: #000000; }
			.size-text { font-family: Arial, sans-serif; font-size: 12px; fill: #000000; }
			.padding-pattern { fill: #CCCCCC; fill-opacity: 0.3; }
		</style>
		<rect width="100%" height="100%" fill="white"/>
	<text x="10" y="50" class="struct-name" fill="#000000">{{.Name}}</text>
<text x="10" y="70" class="field-text" fill="#000000">Total size: {{.TotalSize}} bytes | Wasted: {{.WastedBytes}} bytes ({{.WastedPercent}}%)</text>
<text x="10" y="90" class="field-text" fill="#000000">Original layout:</text>

<!-- Original Layout (Vertical) -->
<text x="200" y="120" class="offset-text" fill="#000000">Offset</text>
<text x="300" y="120" class="offset-text" fill="#000000">Field</text>
<text x="650" y="120" class="offset-text" fill="#000000">Size</text>

{{$xPos := 300.0}}
{{$barWidth := 350.0}}
{{$yOffset := 140.0}}
{{$rowHeight := 40.0}}
{{$currentOffset := 0}}

{{range .Fields}}
<!-- Field offset marker -->
{{$yPos := add $yOffset (div $rowHeight 2.0)}}
{{$yPos := add $yPos 5.0}}
<text x="200" y="{{$yPos}}" class="offset-text" text-anchor="end" fill="#000000">{{.Offset}}</text>

<!-- Field name -->
<text x="280" y="{{$yPos}}" class="field-label" fill="#000000">{{.Name}}</text>

<!-- Field bar -->
<rect x="{{$xPos}}" y="{{$yOffset}}" width="{{$barWidth}}" height="{{$rowHeight}}" fill="{{.Color}}" stroke="{{if .IsPadding}}gray{{else}}black{{end}}" stroke-width="1" {{if .IsPadding}}stroke-dasharray="5,5"{{end}}/>

<!-- Field size -->
{{$labelX := add $xPos $barWidth}}
{{$labelX := add $labelX 20.0}}
<text x="{{$labelX}}" y="{{$yPos}}" class="size-text" fill="#000000">{{.Size}} bytes</text>

<!-- Update for next row -->
{{$yOffset = add $yOffset $rowHeight}}
{{end}}

<!-- End offset marker -->
<text x="200" y="{{$yOffset}}" class="offset-text" text-anchor="end" fill="#000000">{{.TotalSize}}</text>

<!-- Field breakdown -->
{{$breakdownY := add $yOffset 50.0}}
<text x="10" y="{{$breakdownY}}" class="field-text" fill="#000000">Field breakdown:</text>
{{range $i, $f := .FieldBreakdown}}
{{$itemY := add $yOffset 70.0}}
{{$itemY := add $itemY (mul (float64 $i) 15.0)}}
<text x="10" y="{{$itemY}}" class="field-text" fill="{{if $f.IsPadding}}#FF0000{{else}}#000000{{end}}">{{$f.Text}}</text>
{{end}}

<!-- Optimized Layout (Vertical) -->
{{$optimizedYOffset := add $yOffset 100.0}}
{{$optimizedYOffset := add $optimizedYOffset (mul (float64 (len .FieldBreakdown)) 15.0)}}
<text x="10" y="{{$optimizedYOffset}}" class="field-text" fill="#000000">Optimized layout: {{.OptimizedSize}} bytes (saved {{.SavedBytes}} bytes, {{.OptimizedWastePercent}}% waste)</text>

{{$headerY := add $optimizedYOffset 30.0}}
<text x="200" y="{{$headerY}}" class="offset-text" fill="#000000">Offset</text>
<text x="300" y="{{$headerY}}" class="offset-text" fill="#000000">Field</text>
<text x="650" y="{{$headerY}}" class="offset-text" fill="#000000">Size</text>

{{$yOffset = add $optimizedYOffset 50.0}}
{{range .OptimizedFields}}
<!-- Field offset marker -->
{{$yPos := add $yOffset (div $rowHeight 2.0)}}
{{$yPos := add $yPos 5.0}}
<text x="200" y="{{$yPos}}" class="offset-text" text-anchor="end" fill="#000000">{{.Offset}}</text>

<!-- Field name -->
<text x="280" y="{{$yPos}}" class="field-label" fill="#000000">{{.Name}}</text>

<!-- Field bar -->
<rect x="{{$xPos}}" y="{{$yOffset}}" width="{{$barWidth}}" height="{{$rowHeight}}" fill="{{.Color}}" stroke="{{if .IsPadding}}gray{{else}}black{{end}}" stroke-width="1" {{if .IsPadding}}stroke-dasharray="5,5"{{end}}/>

<!-- Field size -->
{{$labelX := add $xPos $barWidth}}
{{$labelX := add $labelX 20.0}}
<text x="{{$labelX}}" y="{{$yPos}}" class="size-text" fill="#000000">{{.Size}} bytes</text>

<!-- Update for next row -->
{{$yOffset = add $yOffset $rowHeight}}
{{end}}

<!-- End offset marker -->
<text x="200" y="{{$yOffset}}" class="offset-text" text-anchor="end" fill="#000000">{{.OptimizedSize}}</text>

<!-- Suggested Code -->
{{$codeY := add $yOffset 50.0}}
<text x="10" y="{{$codeY}}" class="field-text" fill="#000000">Suggested code:</text>
{{$codeHeaderY := add $yOffset 70.0}}
<text x="10" y="{{$codeHeaderY}}" class="field-text" fill="#000000">type {{.Name}}Optimized struct {</text>
{{range $i, $f := .OptimizedFieldsCode}}
{{$lineY := add $yOffset 85.0}}
{{$lineY := add $lineY (mul (float64 $i) 15.0)}}
<text x="10" y="{{$lineY}}" class="field-text" fill="#000000">    {{$f}}</text>
{{end}}
{{$endY := add $yOffset 85.0}}
{{$endY := add $endY (mul (float64 (len .OptimizedFieldsCode)) 15.0)}}
<text x="10" y="{{$endY}}" class="field-text" fill="#000000">}</text>
</svg>
{{end}}`
)
