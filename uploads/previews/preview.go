package uploads

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"

	"github.com/disintegration/imaging"
)

func GenerateImagePreview(srcPath string, width int) (string, error) {
	// Open the image file
	img, err := imaging.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}

	// Resize while preserving aspect ratio
	preview := imaging.Resize(img, width, 0, imaging.Lanczos)

	// Save the preview
	base := filepath.Base(srcPath)
	previewPath := filepath.Join("uploads/previews/images", base)
	err = imaging.Save(preview, previewPath)
	if err != nil {
		return "", fmt.Errorf("failed to save preview: %w", err)
	}

	return previewPath, nil
}

func GeneratePDFPreview(srcPath string, width int) (string, error) {
	// Create a simple PDF placeholder image
	height := width * 3 / 4 // Common document aspect ratio

	// Create a gray background (this will be our border)
	borderSize := 2
	grayBg := imaging.New(width, height, color.RGBA{200, 200, 200, 255})

	// Create a white rectangle for the inner content
	whiteRect := imaging.New(width-(borderSize*2), height-(borderSize*2), color.White)

	// Paste the white rectangle onto the gray background to create border effect
	img := imaging.Paste(grayBg, whiteRect, image.Pt(borderSize, borderSize))

	base := filepath.Base(srcPath)
	previewPath := filepath.Join("uploads/previews/pdf", base+".png")
	err := imaging.Save(img, previewPath)
	if err != nil {
		return "", fmt.Errorf("failed to save PDF preview: %w", err)
	}

	return previewPath, nil
}
