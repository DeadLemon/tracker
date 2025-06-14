package utils

import (
	"image"
)

// ConstrainBoundingBox prevents the bounding box from growing too large
func ConstrainBoundingBox(rect image.Rectangle, initialSize int, maxGrowth float64, minSize, imgWidth, imgHeight int) image.Rectangle {
	// Calculate current size
	currentWidth := rect.Dx()
	currentHeight := rect.Dy()

	// Calculate maximum allowed size based on initial size
	maxAllowedSize := int(float64(initialSize) * maxGrowth)

	// Constrain width and height
	newWidth := currentWidth
	newHeight := currentHeight

	if currentWidth > maxAllowedSize {
		newWidth = maxAllowedSize
	}
	if currentHeight > maxAllowedSize {
		newHeight = maxAllowedSize
	}

	// Ensure minimum size
	if newWidth < minSize {
		newWidth = minSize
	}
	if newHeight < minSize {
		newHeight = minSize
	}

	// Calculate center point of original rectangle
	centerX := rect.Min.X + rect.Dx()/2
	centerY := rect.Min.Y + rect.Dy()/2

	// Create new rectangle centered on the same point
	halfWidth := newWidth / 2
	halfHeight := newHeight / 2

	newRect := image.Rect(
		centerX-halfWidth,
		centerY-halfHeight,
		centerX+halfWidth,
		centerY+halfHeight,
	)

	// Ensure the rectangle stays within image bounds
	if newRect.Min.X < 0 {
		newRect = newRect.Add(image.Pt(-newRect.Min.X, 0))
	}
	if newRect.Min.Y < 0 {
		newRect = newRect.Add(image.Pt(0, -newRect.Min.Y))
	}
	if newRect.Max.X > imgWidth {
		newRect = newRect.Add(image.Pt(imgWidth-newRect.Max.X, 0))
	}
	if newRect.Max.Y > imgHeight {
		newRect = newRect.Add(image.Pt(0, imgHeight-newRect.Max.Y))
	}

	return newRect
}
