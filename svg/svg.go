package svg

import (
	"fmt"
	"strings"

	"github.com/buarki/viztruct/structi"
)

const (
	blockHeight   = 40
	paddingX      = 10
	paddingY      = 10
	structSpacing = 30
	textOffsetY   = 20
	// TODO make these ones configurable
	compareMode  = true
	optimizeMode = true
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

func getTypeColor(typeName string) string {
	if color, ok := typeColors[typeName]; ok {
		return color
	}
	return "#AAAAAA" // default gray for unknown types
}

func BuildVisualization(structs []structi.Info) string {
	// compute total SVG dimensions
	var totalWidth int = 1200
	var totalHeight int = 0
	var sb strings.Builder

	// compute height needed for all structs
	for _, s := range structs {
		baseHeight := 60

		originalLayoutHeight := len(s.Fields)*blockHeight + 80 // blocks + padding + title + offset markers

		fieldBreakdownHeight := len(s.Fields)*20 + 40 // text lines + title + padding

		optimizedLayoutHeight := 0
		if optimizeMode {
			optimizedLayoutHeight = len(s.OptimizedFields)*blockHeight + 80 // blocks + padding + title + offset markers
			// add height for suggested code
			optimizedLayoutHeight += len(s.OptimizedFields)*20 + 60 // text lines + title + padding
		}

		structHeight := baseHeight + originalLayoutHeight + fieldBreakdownHeight + optimizedLayoutHeight + structSpacing
		totalHeight += structHeight
	}

	// adding some extra padding at the bottom
	totalHeight += paddingY * 2

	sb.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
		<style>
			.field-text { font-family: Arial, sans-serif; font-size: 14px; fill: #000000; }
			.struct-name { font-family: Arial, sans-serif; font-size: 16px; font-weight: bold; fill: #000000; }
			.offset-text { font-family: Arial, sans-serif; font-size: 12px; fill: #000000; }
			.size-text { font-family: Arial, sans-serif; font-size: 12px; fill: #000000; }
			.padding-pattern { fill: #CCCCCC; fill-opacity: 0.3; }
		</style>
		<rect width="100%%" height="100%%" fill="white"/>
	`, totalWidth, totalHeight))

	// draw each struct
	y := paddingY
	for _, structInfo := range structs {
		if compareMode && optimizeMode {
			// draw original and optimized layouts side by side
			structWidth := (totalWidth - 3*paddingX) / 2
			y = drawStructVisualization(&sb, structInfo, paddingX, y, structWidth)
		} else {
			// draw single layout
			y = drawStructVisualization(&sb, structInfo, paddingX, y, totalWidth-2*paddingX)
		}
		y += structSpacing
	}

	sb.WriteString("</svg>")
	return sb.String()
}

func drawStructVisualization(sb *strings.Builder, structInfo structi.Info, x, y int, width int) int {
	wastedBytes, wastedPercent := structInfo.WastedSpace()

	structTotalSize := structInfo.TotalSize()

	// draw struct name and stats with explicit fill color
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="struct-name" fill="#000000">%s</text>
<text x="%d" y="%d" class="size-text" fill="#000000">Total size: %d bytes | Wasted: %d bytes (%.1f%%)</text>
`, x, y, structInfo.Name, x, y+20, structTotalSize, wastedBytes, wastedPercent))

	y += 40

	// Calculate scale factor based on the max offset
	maxOffset := structTotalSize
	if maxOffset == 0 {
		maxOffset = 1 // Avoid division by zero
	}
	scale := float64(width) / float64(maxOffset)

	// draw original struct layout with explicit fill color
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="#000000">Original layout:</text>
`, x, y))
	y += 20

	// add extra space for field names above blocks
	y += 40 // Space for rotated text

	// draw blocks for original layout
	blockY := y
	for _, field := range structInfo.Fields {
		blockX := float64(x) + float64(field.Offset)*scale
		blockWidth := float64(field.Size) * scale

		color := "#CCCCCC"
		if !field.IsPadding {
			color = getTypeColor(field.TypeName)
		}

		// add field name above the block for non-padding fields
		if !field.IsPadding {
			// Calculate position for rotated text
			textX := blockX + blockWidth/2
			textY := float64(blockY - 10) // Position above the block

			// add rotated text with transform (90 degrees for upward text)
			sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" class="field-text" text-anchor="end" transform="rotate(90 %.1f %.1f)" fill="#000000">%s</text>
`, textX, textY, textX, textY, field.Name))
		}

		// draw the block
		if field.IsPadding {
			sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%d" width="%.1f" height="%d" fill="%s" stroke="gray" stroke-width="1" stroke-dasharray="5,5"/>
`, blockX, blockY, blockWidth, blockHeight, color))
		} else {
			sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%d" width="%.1f" height="%d" fill="%s" stroke="black" stroke-width="1"/>
`, blockX, blockY, blockWidth, blockHeight, color))
		}

		// add offset markers with explicit fill color
		sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%d" class="offset-text" text-anchor="middle" fill="#000000">%d</text>
