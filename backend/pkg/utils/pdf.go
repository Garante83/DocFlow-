package utils

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/go-pdf/fpdf"
)

// PDF configuration constants
const (
	// A4 page dimensions in mm
	pageWidthMM  = 210.0
	pageHeightMM = 297.0
	// Margin from page edges in mm
	pageMarginMM = 20.0
	// Temp file prefix
	tempFilePrefix = "dokumentenscanner"
	// Temp file permissions (read/write for owner, read for group/others)
	tempFileMode = 0644
)

// GeneratePDF generates a PDF from an image.
func GeneratePDF(img []byte) ([]byte, error) {
	// Validate image data
	if len(img) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}

	// Decode to get dimensions and format
	decodedImg, format, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	bounds := decodedImg.Bounds()
	imgWidth := float64(bounds.Dx())
	imgHeight := float64(bounds.Dy())

	// Determine file extension based on format
	ext := ".png"
	if format == "jpeg" {
		ext = ".jpg"
	}

	// Create temp file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, tempFilePrefix+ext)
	if err := os.WriteFile(tmpFile, img, tempFileMode); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	// Create PDF
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Calculate scale to fit on page with margins
	scale := 1.0
	if imgWidth > pageWidthMM-pageMarginMM {
		scale = (pageWidthMM - pageMarginMM) / imgWidth
	}
	if imgHeight*scale > pageHeightMM-pageMarginMM {
		scale = (pageHeightMM - pageMarginMM) / imgHeight
	}

	// Center the image on the page
	x := (pageWidthMM - imgWidth*scale) / 2
	y := (pageHeightMM - imgHeight*scale) / 2

	// Add image to PDF
	pdf.Image(tmpFile, x, y, imgWidth*scale, imgHeight*scale, false, "", 0, "")

	// Generate PDF bytes
	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %v", err)
	}

	// Validate PDF was generated
	pdfBytes := buf.Bytes()
	if len(pdfBytes) == 0 {
		return nil, fmt.Errorf("generated PDF is empty")
	}

	return pdfBytes, nil
}
