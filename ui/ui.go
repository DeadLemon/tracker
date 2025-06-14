package ui

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"gocv.io/x/gocv"

	"tracker/recording"
	"tracker/types"
)

var (
	Blue   = color.RGBA{B: 255}
	Red    = color.RGBA{R: 255}
	Green  = color.RGBA{G: 255}
	Yellow = color.RGBA{R: 255, G: 255}
	White  = color.RGBA{R: 255, G: 255, B: 255}
	Black  = color.RGBA{R: 0, G: 0, B: 0, A: 120}
)

// DrawTrackingRect draws the tracking rectangle on the frame
func DrawTrackingRect(frame *gocv.Mat, rect image.Rectangle, success bool) {
	rectColor := Blue
	if !success {
		rectColor = Red
	}
	_ = gocv.Rectangle(frame, rect, rectColor, 3)
}

// DrawROISelection draws the ROI selection rectangle and crosshair
func DrawROISelection(frame *gocv.Mat, state *types.AppState) {
	if !state.ROISelectionMode {
		return
	}

	// Calculate ROI bounds
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

	selectionRect := image.Rect(x1, y1, x2, y2)
	_ = gocv.Rectangle(frame, selectionRect, Yellow, 2)

	// Draw center cross
	_ = gocv.Line(frame, image.Pt(state.ROICenterX-10, state.ROICenterY), image.Pt(state.ROICenterX+10, state.ROICenterY), Yellow, 1)
	_ = gocv.Line(frame, image.Pt(state.ROICenterX, state.ROICenterY-10), image.Pt(state.ROICenterX, state.ROICenterY+10), Yellow, 1)
}

// DrawStatusMessage draws the main status message
func DrawStatusMessage(frame *gocv.Mat, state *types.AppState, config types.UIConfig) {
	var statusText string
	var textColor color.RGBA

	switch {
	case state.ROISelectionMode:
		statusText = "Arrow keys/WASD: move, +/-: resize, ENTER: confirm, ESC: cancel"
		textColor = Yellow
	case !state.TrackingEnabled:
		if state.AutoTrackingEnabled {
			statusText = "Auto-tracking: Looking for objects..."
		} else {
			statusText = "Press 's' for manual ROI or 'a' for auto"
		}
		textColor = Red
	default:
		statusText = "Tracking active"
		textColor = Green
	}

	if err := gocv.PutText(frame, statusText, image.Pt(10, 30), gocv.FontHersheyPlain, config.StatusFontSize, textColor, 2); err != nil {
		log.Printf("Error adding status text: %v", err)
	}
}

// DrawRecordingStatus draws the recording status and timer
func DrawRecordingStatus(frame *gocv.Mat, state *types.AppState, config types.UIConfig) {
	if !state.IsRecording {
		return
	}

	duration := recording.GetRecordingDuration(state)
	recordingText := fmt.Sprintf("REC %02d:%02d", int(duration.Minutes()), int(duration.Seconds())%60)

	if err := gocv.PutText(frame, recordingText, image.Pt(10, 60), gocv.FontHersheyPlain, config.StatusFontSize, Red, 2); err != nil {
		log.Printf("Error adding recording text: %v", err)
	}
}

// DrawHelpText draws the compact help text in the bottom corner
func DrawHelpText(frame *gocv.Mat, state *types.AppState, config types.UIConfig) {
	helpY := frame.Rows() - config.HelpOffsetY

	var helpText string
	if state.ROISelectionMode {
		helpText = "ROI: Arrows=move  +/-=resize  Enter=confirm  Esc=cancel"
	} else {
		helpText = "Controls: s=ROI  a=auto  r=reset  v=record  d=debug  q=quit"
	}

	// Small background for readability
	textSize := gocv.GetTextSize(helpText, gocv.FontHersheyPlain, config.HelpFontSize, 1)
	helpRect := image.Rect(5, helpY-5, textSize.X+15, helpY+textSize.Y+5)

	if err := gocv.Rectangle(frame, helpRect, Black, -1); err != nil {
		log.Printf("Error drawing help background: %v", err)
	}

	if err := gocv.PutText(frame, helpText, image.Pt(10, helpY+10), gocv.FontHersheyPlain, config.HelpFontSize, White, 1); err != nil {
		log.Printf("Error adding help text: %v", err)
	}
}

// DrawDebugLogs draws the debug log messages on screen
func DrawDebugLogs(frame *gocv.Mat, state *types.AppState, config types.UIConfig) {
	if !state.DebugMode {
		return
	}

	state.DebugLogMutex.Lock()
	logs := make([]string, len(state.DebugLogs))
	copy(logs, state.DebugLogs)
	state.DebugLogMutex.Unlock()

	if len(logs) == 0 {
		return
	}

	// Calculate position for debug logs (right side of screen)
	frameWidth := frame.Cols()
	startY := 100
	lineHeight := 20
	maxWidth := 400
	padding := 10

	// Draw background for debug logs
	debugHeight := len(logs)*lineHeight + padding*2
	debugRect := image.Rect(frameWidth-maxWidth-padding, startY-padding, frameWidth-padding, startY+debugHeight-padding)
	
	if err := gocv.Rectangle(frame, debugRect, Black, -1); err != nil {
		log.Printf("Error drawing debug background: %v", err)
	}

	// Draw debug header
	headerText := fmt.Sprintf("Debug Logs (%d):", len(logs))
	if err := gocv.PutText(frame, headerText, image.Pt(frameWidth-maxWidth, startY), gocv.FontHersheyPlain, config.DebugFontSize, Yellow, 1); err != nil {
		log.Printf("Error adding debug header: %v", err)
	}

	// Draw each log message
	for i, logMsg := range logs {
		y := startY + (i+1)*lineHeight
		
		// Truncate long messages
		if len(logMsg) > 50 {
			logMsg = logMsg[:47] + "..."
		}
		
		if err := gocv.PutText(frame, logMsg, image.Pt(frameWidth-maxWidth, y), gocv.FontHersheyPlain, config.DebugFontSize, White, 1); err != nil {
			log.Printf("Error adding debug text: %v", err)
		}
	}
}

// RenderFrame renders all UI elements on the frame
func RenderFrame(frame *gocv.Mat, state *types.AppState, trackingRect image.Rectangle, trackingSuccess bool, config types.UIConfig) {
	// Draw tracking rectangle if tracking is active
	if state.TrackingEnabled && !trackingRect.Empty() {
		DrawTrackingRect(frame, trackingRect, trackingSuccess)
	}

	// Draw ROI selection if active
	DrawROISelection(frame, state)

	// Draw status messages
	DrawStatusMessage(frame, state, config)
	DrawRecordingStatus(frame, state, config)
	DrawHelpText(frame, state, config)
	
	// Draw debug logs if debug mode is enabled
	DrawDebugLogs(frame, state, config)
}

// PrintStartupInstructions prints the initial control instructions
func PrintStartupInstructions() {
	fmt.Println("Controls:")
	fmt.Println("- Auto-tracking starts automatically")
	fmt.Println("- Press 's' to start live ROI selection")
	fmt.Println("- In ROI mode: Arrow keys or WASD move, +/- resize, ENTER confirm, ESC cancel")
	fmt.Println("- Press 'a' to toggle auto-tracking")
	fmt.Println("- Press 'r' to reset tracking")
	fmt.Println("- Press 'v' to start/stop video recording")
	fmt.Println("- Press 'd' to toggle debug mode (shows last N logs on screen)")
	fmt.Println("- Press 'q' or ESC to quit")
}
