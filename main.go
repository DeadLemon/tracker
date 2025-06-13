package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sort"

	"gocv.io/x/gocv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tracker [camera ID or video file]")
		return
	}

	// open capture device
	device := os.Args[1]
	var capture *gocv.VideoCapture
	var err error
	if _, err = os.Stat(device); err == nil {
		capture, err = gocv.VideoCaptureFile(device)
	} else {
		capture, err = gocv.VideoCaptureDevice(parseCameraID(device))
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	defer capture.Close()

	window := gocv.NewWindow("Flying Object Detector")
	defer window.Close()

	bg := gocv.NewBackgroundSubtractorKNN()
	defer bg.Close()

	prevGray := gocv.NewMat()
	defer prevGray.Close()

	var trajectory []image.Point

	frame := gocv.NewMat()
	defer frame.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(5, 5))
	defer kernel.Close()

	for {
		if ok := capture.Read(&frame); !ok || frame.Empty() {
			break
		}

		stabilized := stabilizeFrame(frame, &prevGray)
		mask := gocv.NewMat()
		bg.Apply(stabilized, &mask)
		gocv.Threshold(mask, &mask, 25, 255, gocv.ThresholdBinary)
		gocv.MorphologyEx(mask, &mask, gocv.MorphOpen, kernel)
		gocv.MorphologyEx(mask, &mask, gocv.MorphClose, kernel)

		contours := gocv.FindContours(mask, gocv.RetrievalExternal, gocv.ChainApproxSimple)
		for i := range contours {
			area := gocv.ContourArea(contours[i])
			if area < 500 {
				continue
			}
			rect := gocv.BoundingRect(contours[i])
			center := image.Pt(rect.Min.X+rect.Dx()/2, rect.Min.Y+rect.Dy()/2)
			trajectory = append(trajectory, center)
			gocv.Rectangle(&frame, rect, color.RGBA{0, 255, 0, 0}, 2)
		}

		for i := 1; i < len(trajectory); i++ {
			gocv.Line(&frame, trajectory[i-1], trajectory[i], color.RGBA{255, 0, 0, 0}, 2)
		}

		window.IMShow(frame)
		if window.WaitKey(1) == 27 { // ESC
			break
		}
	}
}

func parseCameraID(arg string) int {
	var id int
	fmt.Sscanf(arg, "%d", &id)
	return id
}

type translation struct {
	dx, dy float64
}

func stabilizeFrame(frame gocv.Mat, prevGray *gocv.Mat) gocv.Mat {
	gray := gocv.NewMat()
	gocv.CvtColor(frame, &gray, gocv.ColorBGRToGray)
	if prevGray.Empty() {
		*prevGray = gray.Clone()
		return frame
	}
	prevPts := gocv.NewMat()
	gocv.GoodFeaturesToTrack(*prevGray, &prevPts, 200, 0.01, 30)
	if prevPts.Empty() {
		*prevGray = gray.Clone()
		return frame
	}
	nextPts := gocv.NewMat()
	status := gocv.NewMat()
	err := gocv.NewMat()
	gocv.CalcOpticalFlowPyrLK(*prevGray, gray, prevPts, &nextPts, &status, &err)

	var moves []translation
	for i := 0; i < status.Rows(); i++ {
		if status.GetUCharAt(i, 0) == 1 {
			dx := nextPts.GetFloatAt(i, 0) - prevPts.GetFloatAt(i, 0)
			dy := nextPts.GetFloatAt(i, 1) - prevPts.GetFloatAt(i, 1)
			moves = append(moves, translation{dx: float64(dx), dy: float64(dy)})
		}
	}

	*prevGray = gray.Clone()

	if len(moves) == 0 {
		return frame
	}

	dx, dy := medianTranslation(moves)

	mat := gocv.NewMatWithSize(2, 3, gocv.MatTypeCV64F)
	mat.SetDoubleAt(0, 0, 1)
	mat.SetDoubleAt(0, 1, 0)
	mat.SetDoubleAt(0, 2, -dx)
	mat.SetDoubleAt(1, 0, 0)
	mat.SetDoubleAt(1, 1, 1)
	mat.SetDoubleAt(1, 2, -dy)

	result := gocv.NewMat()
	gocv.WarpAffine(frame, &result, mat, image.Pt(frame.Cols(), frame.Rows()))

	return result
}

func medianTranslation(moves []translation) (float64, float64) {
	if len(moves) == 0 {
		return 0, 0
	}
	xs := make([]float64, len(moves))
	ys := make([]float64, len(moves))
	for i, m := range moves {
		xs[i] = m.dx
		ys[i] = m.dy
	}
	return median(xs), median(ys)
}

func median(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
