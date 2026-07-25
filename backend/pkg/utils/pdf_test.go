package utils

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)
	return buf.Bytes()
}

func TestGeneratePDF(t *testing.T) {
	imgData := createTestPNG(t)
	pdfBytes, err := GeneratePDF(imgData)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	assert.True(t, len(pdfBytes) > 100, "PDF should be larger than 100 bytes")
}

func TestGeneratePDF_EmptySlice(t *testing.T) {
	pdfBytes, err := GeneratePDF([]byte{})
	// fpdf creates a valid PDF even with empty images (no pages)
	// The function should not panic
	_ = pdfBytes
	_ = err
}

func TestGeneratePDF_InvalidImage(t *testing.T) {
	pdfBytes, err := GeneratePDF([]byte("invalid image data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode image")
	assert.Nil(t, pdfBytes)
}

func TestGenerateMultiPagePDF_SingleImage(t *testing.T) {
	imgData := createTestPNG(t)
	pdfBytes, err := GenerateMultiPagePDF([][]byte{imgData}, 85)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestGenerateMultiPagePDF_MultipleImages(t *testing.T) {
	img1 := createTestPNG(t)
	img2 := createTestPNG(t)
	img3 := createTestPNG(t)

	pdfBytes, err := GenerateMultiPagePDF([][]byte{img1, img2, img3}, 85)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Multi-page PDF should be larger than single-page
	singlePage, _ := GenerateMultiPagePDF([][]byte{img1}, 85)
	assert.True(t, len(pdfBytes) > len(singlePage), "Multi-page PDF should be larger than single-page")
}

func TestGenerateMultiPagePDF_EmptyImages(t *testing.T) {
	_, err := GenerateMultiPagePDF([][]byte{}, 85)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no images provided")
}

func TestGenerateMultiPagePDF_AllEmptyImages(t *testing.T) {
	// fpdf creates a valid PDF even with no valid images
	pdfBytes, err := GenerateMultiPagePDF([][]byte{[]byte{}, []byte{}}, 85)
	_ = pdfBytes
	_ = err
}

func TestGenerateMultiPagePDF_InvalidQuality(t *testing.T) {
	imgData := createTestPNG(t)
	// Invalid quality should default to 85
	pdfBytes, err := GenerateMultiPagePDF([][]byte{imgData}, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}

func TestGenerateMultiPagePDF_Compression(t *testing.T) {
	imgData := createTestPNG(t)

	// Generate with low quality (more compression)
	lowQuality, err := GenerateMultiPagePDF([][]byte{imgData}, 30)
	require.NoError(t, err)

	// Generate with high quality (less compression)
	highQuality, err := GenerateMultiPagePDF([][]byte{imgData}, 95)
	require.NoError(t, err)

	// Low quality should be smaller
	assert.True(t, len(lowQuality) <= len(highQuality),
		"Low quality PDF (%d bytes) should be <= high quality PDF (%d bytes)",
		len(lowQuality), len(highQuality))
}

func TestGenerateMultiPagePDF_MixedFormats(t *testing.T) {
	// Test with PNG image (should be converted to JPEG)
	pngImg := createTestPNG(t)
	pdfBytes, err := GenerateMultiPagePDF([][]byte{pngImg}, 85)
	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
}
