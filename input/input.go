package input

import (
	"image"
	"log"

	"gocv.io/x/gocv"

	"tracker/recording"
	"tracker/tracking"
	"tracker/types"
)

// HandleEscapeKey handles the ESC key press based on current mode
func HandleEscapeKey(state *types.AppState) bool {
	if state.ROISelectionMode {
		// Cancel ROI selection
		state.ROISelectionMode = false
		// Re-enable auto-tracking when cancelling ROI selection
		if !state.TrackingEnabled {
			state.AutoTrackingEnabled = true
		}
		log.Println("ROI selection cancelled")
		return false // Don't quit
	}

	// Stop recording if active before quitting
	recording.CleanupRecording(state)
	return true // Quit program
}

// HandleMainKeys handles the main application keyboard commands
func HandleMainKeys(key int, state *types.AppState, frame gocv.Mat, trackingConfig types.TrackingConfig, videoConfig types.VideoConfig) bool {
	switch key {
	case 'q': // 'q' to quit
		recording.CleanupRecording(state)
		return true

	case 's': // 's' to start live ROI selection
		if !state.ROISelectionMode {
			state.AutoTrackingEnabled = false
			state.TrackingEnabled = false
			state.ROISelectionMode = true
			// Set initial ROI center to image center
			state.ROICenterX = frame.Cols() / 2
			state.ROICenterY = frame.Rows() / 2
			log.Println("ROI selection mode. Use arrow keys or WASD to move, +/- to resize, ENTER to confirm.")
		}

	case 'a': // 'a' to toggle auto-tracking
		if !state.ROISelectionMode {
			if state.AutoTrackingEnabled {
				state.AutoTrackingEnabled = false
				state.TrackingEnabled = false
				log.Println("Auto-tracking disabled")
			} else {
				tracking.EnableAutoTracking(state)
				log.Println("Auto-tracking enabled")
			}
		}

	case 'r': // 'r' to reset tracking
		tracking.ResetTracking(state)
		log.Println("Tracking reset. Auto-tracking enabled")

	case 'v': // 'v' to toggle video recording
		if err := recording.ToggleRecording(state, frame, videoConfig); err != nil {
			log.Printf("Recording error: %v\n", err)
		}

	case 'd': // 'd' to toggle debug mode
		state.DebugMode = !state.DebugMode
		if state.DebugMode {
			log.Println("Debug mode enabled - logs will appear on screen")
		} else {
			log.Println("Debug mode disabled")
		}
	}

	return false // Don't quit
}

// HandleROIKeys handles keyboard input during ROI selection mode
func HandleROIKeys(key int, state *types.AppState, frame gocv.Mat) {
	if !state.ROISelectionMode {
		return
	}

	switch key {
	case 0: // Up arrow
		state.ROICenterY -= 10
		if state.ROICenterY < state.ROIHeight/2 {
			state.ROICenterY = state.ROIHeight / 2
		}
	case 1: // Down arrow
		state.ROICenterY += 10
		if state.ROICenterY > frame.Rows()-state.ROIHeight/2 {
			state.ROICenterY = frame.Rows() - state.ROIHeight/2
		}
	case 2: // Left arrow
		state.ROICenterX -= 10
		if state.ROICenterX < state.ROIWidth/2 {
			state.ROICenterX = state.ROIWidth / 2
		}
	case 3: // Right arrow
		state.ROICenterX += 10
		if state.ROICenterX > frame.Cols()-state.ROIWidth/2 {
			state.ROICenterX = frame.Cols() - state.ROIWidth/2
		}
	case '+', '=': // Increase size
		state.ROIWidth += 20
		state.ROIHeight += 20
		if state.ROIWidth > frame.Cols() {
			state.ROIWidth = frame.Cols()
		}
		if state.ROIHeight > frame.Rows() {
			state.ROIHeight = frame.Rows()
		}
	case '-', '_': // Decrease size
		state.ROIWidth -= 20
		state.ROIHeight -= 20
		if state.ROIWidth < 40 {
			state.ROIWidth = 40
		}
		if state.ROIHeight < 40 {
			state.ROIHeight = 40
		}
	case 13: // ENTER - confirm ROI
		halfWidth := state.ROIWidth / 2
		halfHeight := state.ROIHeight / 2
		x1 := state.ROICenterX - halfWidth
		y1 := state.ROICenterY - halfHeight
		x2 := state.ROICenterX + halfWidth
		y2 := state.ROICenterY + halfHeight

		// Ensure bounds are within image
		if x1 < 0 {
			x1 = 0
		}
		if y1 < 0 {
			y1 = 0
		}
		if x2 > frame.Cols() {
			x2 = frame.Cols()
		}
		if y2 > frame.Rows() {
			y2 = frame.Rows()
		}

		roi := image.Rect(x1, y1, x2, y2)

		if tracking.InitializeTracking(state, frame, roi) {
			log.Println("Manual tracking started!")
		} else {
			log.Println("Failed to initialize tracker")
			state.ROISelectionMode = false
		}
	}
}

// InitializeROISelection initializes ROI selection with default values
func InitializeROISelection(state *types.AppState) {
	state.ROICenterX = 320
	state.ROICenterY = 240
	state.ROIWidth = 100
	state.ROIHeight = 100
}

// ProcessInput processes all keyboard input for the application
func ProcessInput(key int, state *types.AppState, frame gocv.Mat, trackingConfig types.TrackingConfig, videoConfig types.VideoConfig) bool {
	// Handle ESC key first
	if key == 27 {
		return HandleEscapeKey(state)
	}

	// Handle main application keys
	if shouldQuit := HandleMainKeys(key, state, frame, trackingConfig, videoConfig); shouldQuit {
		return true
	}

	// Handle ROI selection keys
	HandleROIKeys(key, state, frame)

	return false
}
