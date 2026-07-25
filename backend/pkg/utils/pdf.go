package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/go-pdf/fpdf"
)

// PDF configuration constants
const (
	pageWidthMM  = 210.0
	pageHeightMM = 297.0
	pageMarginMM = 20.0
)

// GeneratePDF generates a single-page PDF from a single image (backward-compatible wrapper).
func GeneratePDF(img []byte) ([]byte, error) {
	return GenerateMultiPagePDF([][]byte{img}, 85)
}

// GenerateMultiPagePDF generates a multi-page PDF from multiple images with JPEG compression.
func GenerateMultiPagePDF(images [][]byte, jpegQuality int) ([]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images provided")
	}
	if jpegQuality < 1 || jpegQuality > 100 {
		jpegQuality = 85
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)

	for i, imgData := range images {
		if len(imgData) == 0 {
			continue
		}

		// Decode image to get dimensions and format
		decodedImg, format, err := image.Decode(bytes.NewReader(imgData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode image %d: %v", i, err)
		}

		// Re-encode as JPEG for compression (converts PNG/WebP to JPEG)
		var jpegBuf bytes.Buffer
		switch format {
		case "jpeg":
			jpegBuf.Write(imgData)
		default:
			if err := jpeg.Encode(&jpegBuf, decodedImg, &jpeg.Options{Quality: jpegQuality}); err != nil {
				return nil, fmt.Errorf("failed to encode image %d as JPEG: %v", i, err)
			}
		}

		// Register image from reader (no temp file needed)
		imgName := fmt.Sprintf("img_%d", i)
		info := pdf.RegisterImageReader(imgName, "jpg", &jpegBuf)
		if info == nil {
			return nil, fmt.Errorf("failed to register image %d", i)
		}

		imgW := info.Width()
		imgH := info.Height()

		// Add page for each image
		pdf.AddPage()

		// Calculate scale to fit on page with margins
		scale := 1.0
		if imgW > pageWidthMM-pageMarginMM {
			scale = (pageWidthMM - pageMarginMM) / imgW
		}
		if imgH*scale > pageHeightMM-pageMarginMM {
			scale = (pageHeightMM - pageMarginMM) / imgH
		}

		// Center image on page
		x := (pageWidthMM - imgW*scale) / 2
		y := (pageHeightMM - imgH*scale) / 2

		pdf.Image(imgName, x, y, imgW*scale, imgH*scale, false, "", 0, "")
	}

	// Generate PDF bytes
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %v", err)
	}

	pdfBytes := buf.Bytes()
	if len(pdfBytes) == 0 {
		return nil, fmt.Errorf("generated PDF is empty")
	}

	return pdfBytes, nil
}
