package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/disintegration/imaging"
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

	// Font settings
	monoFontPath     = "public/fonts/Go-Mono.ttf"
	interBoldPath    = "public/fonts/Inter-Bold.ttf"
	interRegularPath = "public/fonts/Inter-Regular.ttf"
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
		FontPath: monoFontPath,
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
		FontPath:     monoFontPath,
		Watermark:    DefaultWatermarkConfig(),
	}
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

func GenerateCodeImage(code, filename string) ([]byte, error) {
	config := DefaultImageConfig()
	config.FontSize = 32 // Even larger text for better visibility

	img, err := GenerateCodeImageWithConfig(code, filename, config)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	return buf.Bytes(), nil
}

func GenerateCodeImageWithConfig(code, filename string, config ImageConfig) (image.Image, error) {
	// Setup canvas with dark background
	bgColor := color.RGBA{46, 52, 64, 255} // #2E3440
	dc := gg.NewContext(config.Width, config.Height)
	dc.SetColor(bgColor)
	dc.Clear()

	// Load fonts
	monoFontPath, err := filepath.Abs(monoFontPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get mono font path: %w", err)
	}

	interBoldPath, err := filepath.Abs(interBoldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get Inter Bold font path: %w", err)
	}

	// Calculate positions
	gradientStop := float64(config.Width) * 0.2
	codeStartX := gradientStop + 40 // 20px after gradient stop
	codeStartY := 120.0

	// Setup syntax highlighting
	iterator, style, err := setupSyntaxHighlighting(code)
	if err != nil {
		return nil, err
	}

	// Load font for code
	if err := dc.LoadFontFace(monoFontPath, config.FontSize); err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Draw code
	x := codeStartX
	y := codeStartY
	lineHeight := config.FontSize * config.LineSpacing

	for _, token := range iterator.Tokens() {
		if token.Value == "" {
			continue
		}

		color := getTokenColor(token, style)
		dc.SetColor(color)

		lines := strings.Split(token.Value, "\n")
		for i, line := range lines {
			if i > 0 {
				x = codeStartX
				y += lineHeight
			}

			// Skip if we've reached the bottom of the image
			if y > float64(config.Height-40) {
				break
			}

			dc.DrawString(line, x, y)
			width, _ := dc.MeasureString(line)
			x += width
		}
	}

	// Create gradient overlay using a smoother technique
	// First stop: solid white rectangle up to gradientStop
	dc.SetRGBA(0.92, 0.92, 0.96, 0.90) // Almost completely opaque white
	dc.DrawRectangle(0, 0, gradientStop, float64(config.Height))
	dc.Fill()

	// Second stop: smooth gradient from white to transparent
	gradientWidth := float64(config.Width) - gradientStop
	for x := 0.0; x < gradientWidth; x++ {
		// Use a smooth easing function instead of linear
		progress := x / gradientWidth
		// Apply cubic easing: progress = 1 - (1-t)^3
		easedProgress := 1 - math.Pow(1-progress, 3)
		alpha := (1.0 - easedProgress) * 0.90

		dc.SetRGBA(0.92, 0.92, 0.96, alpha)
		xPos := gradientStop + x
		dc.DrawRectangle(xPos, 0, 1, float64(config.Height))
		dc.Fill()
	}

	// Draw filename on top of everything with background color using Inter Bold
	if err := dc.LoadFontFace(interBoldPath, 42); err != nil {
		return nil, fmt.Errorf("failed to load Inter Bold font: %w", err)
	}
	dc.SetColor(bgColor)
	dc.DrawString(filename, 40, 60)

	// Add watermark if enabled
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

func GenerateBinaryPreviewImage(filename, mimeType string) (image.Image, error) {
	// Create new context with OG dimensions
	bgColor := color.RGBA{46, 52, 64, 255} // #2E3440
	dc := gg.NewContext(ogImageWidth, ogImageHeight)
	dc.SetColor(bgColor)
	dc.Clear()

	// Load font
	interBoldPath, err := filepath.Abs(interBoldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get font path: %w", err)
	}

	// Calculate gradient stop
	gradientStop := float64(ogImageWidth) * 0.2

	// Create gradient overlay using a smoother technique
	overlayR, overlayG, overlayB := 235.0/255.0, 235.0/255.0, 245.0/255.0 // Light color for overlay and shadows
	overlayAlpha := 0.90
	// First stop: solid white rectangle up to gradientStop
	dc.SetRGBA(overlayR, overlayG, overlayB, overlayAlpha)
	dc.DrawRectangle(0, 0, gradientStop, float64(ogImageHeight))
	dc.Fill()

	// Second stop: smooth gradient from white to transparent
	gradientWidth := float64(ogImageWidth) - gradientStop
	for x := 0.0; x < gradientWidth; x++ {
		// Use a smooth easing function instead of linear
		progress := x / gradientWidth
		// Apply cubic easing: progress = 1 - (1-t)^3
		easedProgress := 1 - math.Pow(1-progress, 3)
		alpha := (1.0 - easedProgress) * overlayAlpha

		dc.SetRGBA(overlayR, overlayG, overlayB, alpha)
		xPos := gradientStop + x
		dc.DrawRectangle(xPos, 0, 1, float64(ogImageHeight))
		dc.Fill()
	}

	// Draw centered title with soft shadow
	if err := dc.LoadFontFace(interBoldPath, 48); err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	titleY := float64(ogImageHeight)/2 - 40
	// Draw shadow first - using overlay color for a softer effect
	shadowOffset := 1.0
	shadowBlur := 10.0
	shadowSteps := 16.0

	// Draw shadow in a circular pattern
	for i := 0.0; i < shadowSteps; i++ {
		angle := (i / shadowSteps) * 2 * math.Pi
		for r := 0.0; r < shadowBlur; r++ {
			progress := r / shadowBlur
			alpha := 0.03 * (1 - progress*progress)

			offsetX := math.Cos(angle) * r * 0.5
			offsetY := math.Sin(angle) * r * 0.5

			dc.SetRGBA(float64(overlayR), float64(overlayG), float64(overlayB), alpha)
			dc.DrawStringAnchored(filename,
				float64(ogImageWidth)/2+offsetX+shadowOffset,
				titleY+offsetY+shadowOffset,
				0.5, 0.5)
		}
	}

	// Draw main text
	dc.SetColor(bgColor)
	dc.DrawStringAnchored(filename, float64(ogImageWidth)/2, titleY, 0.5, 0.5)

	// Draw centered mime type with soft shadow
	if err := dc.LoadFontFace(interBoldPath, 36); err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	mimeTypeY := titleY + 80
	// Draw shadow first - using overlay color for a softer effect
	for i := 0.0; i < shadowSteps; i++ {
		angle := (i / shadowSteps) * 2 * math.Pi
		for r := 0.0; r < shadowBlur; r++ {
			progress := r / shadowBlur
			alpha := 0.03 * (1 - progress*progress)

			offsetX := math.Cos(angle) * r * 0.5
			offsetY := math.Sin(angle) * r * 0.5

			dc.SetRGBA(float64(overlayR), float64(overlayG), float64(overlayB), alpha)
			dc.DrawStringAnchored(mimeType,
				float64(ogImageWidth)/2+offsetX+shadowOffset,
				mimeTypeY+offsetY+shadowOffset,
				0.5, 0.5)
		}
	}

	// Draw main text
	dc.SetColor(bgColor)
	dc.DrawStringAnchored(mimeType, float64(ogImageWidth)/2, mimeTypeY, 0.5, 0.5)

	// Add watermark
	watermark := DefaultWatermarkConfig()
	if err := drawWatermark(dc, watermark); err != nil {
		return nil, fmt.Errorf("failed to draw watermark: %w", err)
	}

	return dc.Image(), nil
}

func GenerateImagePreview(img image.Image) (image.Image, error) {
	// Create new context with OG dimensions
	bgColor := color.RGBA{46, 52, 64, 255} // #2E3440
	dc := gg.NewContext(ogImageWidth, ogImageHeight)
	dc.SetColor(bgColor)
	dc.Clear()

	// Resize to OG image dimensions while maintaining aspect ratio
	resized := imaging.Fit(img, ogImageWidth, ogImageHeight, imaging.Lanczos)

	// Calculate position to center the image
	x := (ogImageWidth - resized.Bounds().Dx()) / 2
	y := (ogImageHeight - resized.Bounds().Dy()) / 2

	// Draw the resized image centered
	dc.DrawImage(resized, x, y)

	// Add watermark
	watermark := DefaultWatermarkConfig()
	if err := drawWatermark(dc, watermark); err != nil {
		return nil, fmt.Errorf("failed to draw watermark: %w", err)
	}

	return dc.Image(), nil
}
