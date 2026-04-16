package cover

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

const (
	width  = 1600
	height = 2400
	margin = 60  // outer border margin
	border = 6   // border thickness
)

const splitY = height * 2 / 5 // boundary between white (top) and gray (bottom)

var (
	clrBg     = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	clrGray   = color.RGBA{R: 70, G: 70, B: 70, A: 255}
	clrBorder = color.RGBA{R: 40, G: 40, B: 40, A: 255}
	clrTitle  = color.RGBA{R: 20, G: 20, B: 20, A: 255}
	clrAuthor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

func Generate(title, author string, fontBytes []byte) ([]byte, error) {
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// White top section
	draw.Draw(img, img.Bounds(), &image.Uniform{clrBg}, image.Point{}, draw.Src)
	// Dark gray bottom section (inside outer border)
	draw.Draw(img, image.Rect(margin, splitY, width-margin, height-margin), &image.Uniform{clrGray}, image.Point{}, draw.Src)

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(f)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetHinting(font.HintingFull)

	// Title in top white section (center of top 2/5)
	parts := splitTitle(title)
	topCenter := splitY / 2
	if len(parts) == 1 {
		if err := drawCenteredText(c, img, f, parts[0], 100, topCenter, clrTitle); err != nil {
			return nil, err
		}
	} else {
		if err := drawCenteredText(c, img, f, parts[0], 100, topCenter-70, clrTitle); err != nil {
			return nil, err
		}
		if err := drawCenteredText(c, img, f, parts[1], 64, topCenter+60, clrTitle); err != nil {
			return nil, err
		}
	}

	// Author in bottom gray section (center of bottom 3/5)
	bottomCenter := splitY + (height-splitY)/2
	if err := drawCenteredText(c, img, f, author, 72, bottomCenter, clrAuthor); err != nil {
		return nil, err
	}

	// Borders drawn last so they appear above fill areas
	drawRect(img, margin, margin, width-margin, height-margin, border, clrBorder)
	inner := margin + 20
	drawRect(img, inner, inner, width-inner, height-inner, 2, clrBorder)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func splitTitle(title string) []string {
	// Split on full-width space used as separator between title and subtitle
	if idx := strings.IndexRune(title, '　'); idx >= 0 {
		return []string{title[:idx], title[idx+3:]} // 　 is 3 bytes in UTF-8
	}
	return []string{title}
}

// drawRect draws a hollow rectangle border.
func drawRect(img *image.RGBA, x0, y0, x1, y1, thick int, col color.RGBA) {
	u := &image.Uniform{col}
	draw.Draw(img, image.Rect(x0, y0, x1, y0+thick), u, image.Point{}, draw.Src) // top
	draw.Draw(img, image.Rect(x0, y1-thick, x1, y1), u, image.Point{}, draw.Src) // bottom
	draw.Draw(img, image.Rect(x0, y0, x0+thick, y1), u, image.Point{}, draw.Src) // left
	draw.Draw(img, image.Rect(x1-thick, y0, x1, y1), u, image.Point{}, draw.Src) // right
}

func drawCenteredText(c *freetype.Context, img *image.RGBA, f *truetype.Font, text string, size float64, y int, col color.RGBA) error {
	c.SetFontSize(size)
	c.SetSrc(&image.Uniform{col})

	opts := &truetype.Options{Size: size, DPI: 72}
	face := truetype.NewFace(f, opts)
	defer face.Close()

	textWidth := 0
	for _, r := range text {
		adv, _ := face.GlyphAdvance(r)
		textWidth += int(adv >> 6)
	}

	x := (width - textWidth) / 2
	if x < margin+30 {
		x = margin + 30
	}

	_, err := c.DrawString(text, freetype.Pt(x, y))
	return err
}
