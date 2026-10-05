package application

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

const (
	pngFormatLabel   = "PNG image"
	jpegFormatLabel  = "JPEG image"
	webpFormatLabel  = "WebP image"
	imageFormatLabel = "Image"
)

func BuildImageDataURL(contentType string, content []byte) string {
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(content)
}

func BuildImagePreview(contentType string, content []byte) string {
	label := imageLabel(contentType)
	configuration, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || configuration.Width <= 0 || configuration.Height <= 0 {
		return label
	}
	return fmt.Sprintf("%s, %dx%d", label, configuration.Width, configuration.Height)
}

func imageLabel(contentType string) string {
	switch contentType {
	case "image/png":
		return pngFormatLabel
	case "image/jpeg":
		return jpegFormatLabel
	case "image/webp":
		return webpFormatLabel
	default:
		return imageFormatLabel
	}
}
