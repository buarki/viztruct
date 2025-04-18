package template

// returning it as a string because functions like os.Getwd and os.Stat
// are unsupported in the WebAssembly runtime environment (e.g., browsers)
var (
	StructLayoutTemplate = `{{define "struct_layout"}}
<svg width="1200" height="1270" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
		<style>
			.field-text { font-family: Arial, sans-serif; font-size: 14px; fill: #000000; }
			.struct-name { font-family: Arial, sans-serif; font-size: 16px; font-weight: bold; fill: #000000; }
			.offset-text { font-family: Arial, sans-serif; font-size: 12px; fill: #000000; }
			.size-text { font-family: Arial, sans-serif; font-size: 12px; fill: #000000; }
			.padding-pattern { fill: #CCCCCC; fill-opacity: 0.3; }
		</style>
		<rect width="100%" height="100%" fill="white"/>
	<text x="10" y="50" class="struct-name" fill="#000000">{{.Name}}</text>
<text x="10" y="70" class="field-text" fill="#000000">Total size: {{.TotalSize}} bytes | Wasted: {{.WastedBytes}} bytes ({{.WastedPercent}}%)</text>
<text x="10" y="90" class="field-text" fill="#000000">Original layout:</text>

{{$yOffset := 150.0}}
{{range .Fields}}
<text x="{{add .LabelX 7.3}}" y="{{sub $yOffset 20.0}}" class="field-text" text-anchor="end" transform="rotate(90 {{add .LabelX 7.3}} {{sub $yOffset 20.0}})" fill="#000000">{{.Name}}</text>
<rect x="{{.X}}" y="{{$yOffset}}" width="{{.Width}}" height="{{.BlockHeight}}" fill="{{.Color}}" stroke="{{if .IsPadding}}gray{{else}}black{{end}}" stroke-width="1" {{if .IsPadding}}stroke-dasharray="5,5"{{end}}/>
<text x="{{.X}}" y="{{add $yOffset 55.0}}" class="offset-text" text-anchor="middle" fill="#000000">{{.Offset}}</text>
{{end}}
<text x="{{.LastOffsetX}}" y="{{add $yOffset 55.0}}" class="offset-text" text-anchor="middle" fill="#000000">{{.TotalSize}}</text>

<text x="10" y="{{add $yOffset 100.0}}" class="field-text" fill="#000000">Field breakdown:</text>
{{range $i, $f := .FieldBreakdown}}
<text x="10" y="{{add $yOffset (add 120.0 (mul (float64 $i) 15.0))}}" class="field-text" fill="{{if $f.IsPadding}}#FF0000{{else}}#000000{{end}}">{{$f.Text}}</text>
{{end}}

{{$optimizedYOffset := add $yOffset 250.0}}
<text x="10" y="{{add $optimizedYOffset 50.0}}" class="field-text" fill="#000000">Optimized layout: {{.OptimizedSize}} bytes (saved {{.SavedBytes}} bytes, {{.OptimizedWastePercent}}% waste)</text>

{{$blockYOffset := add $optimizedYOffset 100.0}}
{{range .OptimizedFields}}
<text x="{{add .LabelX 7.3}}" y="{{sub $blockYOffset 20.0}}" class="field-text" text-anchor="end" transform="rotate(90 {{add .LabelX 7.3}} {{sub $blockYOffset 20.0}})" fill="#000000">{{.Name}}</text>
<rect x="{{.X}}" y="{{$blockYOffset}}" width="{{.Width}}" height="{{.BlockHeight}}" fill="{{.Color}}" stroke="{{if .IsPadding}}gray{{else}}black{{end}}" stroke-width="1" {{if .IsPadding}}stroke-dasharray="5,5"{{end}}/>
<text x="{{.X}}" y="{{add $blockYOffset 55.0}}" class="offset-text" text-anchor="middle" fill="#000000">{{.Offset}}</text>
{{end}}

{{if lt .OptimizedSize .TotalSize}}
<rect x="{{.OptimizedLastX}}" y="{{$blockYOffset}}" width="{{sub .LastOffsetX .OptimizedLastX}}" height="{{.BlockHeight}}" fill="#F5F5F5" stroke="gray" stroke-width="1" stroke-dasharray="5,5"/>
<text x="{{.OptimizedLastX}}" y="{{add $blockYOffset 55.0}}" class="offset-text" text-anchor="middle" fill="#000000">{{.OptimizedSize}}</text>
<text x="{{.LastOffsetX}}" y="{{add $blockYOffset 55.0}}" class="offset-text" text-anchor="middle" fill="#000000">{{.TotalSize}}</text>
{{end}}

<text x="10" y="{{add $blockYOffset 100.0}}" class="field-text" fill="#000000">Suggested code:</text>
<text x="10" y="{{add $blockYOffset 120.0}}" class="field-text" fill="#000000">type {{.Name}}Optimized struct {</text>
{{range $i, $f := .OptimizedFieldsCode}}
<text x="10" y="{{add (add $blockYOffset 135.0) (mul (float64 $i) 15.0)}}" class="field-text" fill="#000000">    {{$f}}</text>
{{end}}
<text x="10" y="{{add $blockYOffset (add 135.0 (mul (float64 (len .OptimizedFieldsCode)) 15.0))}}" class="field-text" fill="#000000">}</text>
</svg>
{{end}}`
)
