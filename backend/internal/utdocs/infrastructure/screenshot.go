package infrastructure

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"lunar/backend/internal/utdocs/domain"
)

const (
	maxScreenshotLines    = 200
	maxScreenshotColumns  = 110
	screenshotFontSize    = 13
	screenshotLineHeight  = 18
	screenshotPadding     = 16
	screenshotTabWidth    = 4
	screenshotBorderWidth = 1
)

var (
	screenshotBackground = color.RGBA{R: 23, G: 25, B: 30, A: 255}
	screenshotTitleBar   = color.RGBA{R: 45, G: 48, B: 56, A: 255}
	screenshotBorder     = color.RGBA{R: 70, G: 74, B: 84, A: 255}
	screenshotTitleText  = color.RGBA{R: 240, G: 242, B: 248, A: 255}
	screenshotBodyText   = color.RGBA{R: 212, G: 216, B: 224, A: 255}
)

type ScreenshotRenderer struct{}

func NewScreenshotRenderer() *ScreenshotRenderer {
	return &ScreenshotRenderer{}
}

func (r *ScreenshotRenderer) Render(request domain.ScreenshotRequest) ([]byte, error) {
	face, err := newScreenshotFace()
	if err != nil {
		return nil, err
	}
	defer face.Close()

	cellWidth, err := screenshotCellWidth(face)
	if err != nil {
		return nil, err
	}

	titleLines := []string{
		TruncateScreenshotLine(
			fmt.Sprintf("repo %s   branch %s   result %s", request.Repo, request.Branch, screenshotResultLabel(request)),
			maxScreenshotColumns,
		),
		TruncateScreenshotLine("command "+request.Command, maxScreenshotColumns),
	}
	bodyLines, _ := PrepareScreenshotLines(request.Output, maxScreenshotLines)

	width := screenshotPadding*2 + maxScreenshotColumns*cellWidth
	titleHeight := screenshotPadding*2 + len(titleLines)*screenshotLineHeight
	bodyHeight := screenshotPadding*2 + len(bodyLines)*screenshotLineHeight
	height := titleHeight + screenshotBorderWidth + bodyHeight

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(screenshotBackground), image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(0, 0, width, titleHeight), image.NewUniform(screenshotTitleBar), image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(0, titleHeight, width, titleHeight+screenshotBorderWidth), image.NewUniform(screenshotBorder), image.Point{}, draw.Src)

	ascent := face.Metrics().Ascent.Ceil()

	titleDrawer := &font.Drawer{Dst: canvas, Src: image.NewUniform(screenshotTitleText), Face: face}
	for index, line := range titleLines {
		titleDrawer.Dot = fixed.P(screenshotPadding, screenshotPadding+ascent+index*screenshotLineHeight)
		titleDrawer.DrawString(line)
	}

	bodyDrawer := &font.Drawer{Dst: canvas, Src: image.NewUniform(screenshotBodyText), Face: face}
	bodyTop := titleHeight + screenshotBorderWidth + screenshotPadding
	for index, line := range bodyLines {
		bodyDrawer.Dot = fixed.P(screenshotPadding, bodyTop+ascent+index*screenshotLineHeight)
		bodyDrawer.DrawString(TruncateScreenshotLine(ExpandScreenshotTabs(line, screenshotTabWidth), maxScreenshotColumns))
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, canvas); err != nil {
		return nil, fmt.Errorf("encode the screenshot: %w", err)
	}
	return buffer.Bytes(), nil
}

func screenshotResultLabel(request domain.ScreenshotRequest) string {
	if request.Total <= 0 {
		return request.Result
	}
	return fmt.Sprintf("%s (%d/%d passed)", request.Result, request.Passed, request.Total)
}

func newScreenshotFace() (font.Face, error) {
	parsed, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse the monospace font: %w", err)
	}

	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    screenshotFontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("prepare the monospace font: %w", err)
	}
	return face, nil
}

func screenshotCellWidth(face font.Face) (int, error) {
	advance, hasAdvance := face.GlyphAdvance('M')
	if !hasAdvance {
		return 0, errors.New("the monospace font has no advance for the sample glyph")
	}
	if advance <= 0 {
		return 0, errors.New("the monospace font advance is not positive")
	}
	return advance.Ceil(), nil
}

func PrepareScreenshotLines(output string, maxLines int) ([]string, int) {
	normalized := strings.ReplaceAll(output, "\r\n", "\n")
	rawLines := strings.Split(normalized, "\n")
	if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
		rawLines = rawLines[:len(rawLines)-1]
	}
	if len(rawLines) == 0 {
		rawLines = []string{""}
	}
	if maxLines <= 0 {
		return []string{}, len(rawLines)
	}
	if len(rawLines) <= maxLines {
		return rawLines, 0
	}

	dropped := len(rawLines) - (maxLines - 1)
	lines := make([]string, 0, maxLines)
	lines = append(lines, rawLines[:maxLines-1]...)
	lines = append(lines, TruncationMarker(dropped))
	return lines, dropped
}

func TruncationMarker(dropped int) string {
	return fmt.Sprintf("... %d lines were left out ...", dropped)
}

func ExpandScreenshotTabs(line string, tabWidth int) string {
	if tabWidth <= 0 || !strings.Contains(line, "\t") {
		return line
	}

	builder := strings.Builder{}
	column := 0
	for _, character := range line {
		if character == '\t' {
			spaces := tabWidth - (column % tabWidth)
			builder.WriteString(strings.Repeat(" ", spaces))
			column += spaces
			continue
		}
		builder.WriteRune(character)
		column++
	}
	return builder.String()
}

func TruncateScreenshotLine(line string, maxColumns int) string {
	if maxColumns <= 0 {
		return ""
	}
	runes := []rune(line)
	if len(runes) <= maxColumns {
		return line
	}
	if maxColumns <= 3 {
		return string(runes[:maxColumns])
	}
	return string(runes[:maxColumns-3]) + "..."
}
