package recording

import (
	"fmt"
	"time"

	"gocv.io/x/gocv"

	"tracker/types"
)

// StartRecording starts video recording with the given configuration
func StartRecording(state *types.AppState, frame gocv.Mat, config types.VideoConfig) error {
	if state.IsRecording {
		return fmt.Errorf("recording already active")
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("tracking_video_%s.mp4", timestamp)

	// Try different codecs for better compatibility
	var vw *gocv.VideoWriter
	var err error
	var usedCodec string

	for _, fourcc := range config.Codecs {
		vw, err = gocv.VideoWriterFile(filename, fourcc, config.FPS, frame.Cols(), frame.Rows(), true)
		if err == nil {
			usedCodec = fourcc
			break
		}
	}

	if err != nil {
		return fmt.Errorf("could not create video writer with any codec: %v", err)
	}

	state.VideoWriter = vw
	state.IsRecording = true
	state.RecordingStartTime = time.Now()
	fmt.Printf("Recording started: %s (codec: %s)\n", filename, usedCodec)

	return nil
}

// StopRecording stops video recording
func StopRecording(state *types.AppState) error {
	if !state.IsRecording {
		return fmt.Errorf("no active recording")
	}

	if state.VideoWriter != nil {
		if err := state.VideoWriter.Close(); err != nil {
			return fmt.Errorf("error closing video writer: %v", err)
		}
		state.VideoWriter = nil
	}

	state.IsRecording = false
	fmt.Println("Recording stopped")

	return nil
}

// ToggleRecording toggles video recording on/off
func ToggleRecording(state *types.AppState, frame gocv.Mat, config types.VideoConfig) error {
	if state.IsRecording {
		return StopRecording(state)
	}
	return StartRecording(state, frame, config)
}

// WriteFrame writes a frame to the video file if recording is active
func WriteFrame(state *types.AppState, frame gocv.Mat) error {
	if state.IsRecording && state.VideoWriter != nil {
		return state.VideoWriter.Write(frame)
	}
	return nil
}

// GetRecordingDuration returns the duration of the current recording
func GetRecordingDuration(state *types.AppState) time.Duration {
	if !state.IsRecording {
		return 0
	}
	return time.Since(state.RecordingStartTime)
}

// CleanupRecording ensures recording is properly stopped and cleaned up
func CleanupRecording(state *types.AppState) {
	if state.IsRecording {
		_ = StopRecording(state)
	}
}
