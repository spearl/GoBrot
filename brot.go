package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"os"

	"github.com/lucasb-eyer/go-colorful"
)

const supersample = true

const (
	iterations             = 5000
	width, height          = 1024, 768
	xmin, ymin, xmax, ymax = 0.278587, -0.012560, 0.285413, -0.007440
	xdelta                 = (float64(xmax-xmin) / width) / 4
	ydelta                 = (float64(ymax-ymin) / height) / 4
)

type GradientTable []struct {
	Col colorful.Color
	Pos float64
}

func (self GradientTable) GetInterpolatedColorFor(t float64) colorful.Color {
	for i := 0; i < len(self)-1; i++ {
		c1 := self[i]
		c2 := self[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Time to blend!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendHcl(c2.Col, t).Clamped()
		}
	}

	// Nothing found means return the last gradient keypoint
	return self[len(self)-1].Col
}

func ParseHex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("ParseHex: " + err.Error())
	}
	return c
}

func mandelbrot(z complex128) (n uint16, v complex128) {
	for ; n < iterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			break
		}
	}
	return
}

func main() {
	keypoints := GradientTable{
		{ParseHex("#9e0142"), 0.0},
		{ParseHex("#d53e4f"), 0.1},
		{ParseHex("#f46d43"), 0.2},
		{ParseHex("#fdae61"), 0.3},
		{ParseHex("#fee090"), 0.4},
		{ParseHex("#ffffbf"), 0.5},
		{ParseHex("#e6f598"), 0.6},
		{ParseHex("#abdda4"), 0.7},
		{ParseHex("#66c2a5"), 0.8},
		{ParseHex("#3288bd"), 0.9},
		{ParseHex("#5e4fa2"), 1.0},
	}

	histogram := make([]int, iterations)

	offX := []float64{-xdelta, xdelta}
	offY := []float64{-ydelta, ydelta}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	vimg := [height][width][]float64{}

	fmt.Fprintln(os.Stderr, "Calculating...")

	for py := 0; py < height; py++ {
		// y is flipped since image Y axis extends down
		y := float64(py)/height*(ymin-ymax) + ymax
		for px := 0; px < width; px++ {
			x := float64(px)/width*(xmax-xmin) + xmin
			vpixels := make([]float64, 0, 4)
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					var nfloat float64
					z := complex(x+offX[i], y+offY[j])
					n, zn := mandelbrot(z)

					if n < iterations {
						histogram[n]++
						nu := math.Log(math.Log(cmplx.Abs(zn))/math.Log(2)) / math.Log(2)
						nfloat = float64(n) + 1 - nu
					} else {
						nfloat = math.Inf(+1)
					}
					vpixels = append(vpixels, nfloat)
				}
			}
			// Store vpixel slice for pixel
			vimg[py][px] = vpixels

			// Supersample
			// vpixels := make([]color.Color, 0, 4)
			// for i := 0; i < 2; i++ {
			// 	for j := 0; j < 2; j++ {
			// 		z := complex(x+offX[i], y+offY[j])
			// 		n := mandelbrot(z)
			// 		if n < iterations {
			// 			vpixels = append(vpixels, keypoints.GetInterpolatedColorFor(float64(n)/iterations))
			// 		} else {
			// 			vpixels = append(vpixels, color.Black)
			// 		}
			// 	}
			// }
			// img.Set(px, py, interpolate(vpixels))
			// var nfloat float64
			// z := complex(x, y)
			// n, zn := mandelbrot(z)
			// if n < iterations {
			// 	histogram[n]++
			// 	nu := math.Log(math.Log(cmplx.Abs(zn))/math.Log(2)) / math.Log(2)
			// 	nfloat = float64(n) + 1 - nu
			// 	// if nfloat-float64(n) < 0 {
			// 	// 	fmt.Fprintln(os.Stderr, nfloat)
			// 	// 	fmt.Fprintln(os.Stderr, n)
			// 	// }
			// } else {
			// 	nfloat = math.Inf(+1)
			// }
			// vimg[py][px] = nfloat
		}
	}

	total := 0
	for _, count := range histogram {
		total += count
	}

	fmt.Fprintln(os.Stderr, "Starting render...")

	// Rendering pass
	for py := 0; py < height; py++ {
		for px := 0; px < width; px++ {
			vpixels := vimg[py][px]
			vcolors := make([]colorful.Color, 4)
			for i, n := range vpixels {
				if math.IsInf(n, +1) {
					vcolors[i] = colorful.MakeColor(color.Black)
				} else {
					hue := 0.0
					for i := uint16(0); i < uint16(n); i++ {
						hue += float64(histogram[i]) / float64(total)
					}
					color1 := keypoints.GetInterpolatedColorFor(hue)

					hue = 0.0
					for i := uint16(0); i < (uint16(n) + 1); i++ {
						hue += float64(histogram[i]) / float64(total)
					}
					color2 := keypoints.GetInterpolatedColorFor(hue)

					_, frac := math.Modf(n)
					colorFinal := color1.BlendHcl(color2, frac).Clamped()
					vcolors[i] = colorFinal
				}

				// n := vimg[py][px]
				// if math.IsInf(n, +1) {
				// 	img.Set(px, py, color.Black)
				// } else {
				// 	hue := 0.0
				// 	for i := uint16(0); i < uint16(n); i++ {
				// 		hue += float64(histogram[i]) / float64(total)
				// 	}
				// 	color1 := keypoints.GetInterpolatedColorFor(hue)

				// 	hue = 0.0
				// 	for i := uint16(0); i < (uint16(n) + 1); i++ {
				// 		hue += float64(histogram[i]) / float64(total)
				// 	}
				// 	color2 := keypoints.GetInterpolatedColorFor(hue)

				// 	_, frac := math.Modf(n)
				// 	colorFinal := color1.BlendHcl(color2, frac).Clamped()
				// 	img.Set(px, py, colorFinal)

				// img.Set(px, py, keypoints.GetInterpolatedColorFor(n/float64(iterations)))
			}
			// blend together vcolors
			img.Set(px, py, vcolors[0].BlendHcl(vcolors[1], 0.5).Clamped().BlendHcl(vcolors[2].BlendHcl(vcolors[3], 0.5).Clamped(), 0.5).Clamped())
		}
	}

	png.Encode(os.Stdout, img)
}

func interpolate(colors []color.Color) color.Color {
	var r, g, b, a uint16
	n := len(colors)
	for _, c := range colors {
		r_, g_, b_, a_ := c.RGBA()
		r += uint16(r_ / uint32(n))
		g += uint16(g_ / uint32(n))
		b += uint16(b_ / uint32(n))
		a += uint16(a_ / uint32(n))
	}
	return color.RGBA64{r, g, b, a}
}
