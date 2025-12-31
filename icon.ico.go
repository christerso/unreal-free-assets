//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	// Create multiple sizes for ICO
	sizes := []int{16, 32, 48, 256}
	var pngData [][]byte

	for _, size := range sizes {
		img := createIcon(size)
		var buf bytes.Buffer
		png.Encode(&buf, img)
		pngData = append(pngData, buf.Bytes())
	}

	// Write ICO file
	writeICO("icon.ico", sizes, pngData)
}

func createIcon(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Unreal orange color
	orange := color.RGBA{245, 130, 32, 255}
	white := color.RGBA{255, 255, 255, 255}

	center := float64(size) / 2
	radius := float64(size) / 2.2

	// Draw orange circle
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center + 0.5
			dy := float64(y) - center + 0.5
			dist := dx*dx + dy*dy
			if dist <= radius*radius {
				img.Set(x, y, orange)
			}
		}
	}

	// Draw "U" for Unreal
	uWidth := float64(size) * 0.5
	uHeight := float64(size) * 0.5
	uThickness := float64(size) * 0.12
	uLeft := center - uWidth/2
	uTop := center - uHeight/2

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx, fy := float64(x), float64(y)

			// Left vertical bar
			if fx >= uLeft && fx <= uLeft+uThickness &&
				fy >= uTop && fy <= uTop+uHeight*0.7 {
				img.Set(x, y, white)
			}

			// Right vertical bar
			if fx >= uLeft+uWidth-uThickness && fx <= uLeft+uWidth &&
				fy >= uTop && fy <= uTop+uHeight*0.7 {
				img.Set(x, y, white)
			}

			// Bottom curve (simplified as rectangle with rounded bottom)
			if fy >= uTop+uHeight*0.5 && fy <= uTop+uHeight {
				dx := fx - center
				halfWidth := uWidth / 2
				curveY := uTop + uHeight*0.5 + (uHeight*0.5)*(1-dx*dx/(halfWidth*halfWidth))
				if fy <= curveY && fx >= uLeft && fx <= uLeft+uWidth {
					// Check if on the edge
					innerLeft := uLeft + uThickness
					innerRight := uLeft + uWidth - uThickness
					innerCurveY := uTop + uHeight*0.5 + (uHeight*0.35)*(1-dx*dx/((halfWidth-uThickness)*(halfWidth-uThickness)))
					if fx < innerLeft || fx > innerRight || fy > innerCurveY {
						img.Set(x, y, white)
					}
				}
			}
		}
	}

	return img
}

func writeICO(filename string, sizes []int, pngData [][]byte) {
	f, _ := os.Create(filename)
	defer f.Close()

	// ICO header
	binary.Write(f, binary.LittleEndian, uint16(0))      // Reserved
	binary.Write(f, binary.LittleEndian, uint16(1))      // Type: 1 = ICO
	binary.Write(f, binary.LittleEndian, uint16(len(sizes))) // Number of images

	// Calculate offsets
	offset := 6 + len(sizes)*16 // Header + directory entries

	// Write directory entries
	for i, size := range sizes {
		w := uint8(size)
		h := uint8(size)
		if size >= 256 {
			w, h = 0, 0 // 0 means 256
		}
		binary.Write(f, binary.LittleEndian, w)                           // Width
		binary.Write(f, binary.LittleEndian, h)                           // Height
		binary.Write(f, binary.LittleEndian, uint8(0))                    // Color palette
		binary.Write(f, binary.LittleEndian, uint8(0))                    // Reserved
		binary.Write(f, binary.LittleEndian, uint16(1))                   // Color planes
		binary.Write(f, binary.LittleEndian, uint16(32))                  // Bits per pixel
		binary.Write(f, binary.LittleEndian, uint32(len(pngData[i])))     // Size of image data
		binary.Write(f, binary.LittleEndian, uint32(offset))              // Offset to image data
		offset += len(pngData[i])
	}

	// Write image data
	for _, data := range pngData {
		f.Write(data)
	}
}
