package template

// returning it as a string because functions like os.Getwd and os.Stat
// are unsupported in the WebAssembly runtime environment (e.g., browsers)
var (
	StructLayoutTemplate = `{{define "struct_layout"}}

{{/* Calculate the needed height based on content */}}
{{$totalFields := len .Fields}}
{{$totalOptimizedFields := len .OptimizedFields}}
{{$totalBreakdownFields := len .FieldBreakdown}}
{{$totalCodeLines := len .OptimizedFieldsCode}}

{{/* Side-by-side: use the taller column */}}
{{$heightForFields := mul 34.0 (float64 $totalFields)}}
{{$heightForOptimizedFields := mul 34.0 (float64 $totalOptimizedFields)}}
{{$maxFieldsHeight := $heightForFields}}
{{if gt $heightForOptimizedFields $heightForFields}}
  {{$maxFieldsHeight = $heightForOptimizedFields}}
{{end}}

{{$heightForBreakdown := mul 18.0 (float64 $totalBreakdownFields)}}
{{$heightForCode := mul 18.0 (float64 $totalCodeLines)}}

{{/* header(100) + stats(50) + labels+headers(50) + fields + gap(30) + breakdown + gap(30) + code */}}
{{$svgHeight := 280.0}}
{{$svgHeight := add $svgHeight $maxFieldsHeight}}
{{$svgHeight := add $svgHeight $heightForBreakdown}}
{{$svgHeight := add $svgHeight $heightForCode}}
{{$svgHeight := add $svgHeight 80.0}}

{{$svgWidth := 1100.0}}
{{$colWidth := 510.0}}
{{$leftX := 24.0}}
{{$rightX := 560.0}}
{{$barWidth := 260.0}}
{{$rowHeight := 34.0}}

<svg width="{{$svgWidth}}" height="{{$svgHeight}}" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
	<style>
		.field-text { font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace; font-size: 12px; fill: #374151; }
		.field-label { font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace; font-size: 11.5px; fill: #1f2937; font-weight: 600; text-anchor: end; }
		.struct-name { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; font-size: 18px; font-weight: 700; fill: #111827; }
		.section-label { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; font-size: 12px; font-weight: 600; fill: #6b7280; text-transform: uppercase; letter-spacing: 0.05em; }
		.offset-text { font-family: 'SF Mono', Consolas, monospace; font-size: 11px; fill: #6b7280; }
		.size-text { font-family: 'SF Mono', Consolas, monospace; font-size: 11px; fill: #6b7280; }
		.stat-value { font-family: 'SF Mono', Consolas, monospace; font-size: 12px; fill: #374151; }
		.stat-label { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; font-size: 10px; fill: #9ca3af; text-transform: uppercase; }
		.code-text { font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', Consolas, monospace; font-size: 12px; fill: #d1d5db; }
		.breakdown-text { font-family: 'SF Mono', Consolas, monospace; font-size: 11px; fill: #6b7280; }
		.col-header { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; font-size: 11px; fill: #9ca3af; }
	</style>

	<!-- Background -->
	<rect width="100%" height="100%" fill="#fafafa" rx="8"/>
	<rect x="0" y="0" width="100%" height="100%" fill="none" stroke="#e5e7eb" stroke-width="1" rx="8"/>

	<!-- Header -->
	<text x="24" y="34" class="struct-name">{{.Name}}</text>

	<!-- Stats bar -->
	<rect x="24" y="46" width="1052" height="36" rx="6" fill="#f3f4f6" stroke="#e5e7eb" stroke-width="0.5"/>
	<text x="40" y="62" class="stat-label">SIZE</text>
	<text x="40" y="76" class="stat-value">{{.TotalSize}}B</text>
	<text x="130" y="62" class="stat-label">WASTED</text>
	<text x="130" y="76" class="stat-value">{{.WastedBytes}}B ({{printf "%.1f" .WastedPercent}}%)</text>
	<text x="310" y="62" class="stat-label">OPTIMIZED</text>
	<text x="310" y="76" class="stat-value">{{.OptimizedSize}}B</text>
	{{if lt 0 .SavedBytes}}<text x="440" y="62" class="stat-label">SAVED</text>
	<text x="440" y="76" class="stat-value" fill="#059669">{{.SavedBytes}}B ({{printf "%.1f" .SavedPercent}}%)</text>{{end}}

	<!-- ==================== SIDE BY SIDE LAYOUTS ==================== -->

	<!-- LEFT: Original Layout -->
	<text x="{{$leftX}}" y="108" class="section-label">Original Layout</text>
	<text x="{{add $leftX 286.0}}" y="108" class="stat-value" font-size="10" fill="#9ca3af">{{.TotalSize}} bytes</text>

	<!-- Left column headers -->
	<text x="{{add $leftX 42.0}}" y="126" class="col-header">Off</text>
	<text x="{{add $leftX 120.0}}" y="126" class="col-header">Field</text>
	<text x="{{add $leftX 438.0}}" y="126" class="col-header">Size</text>

	{{$leftYOffset := 136.0}}
	{{$leftBarX := add $leftX 135.0}}

	{{range .Fields}}
	{{$yPos := add $leftYOffset (div $rowHeight 2.0)}}
	{{$yPos := add $yPos 3.0}}
	<text x="{{add $leftX 42.0}}" y="{{$yPos}}" class="offset-text" text-anchor="end">{{.Offset}}</text>
	<text x="{{add $leftX 120.0}}" y="{{$yPos}}" class="field-label">{{.Name}}</text>
	<rect x="{{$leftBarX}}" y="{{add $leftYOffset 2.0}}" width="{{$barWidth}}" height="{{sub $rowHeight 4.0}}" rx="3" fill="{{.Color}}" stroke="{{if .IsPadding}}#d1d5db{{else}}#374151{{end}}" stroke-width="{{if .IsPadding}}1{{else}}0.5{{end}}" stroke-opacity="0.5" {{if .IsPadding}}stroke-dasharray="4,3"{{end}} fill-opacity="{{if .IsPadding}}0.45{{else}}0.85{{end}}"/>
	<text x="{{add $leftBarX 272.0}}" y="{{$yPos}}" class="size-text">{{.Size}}B</text>
	{{$leftYOffset = add $leftYOffset $rowHeight}}
	{{end}}
	<text x="{{add $leftX 42.0}}" y="{{add $leftYOffset 3.0}}" class="offset-text" text-anchor="end">{{.TotalSize}}</text>

	<!-- Vertical divider -->
	<line x1="{{sub $rightX 12.0}}" y1="100" x2="{{sub $rightX 12.0}}" y2="{{add $leftYOffset 10.0}}" stroke="#e5e7eb" stroke-width="1"/>

	<!-- RIGHT: Optimized Layout -->
	<text x="{{$rightX}}" y="108" class="section-label">Optimized Layout</text>
	<text x="{{add $rightX 286.0}}" y="108" class="stat-value" font-size="10" fill="#059669">{{.OptimizedSize}} bytes ({{printf "%.1f" .OptimizedWastePercent}}% waste)</text>

	<!-- Right column headers -->
	<text x="{{add $rightX 42.0}}" y="126" class="col-header">Off</text>
	<text x="{{add $rightX 120.0}}" y="126" class="col-header">Field</text>
	<text x="{{add $rightX 438.0}}" y="126" class="col-header">Size</text>

	{{$rightYOffset := 136.0}}
	{{$rightBarX := add $rightX 135.0}}

	{{range .OptimizedFields}}
	{{$yPos := add $rightYOffset (div $rowHeight 2.0)}}
	{{$yPos := add $yPos 3.0}}
	<text x="{{add $rightX 42.0}}" y="{{$yPos}}" class="offset-text" text-anchor="end">{{.Offset}}</text>
	<text x="{{add $rightX 120.0}}" y="{{$yPos}}" class="field-label">{{.Name}}</text>
	<rect x="{{$rightBarX}}" y="{{add $rightYOffset 2.0}}" width="{{$barWidth}}" height="{{sub $rowHeight 4.0}}" rx="3" fill="{{.Color}}" stroke="{{if .IsPadding}}#d1d5db{{else}}#374151{{end}}" stroke-width="{{if .IsPadding}}1{{else}}0.5{{end}}" stroke-opacity="0.5" {{if .IsPadding}}stroke-dasharray="4,3"{{end}} fill-opacity="{{if .IsPadding}}0.45{{else}}0.85{{end}}"/>
	<text x="{{add $rightBarX 272.0}}" y="{{$yPos}}" class="size-text">{{.Size}}B</text>
	{{$rightYOffset = add $rightYOffset $rowHeight}}
	{{end}}
	<text x="{{add $rightX 42.0}}" y="{{add $rightYOffset 3.0}}" class="offset-text" text-anchor="end">{{.OptimizedSize}}</text>

	<!-- ==================== BELOW BOTH COLUMNS ==================== -->

	<!-- Use the taller column's Y offset -->
	{{$bottomY := $leftYOffset}}
	{{if gt $rightYOffset $leftYOffset}}{{$bottomY = $rightYOffset}}{{end}}

	<!-- Horizontal divider -->
	{{$dividerY := add $bottomY 20.0}}
	<line x1="24" y1="{{$dividerY}}" x2="1076" y2="{{$dividerY}}" stroke="#e5e7eb" stroke-width="1"/>

	<!-- Field details -->
	{{$detailsY := add $dividerY 22.0}}
	<text x="24" y="{{$detailsY}}" class="section-label">Field Details</text>
	{{$detailStartY := add $detailsY 18.0}}
	{{range $i, $f := .FieldBreakdown}}
	{{$itemY := add $detailStartY (mul (float64 $i) 18.0)}}
	<text x="32" y="{{$itemY}}" class="breakdown-text" fill="{{if $f.IsPadding}}#ef4444{{else}}#6b7280{{end}}">{{$f.Text}}</text>
	{{end}}

	<!-- Suggested code -->
	{{$codeY := add $detailStartY 14.0}}
	{{$codeY := add $codeY (mul (float64 (len .FieldBreakdown)) 18.0)}}
	<text x="24" y="{{$codeY}}" class="section-label">Suggested Code</text>
	{{$codeBoxY := add $codeY 10.0}}
	{{$codeBoxHeight := add 40.0 (mul (float64 (len .OptimizedFieldsCode)) 18.0)}}
	<rect x="24" y="{{$codeBoxY}}" width="1052" height="{{$codeBoxHeight}}" rx="6" fill="#1f2937"/>
	{{$codeHeaderY := add $codeBoxY 22.0}}
	<text x="40" y="{{$codeHeaderY}}" class="code-text"><tspan fill="#c084fc">type</tspan> <tspan fill="#67e8f9">{{.Name}}Optimized</tspan> <tspan fill="#c084fc">struct</tspan> <tspan fill="#d1d5db">{</tspan></text>
	{{range $i, $f := .OptimizedFieldsCode}}
	{{$lineY := add $codeHeaderY 18.0}}
	{{$lineY := add $lineY (mul (float64 $i) 18.0)}}
	<text x="56" y="{{$lineY}}" class="code-text">{{$f}}</text>
	{{end}}
	{{$endY := add $codeHeaderY 18.0}}
	{{$endY := add $endY (mul (float64 (len .OptimizedFieldsCode)) 18.0)}}
	<text x="40" y="{{$endY}}" class="code-text">}</text>
</svg>
{{end}}`
)
