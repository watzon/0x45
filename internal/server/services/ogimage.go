package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/fogleman/gg"
)

const (
	// OpenGraph recommended image dimensions
	ogImageWidth  = 1200
	ogImageHeight = 630

	// Text settings
	maxLines     = 25
	fontSize     = 20
	lineSpacing  = 1.5
	padding      = 3.5
	borderWidth  = 2.0
	borderRadius = 15.0
	tabWidth     = 4 // Number of spaces per tab

	// Line number settings
	lineNumPadding = 10.0
	lineNumWidth   = 50.0
	lineNumColor   = 0x666666

	// Font settings
	fontPath = "public/fonts/Go-Mono.ttf"
)

// WatermarkConfig holds configuration for the image watermark
type WatermarkConfig struct {
	Enabled  bool
	Text     string
	FontSize float64
	Color    color.Color
	PaddingX float64
	PaddingY float64
	FontPath string
}

// DefaultWatermarkConfig returns the default watermark configuration
func DefaultWatermarkConfig() WatermarkConfig {
	return WatermarkConfig{
		Enabled:  true,
		Text:     "0x45",
		FontSize: 36,
		Color:    color.RGBA{128, 128, 128, 80}, // Semi-transparent gray
		PaddingX: 20,
		PaddingY: 20,
		FontPath: fontPath,
	}
}

// ImageConfig holds all configuration for image generation
type ImageConfig struct {
	Width        int
	Height       int
	MaxLines     int
	FontSize     float64
	LineSpacing  float64
	Padding      float64
	BorderWidth  float64
	BorderRadius float64
	FontPath     string
	Watermark    WatermarkConfig
}

// DefaultImageConfig returns the default image configuration
func DefaultImageConfig() ImageConfig {
	return ImageConfig{
		Width:        ogImageWidth,
		Height:       ogImageHeight,
		MaxLines:     maxLines,
		FontSize:     fontSize,
		LineSpacing:  lineSpacing,
		Padding:      padding,
		BorderWidth:  borderWidth,
		BorderRadius: borderRadius,
		FontPath:     fontPath,
		Watermark:    DefaultWatermarkConfig(),
	}
}

