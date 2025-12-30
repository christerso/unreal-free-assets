package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// CreateIconFile creates the icon PNG file and returns its path
func createIconFile() string {
	iconPath := filepath.Join(dataDir, "icon.png")
	
	// Check if icon already exists
	if _, err := os.Stat(iconPath); err == nil {
		return iconPath
	}
	
	// Create icon
	size := 64
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	orange := color.NRGBA{245, 130, 32, 255}
	white := color.NRGBA{255, 255, 255, 255}

	center := float64(size) / 2
	radius := float64(size)/2 - 2

	// Draw filled orange circle
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

	// Draw "U" in white
	for y := 16; y <= 40; y++ {
		for x := 20; x <= 25; x++ {
			img.Set(x, y, white)
		}
	}
	for y := 16; y <= 40; y++ {
		for x := 38; x <= 43; x++ {
			img.Set(x, y, white)
		}
	}
	for y := 40; y <= 47; y++ {
		for x := 20; x <= 43; x++ {
			img.Set(x, y, white)
		}
	}

	// Save to file
	f, err := os.Create(iconPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	
	png.Encode(f, img)
	return iconPath
}

// LoadIconResource loads icon from file as Fyne resource
func loadIconResource() []byte {
	iconPath := createIconFile()
	if iconPath == "" {
		return nil
	}
	
	data, err := os.ReadFile(iconPath)
	if err != nil {
		return nil
	}
	return data
}

// iconData will be set after dataDir is initialized
var iconData []byte

func initIcon() {
	iconData = loadIconResource()
}

// Alternative: generate PNG bytes in memory
func generateIconBytes() []byte {
	size := 64
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	orange := color.NRGBA{245, 130, 32, 255}
	white := color.NRGBA{255, 255, 255, 255}

	center := float64(size) / 2
	radius := float64(size)/2 - 2

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

	// U letter
	for y := 16; y <= 40; y++ {
		for x := 20; x <= 25; x++ {
			img.Set(x, y, white)
		}
		for x := 38; x <= 43; x++ {
			img.Set(x, y, white)
		}
	}
	for y := 40; y <= 47; y++ {
		for x := 20; x <= 43; x++ {
			img.Set(x, y, white)
		}
	}

	buf := new(bytes.Buffer)
	png.Encode(buf, img)
	return buf.Bytes()
}
