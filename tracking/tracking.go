package tracking

import (
	"image"
	"log"

	"gocv.io/x/gocv"

	"tracker/types"
	"tracker/utils"
)

// TryTrackingRecovery attempts to recover lost tracking by searching around the last known position
func TryTrackingRecovery(frame gocv.Mat, tracker gocv.Tracker, lastRect image.Rectangle, searchRadius int) bool {
	// Expand search area around last known position
	centerX := lastRect.Min.X + lastRect.Dx()/2
	centerY := lastRect.Min.Y + lastRect.Dy()/2

	// Create expanded search rectangle
	searchRect := image.Rect(
		centerX-searchRadius,
		centerY-searchRadius,
		centerX+searchRadius,
		centerY+searchRadius,
	)

	// Ensure search area is within frame bounds
	if searchRect.Min.X < 0 {
		searchRect.Min.X = 0
	}
	if searchRect.Min.Y < 0 {
		searchRect.Min.Y = 0
	}
	if searchRect.Max.X > frame.Cols() {
		searchRect.Max.X = frame.Cols()
	}
	if searchRect.Max.Y > frame.Rows() {
		searchRect.Max.Y = frame.Rows()
	}

	// Try to reinitialize tracker with expanded area
	return tracker.Init(frame, searchRect)
}

// ProcessAutoTracking handles automatic object detection and tracking initialization
func ProcessAutoTracking(state *types.AppState, frame gocv.Mat, config types.TrackingConfig) {
	if state.TrackingEnabled || !state.AutoTrackingEnabled || state.ROISelectionMode || state.FrameCount <= config.StabilizationFrames {
		return
	}

	if err := state.BackSub.Apply(frame, &state.FgMask); err != nil {
		log.Printf("Error applying background subtractor: %v", err)
		return
	}

	// Find contours of moving objects
	contours := gocv.FindContours(state.FgMask, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	var largestContourIndex = -1
	var largestArea float64

	for i := 0; i < contours.Size(); i++ {
		contour := contours.At(i)
		area := gocv.ContourArea(contour)
		if area > largestArea && area > config.MinContourArea {
			largestArea = area
			largestContourIndex = i
		}
	}

	if largestContourIndex >= 0 {
		largestContour := contours.At(largestContourIndex)
		roi := gocv.BoundingRect(largestContour)

		// Add padding to the bounding box
		padding := 20
		roi = image.Rect(
			roi.Min.X-padding,
			roi.Min.Y-padding,
			roi.Max.X+padding,
			roi.Max.Y+padding,
		)

		// Ensure ROI is within image bounds
		if roi.Min.X < 0 {
			roi.Min.X = 0
		}
		if roi.Min.Y < 0 {
			roi.Min.Y = 0
		}
		if roi.Max.X > frame.Cols() {
			roi.Max.X = frame.Cols()
		}
		if roi.Max.Y > frame.Rows() {
			roi.Max.Y = frame.Rows()
		}

		if state.Tracker.Init(frame, roi) {
			state.ROI = roi
			state.TrackingEnabled = true
			state.AutoTrackingEnabled = false
			state.InitialROISize = (roi.Dx() + roi.Dy()) / 2
			log.Printf("Auto-tracking started! ROI: %dx%d at (%d,%d)\n", roi.Dx(), roi.Dy(), roi.Min.X, roi.Min.Y)
		}
	}
}

// ProcessTracking handles active object tracking and failure recovery
func ProcessTracking(state *types.AppState, frame gocv.Mat, config types.TrackingConfig) image.Rectangle {
	if !state.TrackingEnabled {
		return image.Rectangle{}
	}

	rect, ok := state.Tracker.Update(frame)
	if ok {
		// Successful tracking - reset failure count
		state.TrackingFailureCount = 0

		// Check for dramatic size changes that might indicate target switching
		currentSize := (rect.Dx() + rect.Dy()) / 2
		if !state.LastKnownRect.Empty() {
			lastSize := (state.LastKnownRect.Dx() + state.LastKnownRect.Dy()) / 2
			sizeRatio := float64(currentSize) / float64(lastSize)
			if sizeRatio > config.SizeChangeThreshold || sizeRatio < (1.0/config.SizeChangeThreshold) {
				// Suspect target switching - use last known position and increment failure count
				log.Printf("Suspicious size change detected (ratio: %.2f), using last known position\n", sizeRatio)
				state.TrackingFailureCount++
				rect = state.LastKnownRect
				return rect
			}
		}

		// Apply adaptive bounding box size control
		rect = utils.ConstrainBoundingBox(rect, state.InitialROISize, config.MaxROIGrowth, config.MinROISize, frame.Cols(), frame.Rows())
		state.LastKnownRect = rect
		return rect
	}

	// Tracking failed - increment failure count and try recovery
	state.TrackingFailureCount++
	log.Printf("Tracking failure %d/%d\n", state.TrackingFailureCount, config.MaxTrackingFailures)

	if state.TrackingFailureCount >= config.MaxTrackingFailures {
		// Too many failures, give up and re-enable auto-tracking
		state.TrackingEnabled = false
		state.TrackingFailureCount = 0
		if !state.ROISelectionMode {
			state.AutoTrackingEnabled = true
			log.Println("Tracking lost permanently. Re-enabling auto-tracking...")
		} else {
			log.Println("Tracking lost permanently.")
		}
		return image.Rectangle{}
	}

	if !state.LastKnownRect.Empty() {
		// Try to recover tracking using last known position
		if TryTrackingRecovery(frame, state.Tracker, state.LastKnownRect, config.SearchRadius) {
			log.Printf("Tracking recovery successful at attempt %d\n", state.TrackingFailureCount)
			state.TrackingFailureCount = 0
		}
		return state.LastKnownRect
	}

	return image.Rectangle{}
}

// InitializeTracking initializes tracking with a given ROI
func InitializeTracking(state *types.AppState, frame gocv.Mat, roi image.Rectangle) bool {
	if state.Tracker.Init(frame, roi) {
		state.ROI = roi
		state.TrackingEnabled = true
		state.ROISelectionMode = false
		state.AutoTrackingEnabled = false
		state.InitialROISize = (roi.Dx() + roi.Dy()) / 2
		return true
	}
	return false
}

// ResetTracking resets all tracking state
func ResetTracking(state *types.AppState) {
	state.TrackingEnabled = false
	state.AutoTrackingEnabled = true
	state.ROISelectionMode = false
	state.FrameCount = 0
}

// EnableAutoTracking starts auto-tracking mode
func EnableAutoTracking(state *types.AppState) {
	if !state.ROISelectionMode {
		state.AutoTrackingEnabled = true
		state.TrackingEnabled = false
		state.FrameCount = 0
	}
}