// wordWrap wraps text at the specified width
func wordWrap(text string, dc *gg.Context, maxWidth float64) []string {
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	currentLine := words[0]

	for _, word := range words[1:] {
		width, _ := dc.MeasureString(currentLine + " " + word)
		if width <= maxWidth {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

type codeImageContext struct {
	dc           *gg.Context
	style        *chroma.Style
	lineNumWidth float64
	maxWidth     float64
	lineHeight   float64
	currentLine  int
	spaceWidth   float64 // Width of a single space character
}

func setupContext(dc *gg.Context, style *chroma.Style, lineNumWidth, maxWidth, lineHeight float64) *codeImageContext {
	// Measure space width for tab calculations
	spaceWidth, _ := dc.MeasureString(" ")

	return &codeImageContext{
		dc:           dc,
		style:        style,
		lineNumWidth: lineNumWidth,
		maxWidth:     maxWidth,
		lineHeight:   lineHeight,
		currentLine:  1,
		spaceWidth:   spaceWidth,
	}
}

// expandTabs replaces tabs with the appropriate number of spaces
func (ctx *codeImageContext) expandTabs(text string, currentX float64) string {
	var result strings.Builder
	column := int(currentX / ctx.spaceWidth)

	for _, r := range text {
		if r == '\t' {
			spaces := tabWidth - (column % tabWidth)
			result.WriteString(strings.Repeat(" ", spaces))
			column += spaces
		} else {
			result.WriteRune(r)
			if r == '\n' {
				column = 0
			} else {
				column++
			}
		}
	}
	return result.String()
}

func drawToken(ctx *codeImageContext, token chroma.Token, x *float64, y *float64) {
	if token.Value == "" {
		return
	}

	// Handle tabs and other special characters
	expandedText := ctx.expandTabs(token.Value, *x)

	// Set color for the token
	ctx.dc.SetColor(getTokenColor(token, ctx.style))

	// Split into lines and handle each separately
	lines := strings.Split(expandedText, "\n")
	for i, line := range lines {
		if ctx.currentLine > maxLines {
			break
		}

		// Handle line numbers for new lines
		if i > 0 {
			*x = padding + borderWidth + ctx.lineNumWidth + lineNumPadding
			*y += ctx.lineHeight
			ctx.currentLine++
			drawLineNumbers(ctx, *y)
		}

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if we need to wrap
		if *x > padding+borderWidth+ctx.lineNumWidth+lineNumPadding {
			remainingWidth := float64(ogImageWidth) - *x - padding - borderWidth
			width, _ := ctx.dc.MeasureString(line)
			if width > remainingWidth {
				*x = padding + borderWidth + ctx.lineNumWidth + lineNumPadding
				*y += ctx.lineHeight
				ctx.currentLine++
				drawLineNumbers(ctx, *y)
			}
		}

		// Draw the text
		ctx.dc.DrawString(line, *x, *y)
		width, _ := ctx.dc.MeasureString(line)
		*x += width
	}
}

// setupCanvas initializes the drawing context with background and border
func setupCanvas(width, height int) (*gg.Context, error) {
	dc := gg.NewContext(width, height)
	dc.Clear()

	// Create clipping path for rounded corners
	dc.DrawRoundedRectangle(borderWidth/2, borderWidth/2,
		float64(width)-borderWidth,
		float64(height)-borderWidth,
		borderRadius)
	dc.Clip()

	// Set background color (dark theme)
	dc.SetColor(color.RGBA{40, 44, 52, 255})
	dc.DrawRectangle(borderWidth/2, borderWidth/2,
		float64(width)-borderWidth,
		float64(height)-borderWidth)
	dc.Fill()

	// Reset clip and draw border
	dc.ResetClip()
	dc.SetColor(color.RGBA{60, 64, 72, 255})
	dc.SetLineWidth(borderWidth)
	dc.DrawRoundedRectangle(borderWidth/2, borderWidth/2,
		float64(width)-borderWidth,
		float64(height)-borderWidth,
		borderRadius)
	dc.Stroke()

	return dc, nil
}

// setupSyntaxHighlighting prepares the lexer and style for syntax highlighting
func setupSyntaxHighlighting(code string) (chroma.Iterator, *chroma.Style, error) {
	lexer := lexers.Analyse(code)
	if lexer == nil {
		lexer = lexers.Get("text")
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil, nil, err
	}

	return iterator, style, nil
}

// drawLineNumbers draws the line number column and separator
func drawLineNumbers(ctx *codeImageContext, y float64) {
	// Draw line number
	ctx.dc.SetColor(color.RGBA{102, 102, 102, 255})
	lineNumX := padding + borderWidth + ctx.lineNumWidth - 5
	ctx.dc.DrawStringAnchored(fmt.Sprintf("%d", ctx.currentLine), lineNumX, y, 1.0, 0)
}

// getTokenColor extracts the color for a token based on its style
func getTokenColor(token chroma.Token, style *chroma.Style) color.Color {
	entry := style.Get(token.Type)
	if entry.IsZero() {
		entry = style.Get(token.Type.SubCategory())
	}
	if entry.IsZero() {
		entry = style.Get(token.Type.Category())
	}

	if !entry.IsZero() && entry.Colour != 0 {
		hexColor := entry.Colour.String()
		hexColor = strings.TrimPrefix(hexColor, "#")
		var r, g, b uint8
		if len(hexColor) == 6 {
			_, err := fmt.Sscanf(hexColor, "%02x%02x%02x", &r, &g, &b)
			if err != nil {
				return color.White
			}
			return color.RGBA{r, g, b, 255}
		}
	}
	return color.White
}

func GenerateCodeImage(code string) ([]byte, error) {
	img, err := GenerateCodeImageWithConfig(code, "", DefaultImageConfig())
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	return buf.Bytes(), nil
}

func GenerateCodeImageWithConfig(code, language string, config ImageConfig) (image.Image, error) {
	// Setup canvas
	dc, err := setupCanvas(config.Width, config.Height)
	if err != nil {
		return nil, err
	}

	// Setup syntax highlighting
	iterator, style, err := setupSyntaxHighlighting(code)
	if err != nil {
		return nil, err
	}

	// Load font
	fontAbsPath, err := filepath.Abs(config.FontPath)
	if err != nil {
		return nil, err
	}
	if err := dc.LoadFontFace(fontAbsPath, config.FontSize); err != nil {
		return nil, err
	}

	// Calculate dimensions
	textStartX := padding + borderWidth + lineNumWidth + lineNumPadding
	maxTextWidth := float64(config.Width) - textStartX - padding - borderWidth

	// Create context
	ctx := setupContext(dc, style, lineNumWidth, maxTextWidth, config.FontSize*lineSpacing)

	// Draw separator line for line numbers
	dc.SetColor(color.RGBA{60, 64, 72, 255})
	dc.SetLineWidth(1)
	dc.DrawLine(
		padding+borderWidth+lineNumWidth,
		borderWidth,
		padding+borderWidth+lineNumWidth,
		float64(config.Height)-borderWidth,
	)
	dc.Stroke()

	// Draw code
	x := textStartX
	y := padding + config.FontSize + borderWidth

	// Draw initial line number
	drawLineNumbers(ctx, y)

	for _, token := range iterator.Tokens() {
		if ctx.currentLine > config.MaxLines {
			break
		}

		drawToken(ctx, token, &x, &y)
	}

	if ctx.currentLine > config.MaxLines {
		dc.SetColor(color.White)
		dc.DrawString("...", textStartX, y+ctx.lineHeight)
	}

	// Add watermark
	if config.Watermark.Enabled {
		if err := drawWatermark(dc, config.Watermark); err != nil {
			return nil, fmt.Errorf("failed to draw watermark: %w", err)
		}
	}

	return dc.Image(), nil
}

func drawWatermark(dc *gg.Context, config WatermarkConfig) error {
	if err := dc.LoadFontFace(config.FontPath, config.FontSize); err != nil {
		return fmt.Errorf("failed to load watermark font: %w", err)
	}

	dc.SetColor(config.Color)
	textWidth, _ := dc.MeasureString(config.Text)

	// Position in bottom right corner
	x := float64(dc.Width()) - textWidth - config.PaddingX
	y := float64(dc.Height()) - config.PaddingY

	// Draw the text
	dc.DrawString(config.Text, x, y)
	return nil
}
