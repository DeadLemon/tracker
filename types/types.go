package types

import (
	"image"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

// AppState holds the complete application state
type AppState struct {
	// Tracking state
	Tracker             gocv.Tracker
	TrackingEnabled     bool
	AutoTrackingEnabled bool
	ROI                 image.Rectangle
	InitialROISize      int

	// Tracking robustness
	TrackingFailureCount int
	LastKnownRect        image.Rectangle

	// ROI selection
	ROISelectionMode bool
	ROICenterX       int
	ROICenterY       int
	ROIWidth         int
	ROIHeight        int

	// Video recording
	IsRecording        bool
	VideoWriter        *gocv.VideoWriter
	RecordingStartTime time.Time

	// Background subtraction
	BackSub gocv.BackgroundSubtractorMOG2
	FgMask  gocv.Mat

	// Frame processing
	FrameCount int

	// Debug logging
	DebugMode    bool
	DebugLogs    []string
	DebugLogMutex sync.Mutex
}

// TrackingConfig holds tracking configuration constants
type TrackingConfig struct {
	MaxROIGrowth        float64
	MinROISize          int
	MaxTrackingFailures int
	SearchRadius        int
	SizeChangeThreshold float64
	MinContourArea      float64
	StabilizationFrames int
}

// DefaultTrackingConfig returns the default tracking configuration
func DefaultTrackingConfig() TrackingConfig {
	return TrackingConfig{
		MaxROIGrowth:        2.0,
		MinROISize:          40,
		MaxTrackingFailures: 12,
		SearchRadius:        120,
		SizeChangeThreshold: 5.0,
		MinContourArea:      500,
		StabilizationFrames: 30,
	}
}

// VideoConfig holds video recording configuration
type VideoConfig struct {
	FPS    float64
	Codecs []string
}

// DefaultVideoConfig returns the default video configuration
func DefaultVideoConfig() VideoConfig {
	return VideoConfig{
		FPS:    30.0,
		Codecs: []string{"H264", "avc1", "x264", "mp4v"},
	}
}

// UIConfig holds UI configuration constants
type UIConfig struct {
	HelpFontSize   float64
	StatusFontSize float64
	HelpOffsetY    int
	MaxDebugLogs   int
	DebugFontSize  float64
}

// DebugLogger provides thread-safe debug logging functionality
type DebugLogger struct {
	state *AppState
	maxLogs int
	originalOutput io.Writer
}

// NewDebugLogger creates a new debug logger
func NewDebugLogger(state *AppState, maxLogs int) *DebugLogger {
	return &DebugLogger{
		state: state,
		maxLogs: maxLogs,
		originalOutput: log.Default().Writer(),
	}
}

// Log adds a debug message to the log buffer
func (d *DebugLogger) Log(message string) {
	if !d.state.DebugMode {
		return
	}
	
	d.state.DebugLogMutex.Lock()
	defer d.state.DebugLogMutex.Unlock()
	
	d.state.DebugLogs = append(d.state.DebugLogs, message)
	if len(d.state.DebugLogs) > d.maxLogs {
		d.state.DebugLogs = d.state.DebugLogs[1:]
	}
}

// GetLogs returns a copy of the current debug logs
func (d *DebugLogger) GetLogs() []string {
	d.state.DebugLogMutex.Lock()
	defer d.state.DebugLogMutex.Unlock()
	
	logs := make([]string, len(d.state.DebugLogs))
	copy(logs, d.state.DebugLogs)
	return logs
}

// Write implements io.Writer interface to capture log output
func (d *DebugLogger) Write(p []byte) (n int, err error) {
	// Forward to original output (console/stderr)
	if d.originalOutput != nil {
		_, _ = d.originalOutput.Write(p)
	}
	
	// Also capture in debug logs if debug mode is enabled
	if d.state.DebugMode {
		message := strings.TrimSpace(string(p))
		// Remove timestamp prefix that log package adds
		if len(message) > 19 && message[4] == '/' && message[7] == '/' && message[10] == ' ' {
			// Format: "2006/01/02 15:04:05 message"
			if spaceIndex := strings.Index(message[11:], " "); spaceIndex != -1 {
				message = message[11+spaceIndex+1:]
			}
		}
		if message != "" {
			d.Log(message)
		}
	}
	
	return len(p), nil
}

// SetAsLogOutput configures this debug logger to capture standard log output
func (d *DebugLogger) SetAsLogOutput() {
	log.SetOutput(d)
}

// RestoreOriginalLogOutput restores the original log output
func (d *DebugLogger) RestoreOriginalLogOutput() {
	if d.originalOutput != nil {
		log.SetOutput(d.originalOutput)
	}
}

// DefaultUIConfig returns the default UI configuration
func DefaultUIConfig() UIConfig {
	return UIConfig{
		HelpFontSize:   0.9,
		StatusFontSize: 1.5,
		HelpOffsetY:    60,
		MaxDebugLogs:   10,
		DebugFontSize:  0.8,
	}
}