`, blockX, blockY+blockHeight+15, field.Offset))
	}

	// add final offset marker with explicit fill color
	sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%d" class="offset-text" text-anchor="middle" fill="#000000">%d</text>
`, float64(x)+float64(maxOffset)*scale, blockY+blockHeight+15, maxOffset))

	// move y past the blocks and offset markers
	y = blockY + blockHeight + 30

	// draw field breakdown with explicit fill color
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="#000000">Field breakdown:</text>
`, x, y))
	y += 20

	for _, field := range structInfo.Fields {
		text := fmt.Sprintf("%s: Offset=%d, Size=%d", field.Name, field.Offset, field.Size)
		if !field.IsPadding {
			text += fmt.Sprintf(", Type=%s, Align=%d", field.TypeName, field.Align)
		}

		color := "#000000"
		if field.IsPadding {
			color = "#FF0000" // highlight padding in red
		}

		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="%s">%s</text>
`, x, y, color, text))
		y += 15
	}

	if optimizeMode && len(structInfo.OptimizedFields) > 0 {
		_, optimizedWastedPercent := structInfo.OptimazedWastedSpace()
		optimizedSize := structInfo.OptimazedTotalSize()

		y += 10
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="#000000">Optimized layout: %d bytes (saved %d bytes, %.1f%% waste)</text>
`, x, y, optimizedSize, maxOffset-optimizedSize, optimizedWastedPercent))
		y += 20

		// Recalculate scale for optimized layout
		scale = float64(width) / float64(optimizedSize)
		if optimizedSize == 0 {
			scale = float64(width) // Avoid division by zero
		}

		// draw blocks for optimized layout
		y += 40 // Space for rotated text in optimized layout
		blockY = y

		for _, field := range structInfo.OptimizedFields {
			blockX := float64(x) + float64(field.Offset)*scale
			blockWidth := float64(field.Size) * scale

			color := "#CCCCCC"
			if !field.IsPadding {
				color = getTypeColor(field.TypeName)
			}

			// add field name above the block for non-padding fields
			if !field.IsPadding {
				textX := blockX + blockWidth/2
				textY := float64(blockY - 10)

				sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" class="field-text" text-anchor="end" transform="rotate(90 %.1f %.1f)" fill="#000000">%s</text>
`, textX, textY, textX, textY, field.Name))
			}

			// draw the block
			if field.IsPadding {
				sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%d" width="%.1f" height="%d" fill="%s" stroke="gray" stroke-width="1" stroke-dasharray="5,5"/>
`, blockX, blockY, blockWidth, blockHeight, color))
			} else {
				sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%d" width="%.1f" height="%d" fill="%s" stroke="black" stroke-width="1"/>
`, blockX, blockY, blockWidth, blockHeight, color))
			}

			// add offset markers with explicit fill color
			sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%d" class="offset-text" text-anchor="middle" fill="#000000">%d</text>
`, blockX, blockY+blockHeight+15, field.Offset))
		}

		// add final offset marker for optimized layout with explicit fill color
		sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%d" class="offset-text" text-anchor="middle" fill="#000000">%d</text>
`, float64(x)+float64(maxOffset)*scale, blockY+blockHeight+15, maxOffset))

		y = blockY + blockHeight + 30

		// show code suggestion for optimized layout with explicit fill color
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="#000000">Suggested code:</text>
`, x, y))
		y += 20

		// generate the code suggestion with explicit fill color
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="#000000">type %s struct {</text>
`, x, y, structInfo.Name+"Optimized"))
		y += 15

		for _, field := range structInfo.OptimizedFields {
			if !field.IsPadding {
				sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="#000000">    %s %s</text>
`, x, y, field.Name, field.TypeName))
				y += 15
			}
		}

		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="field-text" fill="#000000">}</text>
`, x, y))
		y += 15
	}

	return y
}
