package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

var (
	denimDark    = color.RGBA{R: 0x16, G: 0x29, B: 0x4A, A: 0xFF} // #16294A
	denimPrimary = color.RGBA{R: 0x1F, G: 0x3A, B: 0x68, A: 0xFF} // #1F3A68
	white        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	transparent  = color.RGBA{R: 0, G: 0, B: 0, A: 0}
)

func main() {
	if err := os.MkdirAll("frontend/public", 0755); err != nil {
		panic(err)
	}
	if err := os.MkdirAll("frontend/resources", 0755); err != nil {
		panic(err)
	}

	// 1. Favicons
	savePNG("frontend/public/favicon-16x16.png", renderIcon(16, denimPrimary, white, true))
	savePNG("frontend/public/favicon-32x32.png", renderIcon(32, denimPrimary, white, true))
	savePNG("frontend/public/apple-touch-icon.png", renderIcon(180, denimPrimary, white, true))
	savePNG("frontend/public/android-chrome-192x192.png", renderIcon(192, denimPrimary, white, true))
	savePNG("frontend/public/android-chrome-512x512.png", renderIcon(512, denimPrimary, white, true))

	// 2. OpenGraph 1200x630
	savePNG("frontend/public/og-image.png", renderOGImage(1200, 630))

	// 3. App icons (Capacitor)
	savePNG("frontend/resources/icon.png", renderIcon(1024, denimPrimary, white, true))
	savePNG("frontend/resources/icon-background.png", renderSolid(1024, 1024, denimDark))
	savePNG("frontend/resources/icon-foreground.png", renderIcon(1024, transparent, white, false))

	// 4. Splash 2732x2732
	savePNG("frontend/resources/splash.png", renderSplash(2732, 2732, denimDark, white))

	fmt.Println("Brand assets generated successfully.")
}

func savePNG(path string, img image.Image) {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	fmt.Printf("Generated %s\n", path)
}

func renderSolid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}

func renderIcon(size int, bg, fg color.RGBA, fillBackground bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	if fillBackground && bg.A > 0 {
		draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	}

	scale := float64(size) / 48.0
	pad := scale * 4.0
	effectiveScale := (float64(size) - 2*pad) / 48.0

	// Draw Tag
	drawTagSymbol(img, pad, pad, effectiveScale, fg)
	return img
}

func renderSplash(w, h int, bg, fg color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	iconSize := int(float64(h) * 0.25)
	if iconSize < 300 {
		iconSize = 300
	}
	offsetX := float64(w-iconSize) / 2.0
	offsetY := float64(h-iconSize)/2.0 - float64(h)*0.03
	scale := float64(iconSize) / 48.0

	drawTagSymbol(img, offsetX, offsetY, scale, fg)
	return img
}

func renderOGImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{denimPrimary}, image.Point{}, draw.Src)

	// Draw left tag
	tagSize := 320
	scale := float64(tagSize) / 48.0
	offsetX := 120.0
	offsetY := float64(h-tagSize) / 2.0
	drawTagSymbol(img, offsetX, offsetY, scale, white)

	return img
}

func drawTagSymbol(img *image.RGBA, ox, oy, scale float64, fg color.RGBA) {
	strokeW := 3.2 * scale

	// Tag outer border points: (9,6) -> (31,6) -> (40,15) -> (40,42) -> (9,42) -> (9,6)
	p1x, p1y := ox+9*scale, oy+6*scale
	p2x, p2y := ox+31*scale, oy+6*scale
	p3x, p3y := ox+40*scale, oy+15*scale
	p4x, p4y := ox+40*scale, oy+42*scale
	p5x, p5y := ox+9*scale, oy+42*scale

	drawLine(img, p1x, p1y, p2x, p2y, strokeW, fg)
	drawLine(img, p2x, p2y, p3x, p3y, strokeW, fg)
	drawLine(img, p3x, p3y, p4x, p4y, strokeW, fg)
	drawLine(img, p4x, p4y, p5x, p5y, strokeW, fg)
	drawLine(img, p5x, p5y, p1x, p1y, strokeW, fg)

	// Hole circle cx=17, cy=14, r=3.2, stroke=2.6
	drawCircleHollow(img, ox+17*scale, oy+14*scale, 3.2*scale, 2.6*scale, fg)

	// 6 stitches: x=12, 15.9, 19.8, 23.7, 27.6, 31.5; y=30; w=2.6, h=2.8
	stitchX := []float64{12, 15.9, 19.8, 23.7, 27.6, 31.5}
	for _, sx := range stitchX {
		drawRectFilled(img, ox+sx*scale, oy+30*scale, 2.6*scale, 2.8*scale, fg)
	}
	// 7th stitch: circle cx=36, cy=31.4, r=2.1
	drawCircleFilled(img, ox+36*scale, oy+31.4*scale, 2.1*scale, fg)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1, width float64, col color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	dist := math.Hypot(dx, dy)
	if dist == 0 {
		return
	}
	steps := int(dist * 2.5)
	if steps < 1 {
		steps = 1
	}
	radius := width / 2.0
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		cx := x0 + t*dx
		cy := y0 + t*dy
		drawCircleFilled(img, cx, cy, radius, col)
	}
}

func drawCircleFilled(img *image.RGBA, cx, cy, r float64, col color.RGBA) {
	bounds := img.Bounds()
	minX := int(math.Floor(cx - r))
	maxX := int(math.Ceil(cx + r))
	minY := int(math.Floor(cy - r))
	maxY := int(math.Ceil(cy + r))

	rSq := r * r
	for y := minY; y <= maxY; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		for x := minX; x <= maxX; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			distSq := (float64(x)+0.5-cx)*(float64(x)+0.5-cx) + (float64(y)+0.5-cy)*(float64(y)+0.5-cy)
			if distSq <= rSq {
				img.SetRGBA(x, y, col)
			}
		}
	}
}

func drawCircleHollow(img *image.RGBA, cx, cy, r, strokeW float64, col color.RGBA) {
	innerR := r - strokeW/2.0
	outerR := r + strokeW/2.0
	if innerR < 0 {
		innerR = 0
	}
	bounds := img.Bounds()
	minX := int(math.Floor(cx - outerR))
	maxX := int(math.Ceil(cx + outerR))
	minY := int(math.Floor(cy - outerR))
	maxY := int(math.Ceil(cy + outerR))

	inSq := innerR * innerR
	outSq := outerR * outerR
	for y := minY; y <= maxY; y++ {
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		for x := minX; x <= maxX; x++ {
			if x < bounds.Min.X || x >= bounds.Max.X {
				continue
			}
			distSq := (float64(x)+0.5-cx)*(float64(x)+0.5-cx) + (float64(y)+0.5-cy)*(float64(y)+0.5-cy)
			if distSq >= inSq && distSq <= outSq {
				img.SetRGBA(x, y, col)
			}
		}
	}
}

func drawRectFilled(img *image.RGBA, x, y, w, h float64, col color.RGBA) {
	bounds := img.Bounds()
	minX := int(math.Floor(x))
	maxX := int(math.Ceil(x + w))
	minY := int(math.Floor(y))
	maxY := int(math.Ceil(y + h))

	for cy := minY; cy <= maxY; cy++ {
		if cy < bounds.Min.Y || cy >= bounds.Max.Y {
			continue
		}
		for cx := minX; cx <= maxX; cx++ {
			if cx < bounds.Min.X || cx >= bounds.Max.X {
				continue
			}
			img.SetRGBA(cx, cy, col)
		}
	}
}
