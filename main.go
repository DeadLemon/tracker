package main

import (
	"fmt"
	"log"

	"gocv.io/x/gocv"
	"gocv.io/x/gocv/contrib"

	"tracker/input"
	"tracker/recording"
	"tracker/tracking"
	"tracker/types"
	"tracker/ui"
)

func main() {
	// Initialize video capture
	vc, err := gocv.OpenVideoCapture(0)
	if err != nil {
		log.Fatal("failed to open video capture device:", err)
	}
	defer func() { _ = vc.Close() }()

	// Initialize window
	w := gocv.NewWindow("tracker")
	defer func() { _ = w.Close() }()

	// Initialize tracker
	tracker := contrib.NewTrackerCSRT()
	defer func() { _ = tracker.Close() }()

	// Initialize frame matrix
	frame := gocv.NewMat()
	defer func() { _ = frame.Close() }()

	// Initialize application state
	state := &types.AppState{
		Tracker:             tracker,
		AutoTrackingEnabled: true,
		BackSub:             gocv.NewBackgroundSubtractorMOG2(),
		FgMask:              gocv.NewMat(),
	}
	defer func() { _ = state.BackSub.Close() }()
	defer func() { _ = state.FgMask.Close() }()

	// Initialize ROI selection defaults
	input.InitializeROISelection(state)

	// Load configurations
	trackingConfig := types.DefaultTrackingConfig()
	videoConfig := types.DefaultVideoConfig()
	uiConfig := types.DefaultUIConfig()
	
	// Initialize debug logger
	debugLogger := types.NewDebugLogger(state, uiConfig.MaxDebugLogs)
	
	// Set debug logger to capture standard log output
	debugLogger.SetAsLogOutput()
	defer debugLogger.RestoreOriginalLogOutput()

	// Print startup instructions
	ui.PrintStartupInstructions()

	// Main loop
	for {
		// Read frame from camera
		if ok := vc.Read(&frame); !ok {
			break
		}

		if frame.Empty() {
			continue
		}

		// Mirror the image horizontally
		if err := gocv.Flip(frame, &frame, 1); err != nil {
			log.Printf("Error flipping image: %v", err)
			continue
		}

		state.FrameCount++
		
		// Debug logging for frame processing (every 60 frames to avoid spam)
		if state.FrameCount%60 == 0 {
			debugLogger.Log(fmt.Sprintf("Frame %d processed", state.FrameCount))
		}

		// Process auto-tracking
		tracking.ProcessAutoTracking(state, frame, trackingConfig)

		// Enable auto-tracking by default if nothing is active
		if !state.TrackingEnabled && !state.AutoTrackingEnabled && !state.ROISelectionMode {
			state.AutoTrackingEnabled = true
			debugLogger.Log("Auto-tracking re-enabled (default state)")
		}

		// Process tracking and get current rectangle
		trackingRect := tracking.ProcessTracking(state, frame, trackingConfig)
		trackingSuccess := state.TrackingEnabled && !trackingRect.Empty() && state.TrackingFailureCount == 0
		
		// Debug logging for tracking state (less frequent to avoid spam)
		if state.TrackingEnabled && state.FrameCount%30 == 0 {
			debugLogger.Log(fmt.Sprintf("Tracking: %dx%d at (%d,%d)", 
				trackingRect.Dx(), trackingRect.Dy(), trackingRect.Min.X, trackingRect.Min.Y))
		} else if state.AutoTrackingEnabled && state.FrameCount%120 == 0 {
			debugLogger.Log("Auto-tracking: searching for objects...")
		}

		// Write frame to video if recording
		if err := recording.WriteFrame(state, frame); err != nil {
			log.Printf("Error writing video frame: %v", err)
		} else if state.IsRecording && state.FrameCount%30 == 0 {
			// Log recording status every 30 frames to avoid spam
			debugLogger.Log("Recording active")
		}

		// Render all UI elements
		ui.RenderFrame(&frame, state, trackingRect, trackingSuccess, uiConfig)

		// Display frame
		_ = w.IMShow(frame)
		key := w.WaitKey(15)

		// Process input and check for quit
		if shouldQuit := input.ProcessInput(key, state, frame, trackingConfig, videoConfig); shouldQuit {
			break
		}
	}

	// Cleanup recording on exit
	recording.CleanupRecording(state)
}
